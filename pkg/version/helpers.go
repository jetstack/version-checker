package version

import (
	"fmt"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/version/semver"
)

// latestSemver will return the latest ImageTag based on the given options
// restriction, using semver. This should not be used if UseSHA has been
// enabled.
func latestSemver(opts *api.Options, tags []api.ImageTag) (*api.ImageTag, error) {
	var (
		latestImageTag *api.ImageTag
		latestV        *semver.SemVer
	)

	for i := range tags {
		v := semver.Parse(tags[i].Tag)

		if shouldSkipTag(opts, v) {
			continue
		}

		if isBetterSemVer(opts, latestV, v, latestImageTag, &tags[i]) {
			latestV = v
			latestImageTag = &tags[i]
		}
	}

	if latestImageTag == nil {
		return nil, fmt.Errorf("no suitable version found")
	}

	return latestImageTag, nil
}

// latestSHA will return the latest ImageTag based on image timestamps.
func latestSHA(opts *api.Options, tags []api.ImageTag) (*api.ImageTag, error) {
	var latestTag *api.ImageTag

	for i := range tags {
		// Filter out SBOM and Attestation/Sig's...
		if shouldSkipSHA(opts, tags[i].Tag) {
			continue
		}

		if latestTag == nil || tags[i].Timestamp.After(latestTag.Timestamp) {
			latestTag = &tags[i]
		}
	}

	return latestTag, nil
}
