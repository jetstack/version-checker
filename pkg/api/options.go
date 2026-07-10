package api

import "regexp"

// Options is used to describe what restrictions should be used for determining
// the latest image.
type Options struct {
	OverrideURL *string `json:"override-url,omitempty"`

	MatchRegex *string `json:"match-regex,omitempty"`

	PinMajor *int64 `json:"pin-major,omitempty"`
	PinMinor *int64 `json:"pin-minor,omitempty"`
	PinPatch *int64 `json:"pin-patch,omitempty"`

	RegexMatcher *regexp.Regexp `json:"-"`

	// UseSHA cannot be used with any other options
	UseSHA bool `json:"use-sha,omitempty"`
	// Resolve SHA to a TAG
	ResolveSHAToTags bool `json:"resolve-sha-to-tags,omitempty"`
	// Use GitHub releases as the source for latest GHCR versions.
	UseGitHubRelease bool `json:"use-github-release,omitempty"`

	// UseMetaData defines whether tags with '-alpha', '-debian.0' etc. is
	// permissible.
	UseMetaData bool `json:"use-metadata,omitempty"`
}
