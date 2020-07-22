package api

import (
	"regexp"
	"time"
)

const (
	EnableAnnotationKey = "enable.version-checker.io"

	UseSHAAnnotationKey        = "use-sha.version-checker.io"
	UsePreReleaseAnnotationKey = "use-prerelease.version-checker.io"
	MatchRegexAnnotationKey    = "match-regex.version-checker.io"

	PinMajorAnnotationKey = "pin-major.version-checker.io"
	PinMinorAnnotationKey = "pin-minor.version-checker.io"
	PinPatchAnnotationKey = "pin-patch.version-checker.io"

	// TODO: set OS + arch options
)

// Options is used to describe what restrictions should be used for determining
// the latest image.
type Options struct {
	// UseSHA cannot be used with any other options
	UseSHA bool `json:"use-sha,omitempty"`

	UsePreRelease bool    `json:"use-prerelease,omitempty"`
	MatchRegex    *string `json:"match-regex,omitempty"`

	PinMajor *int64 `json:"pin-major,omitempty"`
	PinMinor *int64 `json:"pin-minor,omitempty"`
	PinPatch *int64 `json:"pin-patch,omitempty"`

	RegexMatcher *regexp.Regexp
}

// ImageTag describes a container image tag.
type ImageTag struct {
	Tag          string    `json:"tag"`
	SHA          string    `json:"sha"`
	Timestamp    time.Time `json:"timestamp"`
	Architecture string    `json:"architecture,omitempty"`
	OS           string    `json:"os,omitempty"`
}
