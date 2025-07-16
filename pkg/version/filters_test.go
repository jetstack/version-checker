package version

import (
	"regexp"
	"testing"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/version/semver"
	"github.com/stretchr/testify/assert"
)

func TestShouldSkipTag(t *testing.T) {
	tests := map[string]struct {
		opts       *api.Options
		semVer     *semver.SemVer
		expSkipped bool
	}{
		"skip tag with metadata when UseMetaData is false": {
			opts: &api.Options{
				UseMetaData: false,
			},
			semVer:     semver.Parse("1.2.3-meta"),
			expSkipped: true,
		},
		"do not skip tag with metadata when UseMetaData is true": {
			opts: &api.Options{
				UseMetaData: true,
			},
			semVer:     semver.Parse("1.2.3-meta"),
			expSkipped: false,
		},
		"skip tag when major version does not match PinMajor": {
			opts: &api.Options{
				PinMajor: int64p(2),
			},
			semVer:     semver.Parse("1.2.3"),
			expSkipped: true,
		},
		"do not skip tag when major version matches PinMajor": {
			opts: &api.Options{
				PinMajor: int64p(1),
			},
			semVer:     semver.Parse("1.2.3"),
			expSkipped: false,
		},
		"skip tag when minor version does not match PinMinor": {
			opts: &api.Options{
				PinMinor: int64p(3),
			},
			semVer:     semver.Parse("1.2.3"),
			expSkipped: true,
		},
		"do not skip tag when minor version matches PinMinor": {
			opts: &api.Options{
				PinMinor: int64p(2),
			},
			semVer:     semver.Parse("1.2.3"),
			expSkipped: false,
		},
		"skip tag when patch version does not match PinPatch": {
			opts: &api.Options{
				PinPatch: int64p(4),
			},
			semVer:     semver.Parse("1.2.3"),
			expSkipped: true,
		},
		"do not skip tag when patch version matches PinPatch": {
			opts: &api.Options{
				PinPatch: int64p(3),
			},
			semVer:     semver.Parse("1.2.3"),
			expSkipped: false,
		},
		"skip tag when RegexMatcher does not match": {
			opts: &api.Options{
				RegexMatcher: regexp.MustCompile(`^v2\..*`),
			},
			semVer:     semver.Parse("1.2.3"),
			expSkipped: true,
		},
		"do not skip tag when RegexMatcher matches": {
			opts: &api.Options{
				RegexMatcher: regexp.MustCompile(`^v1\..*`),
			},
			semVer:     semver.Parse("1.2.3"),
			expSkipped: false,
		},
		"do not skip tag when no options are set": {
			opts:       &api.Options{},
			semVer:     semver.Parse("1.2.3"),
			expSkipped: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			skipped := shouldSkipTag(test.opts, test.semVer)
			assert.Equal(t, test.expSkipped, skipped)
		})
	}
}

func int64p(i int64) *int64 {
	return &i
}
