package selfhosted

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jetstack/version-checker/pkg/api"
	selfhostederrors "github.com/jetstack/version-checker/pkg/client/selfhosted/errors"
)

func TestNew(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	t.Run("successful client creation with username and password", func(t *testing.T) {
		opts := &Options{
			Host:     "https://testregistry.com",
			Username: "testuser",
			Password: "testpass",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v2/token", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"token":"testtoken"}`))
		}))
		defer server.Close()

		opts.Host = server.URL
		client, err := New(ctx, log, opts)

		assert.NoError(t, err)
		assert.Equal(t, "testtoken", client.Bearer)
	})

	t.Run("error on invalid URL", func(t *testing.T) {
		opts := &Options{
			Host: "://invalid-url",
		}

		client, err := New(ctx, log, opts)

		assert.Nil(t, client)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed parsing url")
	})

	// t.Run("Error on missing host", func(t *testing.T) {
	// 	opts := &Options{
	// 		Host:   "",
	// 		CAPath: "invalid/path",
	// 	}
	// 	client, err := New(ctx, log, opts)
	// 	assert.Nil(t, client)
	// 	assert.Error(t, err)
	// 	assert.Contains(t, err.Error(), "host cannot be empty")
	// })

	t.Run("error on username/password and bearer token both specified", func(t *testing.T) {
		opts := &Options{
			Host:     "https://testregistry.com",
			Username: "testuser",
			Password: "testpass",
			Bearer:   "testtoken",
		}

		client, err := New(ctx, log, opts)

		assert.Nil(t, client)
		assert.EqualError(t, err, "cannot specify Bearer token as well as username/password")
	})

	t.Run("successful client creation with bearer token", func(t *testing.T) {
		opts := &Options{
			Host:   "https://testregistry.com",
			Bearer: "testtoken",
		}

		client, err := New(ctx, log, opts)

		assert.NoError(t, err)
		assert.Equal(t, "testtoken", client.Bearer)
	})

	t.Run("error on invalid CA path", func(t *testing.T) {
		opts := &Options{
			Host:     "https://testregistry.com",
			CAPath:   "invalid/path",
			Insecure: true,
		}

		client, err := New(ctx, log, opts)

		assert.Nil(t, client)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to append")
	})
}

func TestName(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	client := &Client{
		Options: &Options{
			Host: "testhost",
		},
		log: log,
	}

	assert.Equal(t, "testhost", client.Name())

	client.Host = ""
	assert.Equal(t, "selfhosted", client.Name())
}

