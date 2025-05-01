package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sirupsen/logrus"
)

type hostnameOverride struct {
	Host string
	RT   http.RoundTripper
}

func (r *hostnameOverride) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Host != r.Host {
		if testing.Verbose() {
			fmt.Printf("Overriding URI from: %s to %s\n", req.Host, strings.TrimPrefix(r.Host, "http://"))
		}
		req.Host = strings.TrimPrefix(r.Host, "http://")
		req.URL.Host = strings.TrimPrefix(r.Host, "http://")
		req.URL.Scheme = "http"
	}
	// fmt.Printf("Req: %+v", req)
	return r.RT.RoundTrip(req)
}

func TestTags(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	ctx := context.Background()

	t.Run("successful Tags fetch", func(t *testing.T) {

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NotEmpty(t, r.Header.Get("Authorization"))

			require.Equal(t, "/v2/repositories/testrepo/testimage/tags", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(TagResponse{
				Results: []Result{
					{
						Name:      "v1.0.0",
						Timestamp: time.Now().Add(-24 * time.Hour).Format(time.RFC3339Nano),
						Digest:    "sha256:abcdef",
						Images: []Image{
							{Digest: "sha256:child1", OS: "linux", Architecture: "amd64"},
						},
					},
					{
						Name:      "v2.0.0",
						Timestamp: time.Now().Add(-48 * time.Hour).Format(time.RFC3339Nano),
						Images: []Image{
							{Digest: "sha256:child2", OS: "linux", Architecture: "amd64"},
						},
					},
				},
			})
		}))
		defer server.Close()

		client := &Client{
			Client: server.Client(),
			log:    log,
			Options: Options{
				Token: "testtoken",
			},
		}

		client.Transport = &hostnameOverride{RT: server.Client().Transport, Host: server.URL}

		tags, err := client.Tags(ctx, "NOT USED!", "testrepo", "testimage")
		require.NoError(t, err)
		require.Len(t, tags, 2)

		assert.Equal(t, "v1.0.0", tags[0].Tag)
		assert.Equal(t, "sha256:abcdef", tags[0].SHA)
		assert.Equal(t, api.OS("linux"), tags[0].Children[0].OS)
		assert.Equal(t, api.Architecture("amd64"), tags[0].Children[0].Architecture)

		assert.Equal(t, "v2.0.0", tags[1].Tag)
		assert.NotEmpty(t, tags[1].SHA)
	})

	t.Run("error on invalid response", func(t *testing.T) {

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client := &Client{
			Client: server.Client(),
			log:    log,
			Options: Options{
				Token: "testtoken",
			},
		}
		client.Transport = &hostnameOverride{RT: server.Client().Transport, Host: server.URL}

		tags, err := client.Tags(ctx, "NOT USED!", "testrepo", "testimage")
		assert.Nil(t, tags)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected image tags response")
	})

	t.Run("error on non-200 status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}))
		defer server.Close()

		client := &Client{
			Client: server.Client(),
			log:    log,
			Options: Options{
				Token: "testtoken",
			},
		}
		client.Transport = &hostnameOverride{RT: server.Client().Transport, Host: server.URL}

		tags, err := client.Tags(ctx, "NOT USED!", "testrepo", "testimage")
		assert.Nil(t, tags)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected image")
	})
}
