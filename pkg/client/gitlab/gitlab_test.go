package gitlab

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/stretchr/testify/assert"
	gitlab "github.com/xanzy/go-gitlab"
)

// NewMockGitlabClient creates a new Client with a mocked HTTP client
func NewMockGitlabClient(mockTransport http.RoundTripper, opts *Options) *Client {
	httpClient := &http.Client{Transport: mockTransport}
	client, _ := gitlab.NewClient("", gitlab.WithHTTPClient(httpClient))
	return &Client{
		Client:  client,
		Options: opts,
	}
}

func TestTags(t *testing.T) {
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
			expectedError:         "httpmock: error",
		},
		{
			name:                     "Error in ListRegistryRepositoryTags",
			repo:                     "my-project",
			image:                    "my-image",
			mockRepositoriesResponse: `[{"id":123,"path":"my-image"}]`,
			mockTagsError:            true,
			expectedError:            "httpmock: error",
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
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			baseURL := "https://gitlab.com/api/v4"
			projectRegistryURL := baseURL + "/projects/my-project/registry/repositories?page=1&per_page=20"
			tagsURLPage1 := baseURL + "/projects/my-project/registry/repositories/123/tags?page=1&per_page=20"

			if tt.mockRepositoriesError {
				httpmock.RegisterResponder("GET", projectRegistryURL,
					httpmock.NewStringResponder(500, "httpmock: error"))
			} else {
				httpmock.RegisterResponder("GET", projectRegistryURL,
					httpmock.NewStringResponder(200, tt.mockRepositoriesResponse))
			}

			if tt.mockTagsError {
				httpmock.RegisterResponder("GET", tagsURLPage1,
					httpmock.NewStringResponder(500, "httpmock: error"))
			} else {
				for i, tagsResponse := range tt.mockTagsResponse {
					url := tagsURLPage1
					responderWithHeader := httpmock.ResponderFromResponse(&http.Response{
						StatusCode: 200,
						Header: http.Header{
							"Content-Type":  {"application/json"},
							"X-Total-Pages": {strconv.Itoa(len(tt.mockTagsResponse))},
							"X-Page":        {strconv.Itoa(i + 1)},
						},
						Body: httpmock.NewRespBodyFromString(tagsResponse),
					})
					httpmock.RegisterResponder("GET", url, responderWithHeader)
					tagsURLPage1 = baseURL + "/projects/my-project/registry/repositories/123/tags?page=" + strconv.Itoa(i+2) + "&per_page=20"
				}
			}

			client := NewMockGitlabClient(httpmock.DefaultTransport, &Options{
				Host: baseURL,
			})

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
		})
	}
}