func TestTags(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	t.Run("successful Tags fetch", func(t *testing.T) {
		client := &Client{
			Client: &http.Client{},
			log:    log,
			Options: &Options{
				Host: "testregistry.com",
			},
			httpScheme: "http",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// l.Infof("Got request: %v", r)
			switch r.URL.Path {
			case "/v2/repo/image/tags/list":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(TagResponse{Tags: []string{"v1.0.0", "v2.0.0"}})

			case "/v2/repo/image/manifests/v1.0.0":
				w.Header().Add("Docker-Content-Digest", "sha256:abcdef")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(ManifestResponse{
					Architecture: api.Architecture("amd64"),
					History: []History{
						{
							V1Compatibility: V1CompatibilityWrapper{
								V1Compatibility: V1Compatibility{Created: time.Now()},
							},
						},
					},
				})

			case "/v2/repo/image/manifests/v2.0.0":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`)) // Write some blank content

			// This image is a manifest List
			case "/v2/repo/multiimage/tags/list":
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(TagResponse{Tags: []string{"v2.2.0"}})

			case "/v2/repo/multiimage/manifests/v2.2.0":
				acpt := r.Header.Get("Accept")
				log.Warnf("Got following request: %v", acpt)
				switch acpt {
				// If we have multiple formats...
				case strings.Join([]string{dockerAPIv2Header, dockerAPIv2ManifestList}, ","):
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(V2ManifestListResponse{
						Manifests: []V2ManifestListEntry{
							{Digest: "asjhfvbasjhbfsaj", Platform: api.Platform{OS: api.OS("Linux"), Architecture: api.Architecture("arm64")}},
						},
					})

					// Docker V1 API
				case dockerAPIv1Header:
					w.WriteHeader(http.StatusOK)
					w.Header().Add("Docker-Content-Digest", "sha265:asgjnaskjgbsajgsa")
					_, _ = w.Write([]byte(`{}`)) // Write some blank content

					// Docker V2 Header...
				case dockerAPIv2Header:
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(ErrorResponse{Errors: []ErrorType{
						{
							Code:    "MANIFEST_UNKNOWN",
							Message: `Manifest has media type "application/vnd.docker.distribution.manifest.list.v2+json" but client accepts ["application/vnd.docker.distribution.manifest.v1+json"]`,
						},
					}})

					// ManifestList ONLY
				case dockerAPIv2ManifestList:
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`)) // Write some blank content

				}
			}
		}))
		defer server.Close()

		h, err := url.Parse(server.URL)
		assert.NoError(t, err)

		t.Run("Standard Single Arch Image", func(t *testing.T) {
			tags, err := client.Tags(ctx, h.Host, "repo", "image")
			require.NoError(t, err)
			require.Len(t, tags, 2)

			// We don't care of the order, we just want to make sure we have the tags
			assert.ElementsMatch(t, []string{"v1.0.0", "v2.0.0"}, []string{tags[0].Tag, tags[1].Tag})
			assert.Equal(t, api.Architecture("amd64"), tags[0].Architecture)
			assert.Equal(t, "sha256:abcdef", tags[0].SHA)
		})

		t.Run("MultiArch ManifestList v2.2", func(t *testing.T) {
			tags, err := client.Tags(ctx, h.Host, "repo", "multiimage")

			assert.NoError(t, err)
			require.Len(t, tags, 1)
			assert.Equal(t, "v2.2.0", tags[0].Tag)
		})
	})

	t.Run("error fetching tags", func(t *testing.T) {
		client := &Client{
			Client: &http.Client{},
			log:    log,
			Options: &Options{
				Host: "https://testregistry.com",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		tags, err := client.Tags(ctx, server.URL, "repo", "image")
		assert.Nil(t, tags)
		assert.Error(t, err)
	})
}

func TestDoRequest(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	client := &Client{
		Client: &http.Client{},
		Options: &Options{
			Host: "testhost",
		},
		log:        log,
		httpScheme: "http",
	}

	t.Run("successful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v2/repo/image/tags/list", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"tags":["v1","v2"]}`))
		}))
		defer server.Close()

		h, err := url.Parse(server.URL)
		assert.NoError(t, err)

		var tagResponse TagResponse
		headers, err := client.doRequest(ctx, h.Host+"/v2/repo/image/tags/list", "", &tagResponse)

		assert.NoError(t, err)
		assert.NotNil(t, headers)
		assert.Equal(t, []string{"v1", "v2"}, tagResponse.Tags)
	})

	t.Run("error on non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}))
		defer server.Close()

		h, err := url.Parse(server.URL)
		assert.NoError(t, err)

		var tagResponse TagResponse
		headers, err := client.doRequest(ctx, h.Host+"/v2/repo/image/tags/list", "", &tagResponse)

		assert.Nil(t, headers)
		assert.Error(t, err)
		var httpErr *selfhostederrors.HTTPError
		if errors.As(err, &httpErr) {
			assert.Equal(t, http.StatusNotFound, httpErr.StatusCode)
			assert.Equal(t, "not found", string(httpErr.Body))
		}
	})

	t.Run("error on invalid json response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		h, err := url.Parse(server.URL)
		assert.NoError(t, err)

		var tagResponse TagResponse
		headers, err := client.doRequest(ctx, h.Host+"/v2/repo/image/tags/list", "", &tagResponse)

		assert.Nil(t, headers)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected")
	})

	t.Run("use basic auth in request", func(t *testing.T) {
		username := "foo"
		password := "bar"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, r.Header.Get("Authorization"), fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(username+":"+password))))
			assert.Equal(t, "/v2/repo/image/tags/list", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"tags":["v1","v2"]}`))
		}))
		defer server.Close()

		h, err := url.Parse(server.URL)
		assert.NoError(t, err)

		client.Username = username
		client.Password = password

		var tagResponse TagResponse
		headers, err := client.doRequest(ctx, h.Host+"/v2/repo/image/tags/list", "", &tagResponse)

		assert.NoError(t, err)
		assert.NotNil(t, headers)
		assert.Equal(t, []string{"v1", "v2"}, tagResponse.Tags)
	})
}

