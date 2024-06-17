package version

import (
	"regexp"
	"testing"
	"time"

	"github.com/jetstack/version-checker/pkg/api"

	"github.com/stretchr/testify/assert"
)

// Helper function to parse time
func parseTime(t string) time.Time {
	parsedTime, _ := time.Parse(time.RFC3339, t)
	return parsedTime
}

func TestLatestSemver(t *testing.T) {
	// Ideal Set of Tags
	tags := []api.ImageTag{
		{Tag: "v1.0.0", Timestamp: parseTime("2023-06-01T00:00:00Z")},
		{Tag: "v1.1.0", Timestamp: parseTime("2023-06-02T00:00:00Z")},
		{Tag: "v1.1.1-alpha", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "v1.1.1", Timestamp: parseTime("2023-06-04T00:00:00Z")},
		{Tag: "v2.0.0", Timestamp: parseTime("2023-06-05T00:00:00Z")},
	}
	tagsNoPrefix := []api.ImageTag{
		{Tag: "1.0.0", Timestamp: parseTime("2023-06-01T00:00:00Z")},
		{Tag: "1.1.0", Timestamp: parseTime("2023-06-02T00:00:00Z")},
		{Tag: "1.1.1-alpha", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "1.1.1", Timestamp: parseTime("2023-06-04T00:00:00Z")},
		{Tag: "2.0.0", Timestamp: parseTime("2023-06-05T00:00:00Z")},
	}
	// Include More Alpha/Beta/RC
	alphaBetaTags := []api.ImageTag{
		{Tag: "v1.0.0", Timestamp: parseTime("2023-06-01T00:00:00Z")},
		{Tag: "v1.1.0", Timestamp: parseTime("2023-06-02T00:00:00Z")},
		{Tag: "v1.1.1-alpha", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "v1.1.1", Timestamp: parseTime("2023-06-04T00:00:00Z")},
		{Tag: "v2.0.0-alpha", Timestamp: parseTime("2023-06-05T00:00:00Z")},
		{Tag: "v2.0.0-beta", Timestamp: parseTime("2023-06-05T00:00:00Z")},
		{Tag: "v2.0.0-rc1", Timestamp: parseTime("2023-06-05T00:00:00Z")},
		{Tag: "v2.0.0-rc2", Timestamp: parseTime("2023-06-05T00:00:00Z")},
		{Tag: "v2.0.0", Timestamp: parseTime("2023-06-05T00:00:00Z")},
	}
	// Images that are all numerical
	nonSemVer := []api.ImageTag{
		{Tag: "20230601", Timestamp: parseTime("2023-06-01T00:00:00Z")},
		{Tag: "20230602", Timestamp: parseTime("2023-06-02T00:00:00Z")},
		{Tag: "20230603", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "20230604", Timestamp: parseTime("2023-06-04T00:00:00Z")},
		{Tag: "20230605", Timestamp: parseTime("2023-06-05T00:00:00Z")},
	}
	// This is to simulate an image that USED to SemVer but stopped
	stoppedSemVer := []api.ImageTag{
		{Tag: "v1.0.0", Timestamp: parseTime("2023-06-01T00:00:00Z")},
		{Tag: "v1.1.0", Timestamp: parseTime("2023-06-02T00:00:00Z")},
		{Tag: "20230603", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "202306031", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "20230604", Timestamp: parseTime("2023-06-04T00:00:00Z")},
		{Tag: "20230605", Timestamp: parseTime("2023-06-05T00:00:00Z")},
		{Tag: "20230606", Timestamp: parseTime("2023-06-06T00:00:00Z")},
	}
	// This is to simulate an image that USED to SemVer but stopped
	startedSemVer := []api.ImageTag{
		{Tag: "20230603", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "202306031", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "20230604", Timestamp: parseTime("2023-06-04T00:00:00Z")},
		{Tag: "20230605", Timestamp: parseTime("2023-06-05T00:00:00Z")},
		{Tag: "20230606", Timestamp: parseTime("2023-06-06T00:00:00Z")},
		{Tag: "v1.0.0", Timestamp: parseTime("2023-06-09T00:00:00Z")},
		{Tag: "v1.1.0", Timestamp: parseTime("2023-06-10T00:00:00Z")},
	}
	// Mixed Numerical and SemVer along with Older images pushed more recently
	badTags := []api.ImageTag{
		{Tag: "v1.0.0", Timestamp: parseTime("2023-06-01T00:00:00Z")},
		{Tag: "v1.1.0", Timestamp: parseTime("2023-06-02T00:00:00Z")},
		{Tag: "9999999", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "v1.1.1-alpha", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "v1.1.1", Timestamp: parseTime("2023-06-04T00:00:00Z")},
		{Tag: "v2.0.0", Timestamp: parseTime("2023-06-05T00:00:00Z")},
		{Tag: "v1.1.1", Timestamp: parseTime("2023-06-06T00:00:00Z")},
	}

	tests := []struct {
		name     string
		opts     *api.Options
		expected string
		tags     []api.ImageTag
	}{
		{
			name:     "No constraints",
			opts:     &api.Options{},
			expected: "v2.0.0",
		},
		{
			name: "Regex match v1.*",
			opts: &api.Options{
				RegexMatcher: regexp.MustCompile("v1.*"),
			},
			expected: "v1.1.1",
		},
		{
			name: "Pin major version 1",
			opts: &api.Options{
				PinMajor: intPtr(1),
			},
			expected: "v1.1.1",
		},
		{
			name: "Pin minor version 1.1",
			opts: &api.Options{
				PinMajor: intPtr(1),
				PinMinor: intPtr(1),
			},
			expected: "v1.1.1",
		},
		{
			name: "Pin patch version 1.1.1",
			opts: &api.Options{
				PinMajor: intPtr(1),
				PinMinor: intPtr(1),
				PinPatch: intPtr(1),
			},
			expected: "v1.1.1",
		},
		{
			name: "Exclude metadata",
			opts: &api.Options{
				UseMetaData: false,
			},
			expected: "v2.0.0",
		},
		{
			name: "Include metadata",
			opts: &api.Options{
				UseMetaData: true,
			},
			expected: "v2.0.0",
		},
		{
			name:     "NoPrefixed Tags",
			opts:     &api.Options{},
			tags:     tagsNoPrefix,
			expected: "2.0.0",
		},
		// Some Bad/Miss-behaving tags
		{
			name:     "Bad Tags",
			opts:     &api.Options{},
			tags:     badTags,
			expected: "v2.0.0",
		},
		// None SemVer tags
		{
			name:     "Non SemVer",
			opts:     &api.Options{},
			tags:     nonSemVer,
			expected: "20230605",
		},
		{
			name:     "Stopped SemVer",
			opts:     &api.Options{},
			tags:     stoppedSemVer,
			expected: "20230606",
		},
		{
			name:     "Started SemVer",
			opts:     &api.Options{},
			tags:     startedSemVer,
			expected: "v1.1.0",
		},
		{
			name:     "Alpha/Beta SemVer",
			opts:     &api.Options{},
			tags:     alphaBetaTags,
			expected: "v1.1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.tags) > 0 {
				tags = tt.tags
			}
			tag, err := latestSemver(tt.opts, tags)
			assert.NoError(t, err)
			assert.NotNil(t, tag)
			assert.Equal(t, tt.expected, tag.Tag)
		})
	}
}

func intPtr(i uint64) *uint64 {
	return &i
}
