package options

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jetstack/version-checker/pkg/api"
)

func TestBuild(t *testing.T) {
	tests := map[string]struct {
		containerName string
		annotations   map[string]string
		expOptions    *api.Options
		expErr        string
	}{
		"if annotations not using the same name, ignore": {
			containerName: "test-name",
			annotations: map[string]string{
				api.PinPatchAnnotationKey + "/test-name-foo":    "foo",
				api.PinMinorAnnotationKey + "/test-name-foo":    "foo",
				api.PinMajorAnnotationKey + "/test-name-foo":    "foo",
				api.UseSHAAnnotationKey + "/test-name-foo":      "foo",
				api.UseMetaDataAnnotationKey + "/test-name-foo": "foo",
				api.OverrideURLAnnotationKey + "/test-name-foo": "foo",
			},
			expOptions: new(api.Options),
			expErr:     "",
		},
		"should not be able to set patch pin without major or minor pins": {
			containerName: "test-name",
			annotations: map[string]string{
				api.PinPatchAnnotationKey + "/test-name": "5",
			},
			expOptions: nil,
			expErr:     `unable to set "pin-patch.version-checker.io/test-name" without setting "pin-minor.version-checker.io/test-name" and "pin-major.version-checker.io/test-name"`,
		},
		"should not be able to set minor pin without major pin": {
			containerName: "test-name",
			annotations: map[string]string{
				api.PinMinorAnnotationKey + "/test-name": "5",
			},
			expOptions: nil,
			expErr:     `unable to set "pin-minor.version-checker.io/test-name" without setting "pin-major.version-checker.io/test-name"`,
		},
		"should not be able to set minor pin without major pin even with patch": {
			containerName: "test-name",
			annotations: map[string]string{
				api.PinPatchAnnotationKey + "/test-name": "5",
				api.PinMinorAnnotationKey + "/test-name": "5",
			},
			expOptions: nil,
			expErr:     `unable to set "pin-minor.version-checker.io/test-name" without setting "pin-major.version-checker.io/test-name", unable to set "pin-patch.version-checker.io/test-name" without setting "pin-minor.version-checker.io/test-name" and "pin-major.version-checker.io/test-name"`,
		},
		"cannot use sha with non sha options (regex)": {
			containerName: "test-name",
			annotations: map[string]string{
				api.MatchRegexAnnotationKey + "/test-name": "5",
				api.UseSHAAnnotationKey + "/test-name":     "true",
			},
			expOptions: nil,
			expErr:     `cannot define "use-sha.version-checker.io/test-name" with any semver options`,
		},
		"cannot use sha with non sha options (pins)": {
			containerName: "test-name",
			annotations: map[string]string{
				api.PinMajorAnnotationKey + "/test-name": "5",
				api.PinMinorAnnotationKey + "/test-name": "5",
				api.UseSHAAnnotationKey + "/test-name":   "true",
			},
			expOptions: nil,
			expErr:     `cannot define "use-sha.version-checker.io/test-name" with any semver options`,
		},
		"output options for pins and add metadata": {
			containerName: "test-name",
			annotations: map[string]string{
				api.PinMajorAnnotationKey + "/test-name":    "1",
				api.PinMinorAnnotationKey + "/test-name":    "2",
				api.PinPatchAnnotationKey + "/test-name":    "3",
				api.UseMetaDataAnnotationKey + "/test-name": "true",
			},
			expOptions: &api.Options{
				PinMajor:    int64p(1.0),
				PinMinor:    int64p(2.0),
				PinPatch:    int64p(3.0),
				UseMetaData: true,
			},
			expErr: "",
		},
		"output options for override url and regex": {
			containerName: "test-name",
			annotations: map[string]string{
				api.MatchRegexAnnotationKey + "/test-name":  `v1\.2\.1`,
				api.OverrideURLAnnotationKey + "/test-name": "foo.bar.io",
			},
			expOptions: &api.Options{
				MatchRegex:   stringp(`v1\.2\.1`),
				OverrideURL:  stringp("foo.bar.io"),
				RegexMatcher: regexp.MustCompile(`v1\.2\.1`),
			},
			expErr: "",
		},
		"output options for sha": {
			containerName: "test-name",
			annotations: map[string]string{
				api.UseSHAAnnotationKey + "/test-name": "true",
			},
			expOptions: &api.Options{
				UseSHA: true,
			},
			expErr: "",
		},
		"output options for resolve sha": {
			containerName: "test-name",
			annotations: map[string]string{
				api.ResolveSHAToTagsKey + "/test-name": "true",
			},
			expOptions: &api.Options{
				ResolveSHAToTags: true,
			},
			expErr: "",
		},
		"bool options that don't have 'true' and nothing": {
			containerName: "test-name",
			annotations: map[string]string{
				api.UseSHAAnnotationKey + "/test-name":      "false",
				api.UseMetaDataAnnotationKey + "/test-name": "foo",
			},
			expOptions: new(api.Options),
			expErr:     "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			options, err := New(test.annotations).Options(test.containerName)

			if len(test.expErr) > 0 {
				assert.Error(t, err)
				assert.Equal(t, test.expErr, err.Error())

			} else {
				require.NoError(t, err)
			}

			assert.Exactly(t, test.expOptions, options)
		})
	}
}

