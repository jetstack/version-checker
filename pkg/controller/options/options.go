package options

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jetstack/version-checker/pkg/api"
)

// Builder is a struct for building container search options.
type Builder struct {
	ans map[string]string
}

type optionsHandler func(name string, opts *api.Options, setNonSha *bool, errs *[]string) error

// New contructs a new Builder.
func New(annotations map[string]string) *Builder {
	return &Builder{
		ans: annotations,
	}
}

// Options will build the tag options based on pod annotations and container
// name.
func (b *Builder) Options(name string) (*api.Options, error) {
	var (
		opts      api.Options
		errs      []string
		setNonSha bool
	)

	// Define the handlers
	handlers := []optionsHandler{
		b.handleSHAOption,
		b.handleSHAToTagOption,
		b.handleMetadataOption,
		b.handleRegexOption,
		b.handlePinMajorOption,
		b.handlePinMinorOption,
		b.handlePinPatchOption,
		b.handleOverrideURLOption,
	}

	// Execute each handler
	for _, handler := range handlers {
		if err := handler(name, &opts, &setNonSha, &errs); err != nil {
			errs = append(errs, err.Error())
		}
	}

	// Ensure UseSHA is not used with other semver options
	if opts.UseSHA && setNonSha {
		errs = append(errs,
			fmt.Sprintf("cannot define %q with any semver options", b.index(name, api.UseSHAAnnotationKey)),
		)
	}

	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, ", "))
	}

	return &opts, nil
}
func (b *Builder) handleSHAOption(name string, opts *api.Options, setNonSha *bool, errs *[]string) error {
	if useSHA, ok := b.ans[b.index(name, api.UseSHAAnnotationKey)]; ok && useSHA == "true" {
		opts.UseSHA = true
	}
	return nil
}
func (b *Builder) handleSHAToTagOption(name string, opts *api.Options, setNonSha *bool, errs *[]string) error {
	if ResolveSHAToTags, ok := b.ans[b.index(name, api.ResolveSHAToTagsKey)]; ok && ResolveSHAToTags == "true" {
		opts.ResolveSHAToTags = true
	}
	return nil
}

func (b *Builder) handleMetadataOption(name string, opts *api.Options, setNonSha *bool, errs *[]string) error {
	if useMetaData, ok := b.ans[b.index(name, api.UseMetaDataAnnotationKey)]; ok && useMetaData == "true" {
		*setNonSha = true
		opts.UseMetaData = true
	}
	return nil
}

func (b *Builder) handleRegexOption(name string, opts *api.Options, setNonSha *bool, errs *[]string) error {
	if matchRegex, ok := b.ans[b.index(name, api.MatchRegexAnnotationKey)]; ok {
		*setNonSha = true
		opts.MatchRegex = &matchRegex

		regexMatcher, err := regexp.Compile(matchRegex)
		if err != nil {
			*errs = append(*errs, fmt.Sprintf("failed to compile regex at annotation %q: %s", api.MatchRegexAnnotationKey, err))
		} else {
			opts.RegexMatcher = regexMatcher
		}
	}
	return nil
}

func (b *Builder) handlePinMajorOption(name string, opts *api.Options, setNonSha *bool, errs *[]string) error {
	if pinMajor, ok := b.ans[b.index(name, api.PinMajorAnnotationKey)]; ok {
		*setNonSha = true
		ma, err := strconv.ParseInt(pinMajor, 10, 64)
		if err != nil {
			*errs = append(*errs, fmt.Sprintf("failed to parse %s: %s", b.index(name, api.PinMajorAnnotationKey), err))
		} else {
			opts.PinMajor = &ma
		}
	}
	return nil
}

func (b *Builder) handlePinMinorOption(name string, opts *api.Options, setNonSha *bool, errs *[]string) error {
	if pinMinor, ok := b.ans[b.index(name, api.PinMinorAnnotationKey)]; ok {
		*setNonSha = true
		if opts.PinMajor == nil {
			*errs = append(*errs, fmt.Sprintf("unable to set %q without setting %q", b.index(name, api.PinMinorAnnotationKey), b.index(name, api.PinMajorAnnotationKey)))
		} else {
			mi, err := strconv.ParseInt(pinMinor, 10, 64)
			if err != nil {
				*errs = append(*errs, fmt.Sprintf("failed to parse %s: %s", b.index(name, api.PinMinorAnnotationKey), err))
			} else {
				opts.PinMinor = &mi
			}
		}
	}
	return nil
}

func (b *Builder) handlePinPatchOption(name string, opts *api.Options, setNonSha *bool, errs *[]string) error {
	if pinPatch, ok := b.ans[b.index(name, api.PinPatchAnnotationKey)]; ok {
		*setNonSha = true
		if opts.PinMajor == nil || opts.PinMinor == nil {
			*errs = append(*errs, fmt.Sprintf("unable to set %q without setting %q and %q", b.index(name, api.PinPatchAnnotationKey), b.index(name, api.PinMinorAnnotationKey), b.index(name, api.PinMajorAnnotationKey)))
		} else {
			pa, err := strconv.ParseInt(pinPatch, 10, 64)
			if err != nil {
				*errs = append(*errs, fmt.Sprintf("failed to parse %s: %s", b.index(name, api.PinPatchAnnotationKey), err))
			} else {
				opts.PinPatch = &pa
			}
		}
	}
	return nil
}

func (b *Builder) handleOverrideURLOption(name string, opts *api.Options, setNonSha *bool, errs *[]string) error {
	if overrideURL, ok := b.ans[b.index(name, api.OverrideURLAnnotationKey)]; ok {
		opts.OverrideURL = &overrideURL
	}
	return nil
}

// IsEnabled will return whether the container has the enabled annotation set.
// Will fall back to default, if not set true/false.
func (b *Builder) IsEnabled(defaultEnabled bool, name string) bool {
	switch b.ans[b.index(name, api.EnableAnnotationKey)] {
	case "true":
		return true
	case "false":
		return false
	default:
		return defaultEnabled
	}
}

// index returns the annotation index give the API annotaion key.
func (b *Builder) index(containerName, annotationName string) string {
	return annotationName + "/" + containerName
}
