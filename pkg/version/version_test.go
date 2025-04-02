package version

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/cache"
	"github.com/jetstack/version-checker/pkg/client"
	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper function to parse time.
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
	sBomtags := []api.ImageTag{
		{Tag: "v1.1.0", Timestamp: parseTime("2023-06-02T00:00:00Z")},
		{Tag: "v1.1.1-alpha", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "v1.1.1", Timestamp: parseTime("2023-06-04T00:00:00Z")},
		{Tag: "sha256-ea5b51fc3bd6d014355e56de6bda7f8f42acf261a0a4645a2107ccbc438e12c3.sig", Timestamp: parseTime("2023-06-04T10:00:00Z")},
		{Tag: "v2.0.0", Timestamp: parseTime("2023-06-05T00:00:00Z")},
		{Tag: "sha256-b019b2a5c384570201ba592be195769e1848d3106c8c56c4bdad7d2ee34748e0.sig", Timestamp: parseTime("2023-06-07T00:00:00Z")},
		{Tag: "sha256-b019b2a5c384570201ba592be195769e1848d3106c8c56c4bdad7d2ee34748e0.att", Timestamp: parseTime("2023-06-07T10:00:00Z")},
		{Tag: "sha256-b019b2a5c384570201ba592be195769e1848d3106c8c56c4bdad7d2ee34748e0.sbom", Timestamp: parseTime("2023-06-07T221:00:00Z")},
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
		{Tag: "202306030", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "202306031", Timestamp: parseTime("2023-06-03T00:00:00Z")},
		{Tag: "202306040", Timestamp: parseTime("2023-06-04T00:00:00Z")},
		{Tag: "202306050", Timestamp: parseTime("2023-06-05T00:00:00Z")},
		{Tag: "202306060", Timestamp: parseTime("2023-06-06T00:00:00Z")},
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
			name:     "Strip Sig/SBOM/Attestations",
			opts:     &api.Options{},
			expected: "v2.0.0",
			tags:     sBomtags,
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
			name: "Bad Tags",
			opts: &api.Options{
				RegexMatcher: regexp.MustCompile(`^v(\d+)(\.\d+)?(\.\d+)?(.*)$`),
			},
			tags:     badTags,
			expected: "v2.0.0",
		},
		// None SemVer tags
		{
			name: "Non SemVer",
			opts: &api.Options{
				RegexMatcher: regexp.MustCompile(`^(\d+)`),
			},
			tags:     nonSemVer,
			expected: "20230605",
		},
		{
			name: "Stopped SemVer",
			opts: &api.Options{
				RegexMatcher: regexp.MustCompile(`^(\d+)`),
			},
			tags:     stoppedSemVer,
			expected: "202306060",
		},
		{
			name: "Started SemVer",
			opts: &api.Options{
				RegexMatcher: regexp.MustCompile(`^v(\d+)(\.\d+)?(\.\d+)?(.*)$`),
			},
			tags:     startedSemVer,
			expected: "v1.1.0",
		},
		{
			name: "Alpha/Beta SemVer",
			opts: &api.Options{
				UseMetaData: false,
			},
			tags:     alphaBetaTags,
			expected: "v1.1.1",
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

func TestLatestSHA(t *testing.T) {
	tests := []struct {
		name        string
		tags        []api.ImageTag
		options     *api.Options
		expectedSHA *string
	}{
		{
			name: "Single tag",
			tags: []api.ImageTag{
				{SHA: "sha1", Timestamp: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			expectedSHA: strPtr("sha1"),
		},
		{
			name: "Multiple tags, latest in the middle",
			tags: []api.ImageTag{
				{SHA: "sha1", Timestamp: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{SHA: "sha2", Timestamp: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{SHA: "sha3", Timestamp: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			expectedSHA: strPtr("sha2"),
		},
		{
			name: "Multiple tags, latest at the end",
			tags: []api.ImageTag{
				{SHA: "sha1", Timestamp: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{SHA: "sha2", Timestamp: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{SHA: "sha3", Timestamp: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			expectedSHA: strPtr("sha3"),
		},
		{
			name: "Multiple tags, including sig",
			tags: []api.ImageTag{
				{SHA: "sha1", Timestamp: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{SHA: "sha2", Timestamp: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{Tag: "sha3.sig", SHA: "sha3", Timestamp: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			expectedSHA: strPtr("sha2"),
		},
		{
			name: "Multiple tags, including att",
			tags: []api.ImageTag{
				{SHA: "sha1", Timestamp: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{SHA: "sha2", Timestamp: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{Tag: "sha3.att", SHA: "sha3", Timestamp: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{Tag: "sha3.att", SHA: "sha3", Timestamp: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			expectedSHA: strPtr("sha2"),
		},
		{
			name: "Multiple tags, including sbom",
			tags: []api.ImageTag{
				{SHA: "sha1", Timestamp: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{SHA: "sha2", Timestamp: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{Tag: "sha3.sbom", SHA: "sha3", Timestamp: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			expectedSHA: strPtr("sha2"),
		},
		{
			name: "Multiple tags, with Regex",
			tags: []api.ImageTag{
				{Tag: "1", SHA: "sha1", Timestamp: time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{Tag: "2", SHA: "sha2", Timestamp: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{Tag: "sha3.sbom", SHA: "sha3", Timestamp: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{Tag: "sha3.jsbfjsabfjs", SHA: "sha3", Timestamp: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			options:     &api.Options{RegexMatcher: regexp.MustCompile("^([0-9]+)$")},
			expectedSHA: strPtr("sha2"),
		},
		{
			name:        "No tags",
			tags:        []api.ImageTag{},
			expectedSHA: nil,
		},
		{
			name: "All tags with the same timestamp",
			tags: []api.ImageTag{
				{SHA: "sha1", Timestamp: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{SHA: "sha2", Timestamp: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)},
				{SHA: "sha3", Timestamp: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)},
			},
			expectedSHA: strPtr("sha1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := latestSHA(tt.options, tt.tags)
			if err != nil {
				t.Errorf("latestSHA() error = %v", err)
				return
			}

			if tt.expectedSHA == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got, "Should have received a version %s got nil", *tt.expectedSHA)
				assert.Equal(t, *tt.expectedSHA, got.SHA)
			}

		})
	}
}

func TestFetch(t *testing.T) {
	tests := []struct {
		name        string
		imageURL    string
		clientTags  []api.ImageTag
		clientError error
		expected    []api.ImageTag
		expectError bool
	}{
		{
			name:     "Successful fetch with tags",
			imageURL: "example.com/image",
			clientTags: []api.ImageTag{
				{Tag: "v1.0.0", Timestamp: parseTime("2023-06-01T00:00:00Z")},
				{Tag: "v1.1.0", Timestamp: parseTime("2023-06-02T00:00:00Z")},
			},
			expected:    []api.ImageTag{{Tag: "v1.0.0", Timestamp: parseTime("2023-06-01T00:00:00Z")}, {Tag: "v1.1.0", Timestamp: parseTime("2023-06-02T00:00:00Z")}},
			expectError: false,
		},
		{
			name:        "No tags found",
			imageURL:    "example.com/empty-image",
			clientTags:  []api.ImageTag{},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Client error",
			imageURL:    "example.com/error-image",
			clientError: fmt.Errorf("failed to fetch tags"),
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockClient{}
			mockClient.On("Tags", mock.Anything, tt.imageURL).Return(tt.clientTags, tt.clientError)

			v := &Version{
				log:    logrus.NewEntry(logrus.New()),
				client: mockClient,
			}

			result, err := v.Fetch(context.Background(), tt.imageURL, nil)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestLatestTagFromImage(t *testing.T) {
	tests := []struct {
		name        string
		imageURL    string
		clientTags  []api.ImageTag
		clientError error
		options     *api.Options
		expectedTag *api.ImageTag
		expectError bool
	}{
		{
			name:     "Latest SemVer tag",
			imageURL: "example.com/image",
			clientTags: []api.ImageTag{
				{Tag: "v1.0.0", Timestamp: parseTime("2023-06-01T00:00:00Z")},
				{Tag: "v1.1.0", Timestamp: parseTime("2023-06-02T00:00:00Z")},
				{Tag: "v2.0.0", Timestamp: parseTime("2023-06-03T00:00:00Z")},
			},
			options:     &api.Options{},
			expectedTag: &api.ImageTag{Tag: "v2.0.0", Timestamp: parseTime("2023-06-03T00:00:00Z")},
			expectError: false,
		},
		{
			name:     "Latest SHA tag",
			imageURL: "example.com/image",
			clientTags: []api.ImageTag{
				{SHA: "sha1", Timestamp: parseTime("2023-06-01T00:00:00Z")},
				{SHA: "sha2", Timestamp: parseTime("2023-06-02T00:00:00Z")},
				{SHA: "sha3", Timestamp: parseTime("2023-06-03T00:00:00Z")},
			},
			options:     &api.Options{UseSHA: true},
			expectedTag: &api.ImageTag{SHA: "sha3", Timestamp: parseTime("2023-06-03T00:00:00Z")},
			expectError: false,
		},
		{
			name:     "No matching tags",
			imageURL: "example.com/image",
			clientTags: []api.ImageTag{
				{Tag: "v1.0.0", Timestamp: parseTime("2023-06-01T00:00:00Z")},
			},
			options:     &api.Options{RegexMatcher: regexp.MustCompile("^v2.*")},
			expectedTag: nil,
			expectError: true,
		},
		{
			name:        "Client error",
			imageURL:    "example.com/error-image",
			clientError: fmt.Errorf("failed to fetch tags"),
			options:     &api.Options{},
			expectedTag: nil,
			expectError: true,
		},
		{
			name:        "No tags returned",
			imageURL:    "example.com/empty-image",
			clientTags:  []api.ImageTag{},
			options:     &api.Options{},
			expectedTag: nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockClient{}
			mockClient.On("Tags", mock.Anything, tt.imageURL).Return(tt.clientTags, tt.clientError)

			log := logrus.NewEntry(logrus.New())
			v := &Version{
				log:    log,
				client: mockClient,
			}
			v.imageCache = cache.New(log, time.Minute, v)

			tag, err := v.LatestTagFromImage(context.Background(), tt.imageURL, tt.options)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, tag)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, tag)
				assert.Equal(t, tt.expectedTag.Tag, tag.Tag)
				assert.Equal(t, tt.expectedTag.Timestamp, tag.Timestamp)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name          string
		log           *logrus.Entry
		client        client.ClientHandler
		cacheTimeout  time.Duration
		expectedCache time.Duration
	}{
		{
			name:          "Valid inputs",
			log:           logrus.NewEntry(logrus.New()),
			client:        &MockClient{},
			cacheTimeout:  time.Minute,
			expectedCache: time.Minute,
		},
		{
			name:          "Zero cache timeout",
			log:           logrus.NewEntry(logrus.New()),
			client:        &MockClient{},
			cacheTimeout:  0,
			expectedCache: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := New(tt.log, tt.client, tt.cacheTimeout)

			assert.NotNil(t, version)
			assert.Equal(t, tt.log.WithField("module", "version_getter"), version.log)
			assert.Equal(t, tt.client, version.client)
			assert.NotNil(t, version.imageCache)
		})
	}
}

func intPtr(i int64) *int64 {
	return &i
}

func strPtr(s string) *string {
	return &s
}

type MockClient struct {
	mock.Mock
}

func (m *MockClient) Tags(ctx context.Context, img string) ([]api.ImageTag, error) {
	args := m.Called(ctx, img)
	return args.Get(0).([]api.ImageTag), args.Error(1)
}
