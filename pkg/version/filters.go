package version

import (
	"strings"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/version/semver"
)

func isSBOMAttestationOrSig(tag string) bool {
	return strings.HasSuffix(tag, ".att") ||
		strings.HasSuffix(tag, ".sig") ||
		strings.HasSuffix(tag, ".sbom")
}

// Used when filtering Tags as a SemVer
func shouldSkipTag(opts *api.Options, v *semver.SemVer) bool {
	// Handle Regex matching
	if opts.RegexMatcher != nil {
		return !opts.RegexMatcher.MatchString(v.String())
	}

	// Handle metadata and version pinning
	return (!opts.UseMetaData && v.HasMetaData()) ||
		(opts.PinMajor != nil && *opts.PinMajor != v.Major()) ||
		(opts.PinMinor != nil && *opts.PinMinor != v.Minor()) ||
		(opts.PinPatch != nil && *opts.PinPatch != v.Patch())
}

// Used when filtering SHA Tags
func shouldSkipSHA(opts *api.Options, sha string) bool {
	// Filter out Sbom and Attestation/Signatures
	if isSBOMAttestationOrSig(sha) {
		return true
	}

	// Allow for Regex Filtering
	if opts != nil && opts.RegexMatcher != nil {
		return !opts.RegexMatcher.MatchString(sha)
	}

	return false
}

// isBetterSemVer compares two semantic version numbers and
// associated image tags to determine if one is considered better than the other.
func isBetterSemVer(_ *api.Options, latestV, v *semver.SemVer, latestImageTag, currentImageTag *api.ImageTag) bool {
	// No latest version set yet
	if latestV == nil {
		return true
	}

	// If the current version is greater than the latest
	if latestV.LessThan(v) {
		return true
	}

	// If the versions are equal, prefer the one with a later timestamp
	if latestV.Equal(v) && currentImageTag.Timestamp.After(latestImageTag.Timestamp) {
		return true
	}

	return false
}
