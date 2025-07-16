package ghcr

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-github/v70/github"
	"github.com/jarcoal/httpmock"
	"github.com/sirupsen/logrus"
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

var testLogger = logrus.NewEntry(&logrus.Logger{Out: io.Discard})
var anonAuth, _ = authn.Anonymous.Authorization()

func TestClient_Tags(t *testing.T) {
	setup()
	defer teardown()

	ctx := context.Background()
	host := "ghcr.io"

	t.Run("successful tags fetch", func(t *testing.T) {
		httpmock.Reset()
		registerCommonResponders()
		registerTagResponders()

		client := New(Options{}, anonAuth, testLogger)
		client.client = github.NewClient(nil) // Use the default HTTP client

		tags, err := client.Tags(ctx, host, "test-user-owner", "test-repo")
		assert.NoError(t, err)
		assert.Len(t, tags, 2)
		assert.Equal(t, "tag1", tags[0].Tag)
		assert.Equal(t, "tag2", tags[1].Tag)
	})

	t.Run("failed to fetch owner type", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("GET", "https://api.github.com/users/test-user-owner",
			func(req *http.Request) (*http.Response, error) {
				return httpmock.NewStringResponse(404, `{"message": "Not Found"}`), nil
			})

		client := New(Options{}, anonAuth, testLogger)
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

		client := New(Options{}, anonAuth, testLogger) // No token provided
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

		client := New(Options{}, &authn.AuthConfig{RegistryToken: token}, testLogger)

		_, err := client.Tags(ctx, host, "test-user-owner", "test-repo")
		assert.NoError(t, err)
	})

	t.Run("ownerType returns user", func(t *testing.T) {
		httpmock.Reset()
		registerCommonResponders()
		registerTagResponders()

		client := New(Options{}, anonAuth, testLogger)
		client.client = github.NewClient(nil) // Use the default HTTP client

		tags, err := client.Tags(ctx, host, "test-user-owner", "test-repo")
		assert.NoError(t, err)
		assert.Len(t, tags, 2)
		assert.Equal(t, "tag1", tags[0].Tag)
		assert.Equal(t, "tag2", tags[1].Tag)
	})

	t.Run("ownerType returns org", func(t *testing.T) {
		httpmock.Reset()
		registerCommonResponders()
		registerTagResponders()

		client := New(Options{}, &authn.AuthConfig{}, testLogger)
		client.client = github.NewClient(nil) // Use the default HTTP client

		tags, err := client.Tags(ctx, host, "test-org-owner", "test-repo")
		assert.NoError(t, err)
		assert.Len(t, tags, 2)
		assert.Equal(t, "tag1", tags[0].Tag)
		assert.Equal(t, "tag2", tags[1].Tag)
	})
}

func TestRepoImage(t *testing.T) {
	tests := map[string]struct {
		path              string
		expRepo, expImage string
	}{
		"empty path should be interpreted as an empty repo and image": {
			path:     "",
			expRepo:  "",
			expImage: "",
		},
		"one segement should be interpreted as 'repo'": {
			path:     "jetstack-cre",
			expRepo:  "jetstack-cre",
			expImage: "",
		},
		"two segments to path should return both": {
			path:     "jetstack-cre/version-checker",
			expRepo:  "jetstack-cre",
			expImage: "version-checker",
		},
		"multiple segments to path should return first segment in repo, rest in image": {
			path:     "k8s-artifacts-prod/ingress-nginx/nginx",
			expRepo:  "k8s-artifacts-prod",
			expImage: "ingress-nginx/nginx",
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// handler.opts.Token = "fake-token"
			repo, image := handler.RepoImageFromPath(test.path)
			if repo != test.expRepo && image != test.expImage {
				t.Errorf("%s: unexpected repo/image, exp=%s/%s got=%s/%s",
					test.path, test.expRepo, test.expImage, repo, image)
			}
		})
	}
}
