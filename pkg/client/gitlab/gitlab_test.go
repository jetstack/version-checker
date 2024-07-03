package gitlab

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jarcoal/httpmock"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/stretchr/testify/assert"
	gitlab "github.com/xanzy/go-gitlab"
)

// NewMockGitlabClient creates a new Client with a mocked HTTP client and retries disabled
func NewMockGitlabClient(mockTransport http.RoundTripper, opts *Options) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 0 // Disable retries
	retryClient.HTTPClient = &http.Client{
		Transport: mockTransport,
		Timeout:   1 * time.Second, // Set a timeout for HTTP requests
	}
	client, _ := gitlab.NewClient("", gitlab.WithHTTPClient(retryClient.StandardClient()))
	return &Client{
		Client:  client,
		Options: opts,
	}
}

func TestTags(t *testing.T) {
	startGlobal := time.Now()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	t.Logf("Global setup took %v", time.Since(startGlobal))

	// Add a default responder to catch any unexpected API calls
	httpmock.RegisterNoResponder(httpmock.NewStringResponder(500, "unexpected API call"))

	tests := []struct {
		name                     string
		repo                     string
		image                    string
		mockRepositoriesResponse string
		mockTagsResponse         []string
		mockRepositoriesError    bool
		mockTagsError            bool
		expectedTags             []api.ImageTag
		expectedError            string
	}{
		{
			name:                     "Successful retrieval with pagination",
			repo:                     "my-project",
			image:                    "my-image",
			mockRepositoriesResponse: `[{"id":123,"path":"my-image"}]`,
			mockTagsResponse: []string{
				`[{"name":"v1.0.0","digest":"sha256:abc123","created_at":"2023-01-01T00:00:00Z"}]`,
				`[{"name":"v1.1.0","digest":"sha256:def456","created_at":"2023-02-01T00:00:00Z"}]`,
			},
			expectedTags: []api.ImageTag{
				{
					Tag:       "v1.0.0",
					SHA:       "sha256:abc123",
					Timestamp: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Tag:       "v1.1.0",
					SHA:       "sha256:def456",
					Timestamp: time.Date(2023, time.February, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			expectedError: "",
		},
		{
			name:                  "Error in ListRegistryRepositories",
			repo:                  "my-project",
			image:                 "my-image",
			mockRepositoriesError: true,
			expectedError:         "giving up after",
		},
		{
			name:                     "Error in ListRegistryRepositoryTags",
			repo:                     "my-project",
			image:                    "my-image",
			mockRepositoriesResponse: `[{"id":123,"path":"my-image"}]`,
			mockTagsError:            true,
			expectedError:            "giving up after",
		},
		{
			name:                     "Repository not found",
			repo:                     "my-project",
			image:                    "my-image",
			mockRepositoriesResponse: `[]`,
			expectedError:            "repository 'my-image' not found",
		},
		{
			name:                     "No tags found",
			repo:                     "my-project",
			image:                    "my-image",
			mockRepositoriesResponse: `[{"id":123,"path":"my-image"}]`,
			mockTagsResponse:         []string{`[]`},
			expectedTags:             []api.ImageTag{},
			expectedError:            "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			httpmock.Reset() // Clear any existing responders
			t.Logf("Reset responders took %v", time.Since(start))

			baseURL := "https://gitlab.com/api/v4"
			projectRegistryURL := baseURL + "/projects/my-project/registry/repositories?page=1&per_page=20"
			tagsURLPage1 := baseURL + "/projects/my-project/registry/repositories/123/tags?page=1&per_page=20"

			if tt.mockRepositoriesError {
				t.Log("Simulating error in ListRegistryRepositories")
				httpmock.RegisterResponder("GET", projectRegistryURL,
					httpmock.NewStringResponder(500, "httpmock: error"))
			} else {
				httpmock.RegisterResponder("GET", projectRegistryURL,
					httpmock.NewStringResponder(200, tt.mockRepositoriesResponse))
			}

			if tt.mockTagsError {
				t.Log("Simulating error in ListRegistryRepositoryTags")
				httpmock.RegisterResponder("GET", tagsURLPage1,
					httpmock.NewStringResponder(500, "httpmock: error"))
			} else {
				for i, tagsResponse := range tt.mockTagsResponse {
					url := tagsURLPage1
					resp := httpmock.NewStringResponse(200, tagsResponse)
					resp.Header.Set("Content-Type", "application/json")
					resp.Header.Set("X-Total-Pages", strconv.Itoa(len(tt.mockTagsResponse)))
					resp.Header.Set("X-Page", strconv.Itoa(i+1))
					httpmock.RegisterResponder("GET", url, httpmock.ResponderFromResponse(resp))
					tagsURLPage1 = baseURL + "/projects/my-project/registry/repositories/123/tags?page=" + strconv.Itoa(i+2) + "&per_page=20"
				}
			}

			client := NewMockGitlabClient(httpmock.DefaultTransport, &Options{
				Host: baseURL,
			})
			t.Logf("Client setup took %v", time.Since(start))

			ctx := context.Background()
			result, err := client.Tags(ctx, baseURL, tt.repo, tt.image)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, len(tt.expectedTags))
				for i, tag := range result {
					assert.Equal(t, tt.expectedTags[i].Tag, tag.Tag)
					assert.Equal(t, tt.expectedTags[i].SHA, tag.SHA)
					assert.WithinDuration(t, tt.expectedTags[i].Timestamp, tag.Timestamp, time.Second)
				}
			}
			duration := time.Since(start)
			t.Logf("Test %s took %v", tt.name, duration)
		})
	}
}
