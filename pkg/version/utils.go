package version

import (
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/version/semver"
)

// latestSemver will return the latest ImageTag based on the given options
// restriction, using semver. This should not be used is UseSHA has been
// enabled.
// TODO: add tests..
func latestSemver(tags []api.ImageTag, opts *api.Options) (*api.ImageTag, error) {
	var (
		latestImageTag *api.ImageTag
		latestV        *semver.SemVer
	)

	for i := range tags {
		// forcing it be the specific arch and os (defaults to true, if not set)
		if !osArchMatch(tags[i], opts) {
			continue
		}

		v := semver.Parse(tags[i].Tag)

		// If regex enabled continue here.
		// If we match, and is less than, update latest.
		if opts.RegexMatcher != nil {
			if opts.RegexMatcher.MatchString(tags[i].Tag) &&
				(latestV == nil || latestV.LessThan(v)) {
				latestV = v
				latestImageTag = &tags[i]
			}

			continue
		}

		// If we have declared we wont use metadata but version has it, continue.
		if !opts.UseMetaData && v.HasMetaData() {
			continue
		}

		if opts.PinMajor != nil && *opts.PinMajor != v.Major() {
			continue
		}
		if opts.PinMinor != nil && *opts.PinMinor != v.Minor() {
			continue
		}
		if opts.PinPatch != nil && *opts.PinPatch != v.Patch() {
			continue
		}

		// If no latest yet set
		if latestV == nil ||
			// If the latest set is less than
			latestV.LessThan(v) ||
			// If the latest is the same tag, but smaller timestamp
			(latestV.Equal(v) && tags[i].Timestamp.After(latestImageTag.Timestamp)) {
			latestV = v
			latestImageTag = &tags[i]
		}
	}

	return latestImageTag, nil
}

// latestSHA will return the latest ImageTag based on image timestamps.
func latestSHA(tags []api.ImageTag, opts *api.Options) (*api.ImageTag, error) {
	var latestTag *api.ImageTag

	for i := range tags {
		// forcing it be the specific arch and os (defalts to true, if not set)
		if !osArchMatch(tags[i], opts) {
			continue
		}
		if latestTag == nil || tags[i].Timestamp.After(latestTag.Timestamp) {
			latestTag = &tags[i]
		}
	}

	return latestTag, nil
}

func osArchMatch(tag api.ImageTag, opts *api.Options) bool {
	if opts.OS == nil || opts.Architecture == nil {
		return true
	}
	return tag.OS == *opts.OS && tag.Architecture == *opts.Architecture
}
