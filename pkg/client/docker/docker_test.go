package docker

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

// Assuming Options, Client, AuthResponse, TagResponse, and other structs are defined in types.go

func TestNew(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		opts        Options
		mockResp    string
		mockStatus  int
		expectError bool
	}{
		{
			name: "Successful Auth",
			opts: Options{
				Username: "testuser",
				Password: "testpassword",
			},
			mockResp:    `{"token": "testtoken"}`,
			mockStatus:  http.StatusOK,
			expectError: false,
		},
		{
			name: "Auth Error",
			opts: Options{
				Username: "testuser",
				Password: "wrongpassword",
			},
			mockResp:    `{"detail": "Invalid credentials"}`,
			mockStatus:  http.StatusUnauthorized,
			expectError: true,
		},
		{
			name: "Token Specified with Username/Password",
			opts: Options{
				Username: "testuser",
				Password: "testpassword",
				Token:    "testtoken",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start by activating httpmock
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			if tt.mockResp != "" {
				httpmock.RegisterResponder("POST", loginURL,
					httpmock.NewStringResponder(tt.mockStatus, tt.mockResp))
			}

			client, err := New(ctx, tt.opts)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				if tt.opts.Token == "" {
					assert.Equal(t, "testtoken", client.Options.Token)
				}
			}
		})
	}
}

func TestTags(t *testing.T) {
	ctx := context.Background()
	opts := Options{
		Token: "testtoken",
	}

	client, err := New(ctx, opts)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	tests := []struct {
		name        string
		repo        string
		image       string
		mockResp    string
		mockStatus  int
		expectError bool
		expectedLen int
	}{
		{
			name:  "Successful Tags Fetch",
			repo:  "testrepo",
			image: "testimage",
			mockResp: `{
				"results": [{
					"name": "v1.0",
					"images": [{
						"digest": "sha256:123",
						"os": "linux",
						"architecture": "amd64"
					}],
					"timestamp": "2021-01-01T00:00:00Z"
				}],
				"next": ""
			}`,
			mockStatus:  http.StatusOK,
			expectError: false,
			expectedLen: 1,
		},
		{
			name:  "No Images",
			repo:  "testrepo",
			image: "noimages",
			mockResp: `{
				"results": [{
					"name": "v1.0",
					"images": [],
					"timestamp": "2021-01-01T00:00:00Z"
				}],
				"next": ""
			}`,
			mockStatus:  http.StatusOK,
			expectError: false,
			expectedLen: 0,
		},
		{
			name:        "API Error",
			repo:        "testrepo",
			image:       "errorimage",
			mockResp:    `{"detail": "Not found"}`,
			mockStatus:  http.StatusNotFound,
			expectError: false,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start by activating httpmock
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			mockURL := fmt.Sprintf(lookupURL, tt.repo, tt.image)
			httpmock.RegisterResponder("GET", mockURL,
				httpmock.NewStringResponder(tt.mockStatus, tt.mockResp))

			tags, err := client.Tags(ctx, "", tt.repo, tt.image)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, tags)
			} else {
				assert.NoError(t, err)
				assert.Len(t, tags, tt.expectedLen)
			}
		})
	}
}

func TestBasicAuthSetup(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		opts          Options
		mockResp      string
		mockStatus    int
		expectError   bool
		expectedToken string
	}{
		{
			name: "Successful Auth Setup",
			opts: Options{
				Username: "testuser",
				Password: "testpassword",
			},
			mockResp:      `{"token": "testtoken"}`,
			mockStatus:    http.StatusOK,
			expectError:   false,
			expectedToken: "testtoken",
		},
		{
			name: "Auth Setup Error",
			opts: Options{
				Username: "testuser",
				Password: "wrongpassword",
			},
			mockResp:    `{"detail": "Invalid credentials"}`,
			mockStatus:  http.StatusUnauthorized,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start by activating httpmock
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterResponder("POST", loginURL,
				httpmock.NewStringResponder(tt.mockStatus, tt.mockResp))

			client := &http.Client{
				Timeout: time.Second * 10,
			}
			token, err := basicAuthSetup(ctx, client, tt.opts)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}
