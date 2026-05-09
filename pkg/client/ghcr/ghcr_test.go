package ghcr

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v70/github"
	"github.com/jarcoal/httpmock"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/stretchr/testify/assert"
)

func setup() {
	httpmock.Activate()
}

func teardown() {
	httpmock.DeactivateAndReset()
}

func registerCommonResponders() {
	httpmock.RegisterResponder("GET", "https://api.github.com/users/test-user-owner",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, `{"type":"User"}`), nil
		})
	httpmock.RegisterResponder("GET", "https://api.github.com/users/test-org-owner",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, `{"type":"Organization"}`), nil
		})
}

func registerTagResponders() {
	httpmock.RegisterResponder("GET", "https://api.github.com/users/test-user-owner/packages/container/test-repo/versions",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, `[
				{
					"name": "sha123",
					"metadata": {
						"container": {
							"tags": ["tag1", "tag2"]
						}
					},
					"created_at": "2023-07-08T12:34:56Z"
				}
			]`), nil
		})
	httpmock.RegisterResponder("GET", "https://api.github.com/orgs/test-org-owner/packages/container/test-repo/versions",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, `[
				{
					"name": "sha123",
					"metadata": {
						"container": {
							"tags": ["tag1", "tag2"]
						}
					},
					"created_at": "2023-07-08T12:34:56Z"
				}
			]`), nil
		})
}

func registerReleaseResponders() {
	httpmock.RegisterResponder("GET", "https://api.github.com/repos/test-user-owner/test-repo/releases",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, `[
				{
					"tag_name": "v1.0.0",
					"published_at": "2023-07-08T12:34:56Z"
				},
				{
					"tag_name": "v1.1.0",
					"created_at": "2023-08-08T12:34:56Z"
				},
				{
					"tag_name": "v9.9.9",
					"draft": true,
					"published_at": "2023-09-08T12:34:56Z"
				}
			]`), nil
		})
}

func TestClient_Tags(t *testing.T) {
	setup()
	defer teardown()

	ctx := context.Background()
	host := "ghcr.io"

	t.Run("successful tags fetch", func(t *testing.T) {
		httpmock.Reset()
		registerCommonResponders()
		registerTagResponders()

		client := New(Options{})
		client.client = github.NewClient(nil) // Use the default HTTP client

		tags, err := client.Tags(ctx, host, "test-user-owner", "test-repo")
		assert.NoError(t, err)
		assert.Len(t, tags, 2)
		assert.ElementsMatch(t, []string{"tag1", "tag2"}, []string{tags[0].Tag, tags[1].Tag})
	})

	t.Run("failed to fetch owner type", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/users/test-user-owner",
			func(req *http.Request) (*http.Response, error) {
				return httpmock.NewStringResponse(404, `{"message": "Not Found"}`), nil
			})

		client := New(Options{})
		client.client = github.NewClient(nil) // Use the default HTTP client

		_, err := client.Tags(ctx, host, "test-user-owner", "test-repo")
		assert.Error(t, err)
	})

	t.Run("token not set, no authorization header", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/users/test-user-owner",
			func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Authorization") != "" {
					t.Errorf("expected no Authorization header, got %s", req.Header.Get("Authorization"))
				}
				return httpmock.NewStringResponse(200, `{"type":"User"}`), nil
			})
		registerTagResponders()

		client := New(Options{}) // No token provided
		client.client = github.NewClient(nil)

		_, err := client.Tags(ctx, host, "test-user-owner", "test-repo")
		assert.NoError(t, err)
	})

	t.Run("token set, authorization header sent", func(t *testing.T) {
		token := "test-token"
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/users/test-user-owner",
			func(req *http.Request) (*http.Response, error) {
				authHeader := req.Header.Get("Authorization")
				expectedAuthHeader := "Bearer " + token
				if authHeader != expectedAuthHeader {
					t.Errorf("expected Authorization header %s, got %s", expectedAuthHeader, authHeader)
				}
				return httpmock.NewStringResponse(200, `{"type":"User"}`), nil
			})

		registerTagResponders()

		client := New(Options{Token: token})

		_, err := client.Tags(ctx, host, "test-user-owner", "test-repo")
		assert.NoError(t, err)
	})

	t.Run("ownerType returns user", func(t *testing.T) {
		httpmock.Reset()
		registerCommonResponders()
		registerTagResponders()

		client := New(Options{})
		client.client = github.NewClient(nil) // Use the default HTTP client

		tags, err := client.Tags(ctx, host, "test-user-owner", "test-repo")
		assert.NoError(t, err)
		assert.Len(t, tags, 2)
		assert.ElementsMatch(t, []string{"tag1", "tag2"}, []string{tags[0].Tag, tags[1].Tag})
	})

	t.Run("ownerType returns org", func(t *testing.T) {
		httpmock.Reset()
		registerCommonResponders()
		registerTagResponders()

		client := New(Options{})
		client.client = github.NewClient(nil) // Use the default HTTP client

		tags, err := client.Tags(ctx, host, "test-org-owner", "test-repo")
		assert.NoError(t, err)
		assert.Len(t, tags, 2)
		assert.ElementsMatch(t, []string{"tag1", "tag2"}, []string{tags[0].Tag, tags[1].Tag})
	})
}

func TestClient_ReleaseTags(t *testing.T) {
	setup()
	defer teardown()

	ctx := context.Background()

	httpmock.Reset()
	registerReleaseResponders()

	client := New(Options{})
	client.client = github.NewClient(nil)

	tags, err := client.ReleaseTags(ctx, "test-user-owner", "test-repo/subpath")
	assert.NoError(t, err)
	assert.Equal(t, []api.ImageTag{
		{Tag: "v1.0.0", Timestamp: parseTime("2023-07-08T12:34:56Z")},
		{Tag: "v1.1.0", Timestamp: parseTime("2023-08-08T12:34:56Z")},
	}, tags)
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}

	return parsed
}
