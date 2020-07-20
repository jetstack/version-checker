package api

import (
	"context"
	"regexp"
	"time"

	"github.com/masterminds/semver"
)

const (
	EnableAnnotationKey = "version-checker.io/enable"

	UseSHAAnnotationKey        = "use-sha.version-checker.io"
	UsePreReleaseAnnotationKey = "use-prerelease.version-checker.io"
	MatchRegexAnnotationKey    = "match-regex.version-checker.io"

	PinMajorAnnotationKey = "pin-major.version-checker.io"
	PinMinorAnnotationKey = "pin-minor.version-checker.io"
	PinPatchAnnotationKey = "pin-patch.version-checker.io"

	// TODO: set OS + arch options
)

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

type ImageTag struct {
	Tag          string    `json:"tag"`
	SHA          string    `json:"sha"`
	Timestamp    time.Time `json:"timestamp"`
	Architecture string    `json:"architecture,omitempty"`
	OS           string    `json:"os,omitempty"`

	SemVer *semver.Version
}

type ImageClient interface {
	// IsClient will return true if this client is appropriate for the given
	// image URL.
	IsClient(imageURL string) bool

	// Tags will return the available tags for the given image URL at the remote
	// repository.
	Tags(ctx context.Context, imageURL string) ([]ImageTag, error)
}
