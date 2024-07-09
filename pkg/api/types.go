package api

import (
	"regexp"
	"time"
)

const (
	// EnableAnnotationKey is used for enabling or disabling version-checker for
	// a given container.
	EnableAnnotationKey = "enable.version-checker.io"

	// OverrideURLAnnotationKey is used to override the lookup URL. Useful when
	// mirroring images.
	OverrideURLAnnotationKey = "override-url.version-checker.io"

	// UseSHAAnnotationKey is used to comparing the SHA digests of images. This
	// is silently set to true if the container image using using the SHA digest
	// as its tag.
	UseSHAAnnotationKey = "use-sha.version-checker.io"

	// MatchRegexAnnotationKey will enforce that tags that are looked up must
	// match this regex. UseMetaDataAnnotationKey is not required when this is
	// set. All other options are ignored when this is set.
	MatchRegexAnnotationKey = "match-regex.version-checker.io"

	// UseMetaDataAnnotationKey is defined as a tag containing anything after the
	// patch digit.
	// e.g. v1.0.1-gke.3 v1.0.1-alpha.0, v1.2.3.4
	UseMetaDataAnnotationKey = "use-metadata.version-checker.io"

	// PinMajorAnnotationKey will pin the major version to check.
	PinMajorAnnotationKey = "pin-major.version-checker.io"

	// PinMinorAnnotationKey will pin the minor version to check.
	PinMinorAnnotationKey = "pin-minor.version-checker.io"

	// PinPatchAnnotationKey will pin the patch version to check.
	PinPatchAnnotationKey = "pin-patch.version-checker.io"
)

// Options is used to describe what restrictions should be used for determining
// the latest image.
type Options struct {
	OverrideURL *string `json:"override-url,omitempty"`

	// UseSHA cannot be used with any other options
	UseSHA bool `json:"use-sha,omitempty"`

	MatchRegex *string `json:"match-regex,omitempty"`

	// UseMetaData defines whether tags with '-alpha', '-debian.0' etc. is
	// permissible.
	UseMetaData bool `json:"use-metadata,omitempty"`

	PinMajor *int64 `json:"pin-major,omitempty"`
	PinMinor *int64 `json:"pin-minor,omitempty"`
	PinPatch *int64 `json:"pin-patch,omitempty"`

	RegexMatcher *regexp.Regexp `json:"-"`

	// Architecture and OS to search for
	Architecture *Architecture `json:"pin-architecture,omitempty"`
	OS           *OS           `json:"pin-os,omitempty"`
}

// ImageTag describes a container image tag.
type ImageTag struct {
	Tag          string       `json:"tag"`
	SHA          string       `json:"sha"`
	Timestamp    time.Time    `json:"timestamp"`
	OS           OS           `json:"os,omitempty"`
	Architecture Architecture `json:"architecture,omitempty"`
	Children     []ImageTag   `json:"children,omitempty"`
}

type OS string
type Architecture string

func (i *ImageTag) HasChildren() bool {
	return len(i.Children) > 0
}

func (i *ImageTag) HasArchOS(arch Architecture, os OS) bool {
	if i.matchArchOS(arch, os) {
		return true
	}

	for _, c := range i.Children {
		if c.matchArchOS(arch, os) {
			return true
		}
	}

	return true
}

func (i *ImageTag) MatchSHA(sha string) bool {
	if i.SHA == sha {
		return true
	}

	for _, c := range i.Children {
		if c.MatchSHA(sha) {
			return true
		}
	}

	return false
}

func (i *ImageTag) matchArchOS(arch Architecture, os OS) bool {
	if i.OS != "" && i.OS != os {
		return false
	}

	if i.Architecture != "" && i.Architecture != arch {
		return false
	}

	return true
}
