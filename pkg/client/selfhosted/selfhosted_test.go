package selfhosted

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

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
			switch r.URL.Path {
			case "/v2/repo/image/tags/list":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"tags":["v1.0.0","v2.0.0"]}`))
			case "/v2/repo/image/manifests/v1.0.0":
				w.Header().Add("Docker-Content-Digest", "sha256:abcdef")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"architecture":"amd64","history":[{"v1Compatibility":"{\"created\":\"2023-08-27T12:00:00Z\"}"}]}`))
			case "/v2/repo/image/manifests/v2.0.0":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`)) // Write some blank content
			}
		}))
		defer server.Close()

		h, err := url.Parse(server.URL)
		assert.NoError(t, err)

		tags, err := client.Tags(ctx, h.Host, "repo", "image")

		assert.NoError(t, err)
		assert.Len(t, tags, 2)
		assert.Equal(t, "v1.0.0", tags[0].Tag)
		assert.Equal(t, api.Architecture("amd64"), tags[0].Architecture)
		assert.Equal(t, "sha256:abcdef", tags[0].SHA)
		assert.Equal(t, "v2.0.0", tags[1].Tag)
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
