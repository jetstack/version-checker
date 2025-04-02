package api

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

	//ResolveSHAToTagsKey is used to resolve image sha256 to corresponding tags
	ResolveSHAToTagsKey = "resolve-sha-to-tags.version-checker.io"

	// MatchRegexAnnotationKey will enforce that tags that are looked up must
	// match this regex. UseMetaDataAnnotationKey is not required when this is
	// set. All other options are ignored when this is set.
	MatchRegexAnnotationKey = "match-regex.version-checker.io"

	// UseMetaDataAnnotationKey is defined as a tag containing anything after the
	// patch digit.
	// e.g. v1.0.1-gke.3 v1.0.1-alpha.0, v1.2.3.4...
	UseMetaDataAnnotationKey = "use-metadata.version-checker.io"

	// PinMajorAnnotationKey will pin the major version to check.
	PinMajorAnnotationKey = "pin-major.version-checker.io"

	// PinMinorAnnotationKey will pin the minor version to check.
	PinMinorAnnotationKey = "pin-minor.version-checker.io"

	// PinPatchAnnotationKey will pin the patch version to check.
	PinPatchAnnotationKey = "pin-patch.version-checker.io"
)