func TestIsEnabled(t *testing.T) {
	tests := map[string]struct {
		containerName string
		annotations   map[string]string
		defaultAll    bool
		expEnabled    bool
	}{
		"if no annotations set and default false, then false": {
			containerName: "test-name",
			defaultAll:    false,
			annotations:   nil,
			expEnabled:    false,
		},
		"if no annotations set and default true, then true": {
			containerName: "test-name",
			defaultAll:    true,
			annotations:   nil,
			expEnabled:    true,
		},
		"if annotations set but wrong name with default false, false": {
			containerName: "test-name",
			defaultAll:    false,
			annotations: map[string]string{
				api.EnableAnnotationKey + "/foo": "true",
			},
			expEnabled: false,
		},
		"if annotations set but wrong name with default true, true": {
			containerName: "test-name",
			defaultAll:    true,
			annotations: map[string]string{
				api.EnableAnnotationKey + "/foo": "true",
			},
			expEnabled: true,
		},
		"if annotations set but not true/false with default false, false": {
			containerName: "test-name",
			defaultAll:    false,
			annotations: map[string]string{
				api.EnableAnnotationKey + "/test-name": "foo",
			},
			expEnabled: false,
		},
		"if annotations set but not true/false with default true, true": {
			containerName: "test-name",
			defaultAll:    true,
			annotations: map[string]string{
				api.EnableAnnotationKey + "/test-name": "foo",
			},
			expEnabled: true,
		},
		"if annotations set true and default false, true": {
			containerName: "test-name",
			defaultAll:    false,
			annotations: map[string]string{
				api.EnableAnnotationKey + "/test-name": "true",
			},
			expEnabled: true,
		},
		"if annotations set true and default true, true": {
			containerName: "test-name",
			defaultAll:    true,
			annotations: map[string]string{
				api.EnableAnnotationKey + "/test-name": "true",
			},
			expEnabled: true,
		},
		"if annotations set false and default false, false": {
			containerName: "test-name",
			defaultAll:    false,
			annotations: map[string]string{
				api.EnableAnnotationKey + "/test-name": "false",
			},
			expEnabled: false,
		},
		"if annotations set false and default true, false": {
			containerName: "test-name",
			defaultAll:    true,
			annotations: map[string]string{
				api.EnableAnnotationKey + "/test-name": "false",
			},
			expEnabled: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			enabled := New(test.annotations).IsEnabled(test.defaultAll, test.containerName)
			if !reflect.DeepEqual(enabled, test.expEnabled) {
				t.Errorf("%s: unexpected enabled %v exp=%v got=%v",
					test.containerName, test.annotations, test.expEnabled, enabled)
			}
		})
	}
}

func int64p(i int64) *int64 {
	return &i
}

func stringp(s string) *string {
	return &s
}