func TestSetupBasicAuth(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	client := &Client{
		Client: &http.Client{},
		Options: &Options{
			Username: "testuser",
			Password: "testpass",
		},
		log: log,
	}

	t.Run("successful auth setup", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v2/token", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"token":"testtoken"}`))
		}))
		defer server.Close()

		token, err := client.setupBasicAuth(ctx, server.URL, "/v2/token")
		assert.NoError(t, err)
		assert.Equal(t, "testtoken", token)
	})

	t.Run("error on invalid json response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		token, err := client.setupBasicAuth(ctx, server.URL, "/v2/token")
		assert.Empty(t, token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")
	})

	t.Run("error on non-200 status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
		}))
		defer server.Close()

		token, err := client.setupBasicAuth(ctx, server.URL, "/v2/token")
		assert.Empty(t, token)
		assert.Error(t, err)
		var httpErr *selfhostederrors.HTTPError
		if errors.As(err, &httpErr) {
			assert.Equal(t, http.StatusUnauthorized, httpErr.StatusCode)
			assert.Equal(t, "unauthorized", string(httpErr.Body))
		}
	})

	t.Run("error on request creation failure", func(t *testing.T) {
		client := &Client{
			Client: &http.Client{},
			Options: &Options{
				Username: "testuser",
				Password: "testpass",
			},
			log: log,
		}

		token, err := client.setupBasicAuth(ctx, "localhost:999999", "/v2/token")
		assert.Empty(t, token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send basic auth request")
	})
}

func TestNewTLSConfig(t *testing.T) {
	t.Run("successful TLS config creation with valid CA path", func(t *testing.T) {
		caFile, err := os.CreateTemp("", "ca.pem")
		assert.NoError(t, err)
		defer func() { assert.NoError(t, os.Remove(caFile.Name())) }()

		_, err = caFile.WriteString(`-----BEGIN CERTIFICATE-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwf3Kq/BnEePvM6rSGPP6
6uUbzIAdx0+EjHRJ1yqCqk8MzY+m5OncEjgpG0FDDpdqYPOUE4EzjjIlNInxG8Vi
DfWmi8csEQYrtyNzzlF+bWwWv/1U+UuRgZqtwFZxC4DLIE1Bke4isr7g91DU5B8G
b+6eGHjql0zPz9bL7s5er8kpDp1o6ZZtGPE3F18LPS48pZyRIN/T4vPz4uA/Zay/
aEB8E+yoI8dw48LUVZDjDN3mthBb8k68ngLqBaIgF+1EQpe2I1a/nZBQTu9yn8Z1
Y7nG8XdxKAr5e+CZ8x8NUvydF1DZDSV1Mf1GriMEwLkA5P4oY8EbOxDJTuJrAXjZ
tQIDAQAB
-----END CERTIFICATE-----`)
		assert.NoError(t, err)
		err = caFile.Close()
		assert.NoError(t, err)

		tlsConfig, err := newTLSConfig(false, caFile.Name())
		assert.NoError(t, err)
		assert.NotNil(t, tlsConfig)
		assert.False(t, tlsConfig.InsecureSkipVerify)
	})

	t.Run("successful TLS config creation with empty CA path", func(t *testing.T) {
		tlsConfig, err := newTLSConfig(true, "")
		assert.NoError(t, err)
		assert.NotNil(t, tlsConfig)
		assert.True(t, tlsConfig.InsecureSkipVerify)
	})

	t.Run("error on invalid CA path", func(t *testing.T) {
		tlsConfig, err := newTLSConfig(false, "/invalid/path")
		assert.Nil(t, tlsConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to append")
	})
}
