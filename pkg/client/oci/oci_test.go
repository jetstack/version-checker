package oci

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/jetstack/version-checker/pkg/api"
)

func TestClientTags(t *testing.T) {
	ctx := context.Background()
	emptySha, err := empty.Image.Digest()
	require.NoError(t, err)

	type testCase struct {
		repo     string
		img      string
		wantTags []api.ImageTag
		wantErr  bool
	}
	testCases := map[string]func(t *testing.T, host string) *testCase{
		"should list expected tags": func(t *testing.T, host string) *testCase {
			tc := &testCase{
				repo: "foo",
				img:  "bar",
				wantTags: []api.ImageTag{
					{
						Tag: "a",
						SHA: emptySha.String(),
					},
					{
						Tag: "b",
						SHA: emptySha.String(),
					},
					{
						Tag: "c",
						SHA: emptySha.String(),
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", host, tc.repo, tc.img))
			require.NoError(t, err)

			for _, tag := range tc.wantTags {
				require.NoError(t,
					remote.Write(repo.Tag(tag.Tag), empty.Image),
				)
			}
			return tc
		},
		"should list expected tags within a root repository": func(t *testing.T, host string) *testCase {
			tc := &testCase{
				img: "foo",
				wantTags: []api.ImageTag{
					{
						Tag: "a",
						SHA: emptySha.String(),
					},
					{
						Tag: "b",
						SHA: emptySha.String(),
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s", host, tc.img))
			require.NoError(t, err)

			for _, tag := range tc.wantTags {
				require.NoError(t,
					remote.Write(repo.Tag(tag.Tag), empty.Image),
				)
			}
			return tc
		},
		"should list expected tags within a sub-repository": func(t *testing.T, host string) *testCase {
			tc := &testCase{
				repo: "foo/bar",
				img:  "baz",
				wantTags: []api.ImageTag{
					{
						Tag: "a",
						SHA: emptySha.String(),
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", host, tc.repo, tc.img))
			require.NoError(t, err)

			for _, tag := range tc.wantTags {
				require.NoError(t,
					remote.Write(repo.Tag(tag.Tag), empty.Image),
				)
			}
			return tc
		},
		"should return an empty list and no error for a repository with no tags": func(t *testing.T, host string) *testCase {
			tc := &testCase{
				repo: "foo",
				img:  "bar",
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", host, tc.repo, tc.img))
			require.NoError(t, err)

			// Write a tag but then delete it so the repository
			// exists but it has no tags
			require.NoError(t,
				remote.Write(repo.Tag("latest"), empty.Image),
			)
			require.NoError(t,
				remote.Delete(repo.Tag("latest")),
			)
			return tc
		},
		"should return an error when listing a repository that doesn't exist": func(t *testing.T, host string) *testCase {
			return &testCase{
				repo:    "foo",
				img:     "bar",
				wantErr: true,
			}
		},
	}

	for testName, fn := range testCases {
		t.Run(testName, func(t *testing.T) {
			host := setupRegistry(t)

			c, err := New(new(Options), logrus.NewEntry(logrus.New()))
			require.NoError(t, err)

			tc := fn(t, host)

			gotTags, err := c.Tags(ctx, host, tc.repo, tc.img)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			// We don't care about the order - but to ensure that the elements we expect
			// exist within the output
			assert.ElementsMatch(t, tc.wantTags, gotTags)
		})
	}
}

func TestClientRepoImageFromPath(t *testing.T) {
	tests := map[string]struct {
		path              string
		expRepo, expImage string
	}{
		"empty path should be interpreted as an empty repo and image": {
			path:     "",
			expRepo:  "",
			expImage: "",
		},
		"one segment should be interpreted as 'repo'": {
			path:     "jetstack-cre",
			expRepo:  "",
			expImage: "jetstack-cre",
		},
		"two segments to path should return both": {
			path:     "jetstack-cre/version-checker",
			expRepo:  "jetstack-cre",
			expImage: "version-checker",
		},
		"multiple segments to path should return first segments in repo, last segment in image": {
			path:     "k8s-artifacts-prod/ingress-nginx/nginx",
			expRepo:  "k8s-artifacts-prod/ingress-nginx",
			expImage: "nginx",
		},
	}

	c, err := New(new(Options), logrus.NewEntry(logrus.New()))
	if err != nil {
		t.Fatalf("unexpected error creating client: %s", err)
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo, image := c.RepoImageFromPath(test.path)
			assert.Equal(t, test.expRepo, repo)
			assert.Equal(t, test.expImage, image)
		})
	}
}

func setupRegistry(t *testing.T) string {
	r := httptest.NewServer(registry.New(
		registry.Logger(log.New(io.Discard, "", log.LstdFlags)),
		registry.WithReferrersSupport(false),
	))
	t.Cleanup(r.Close)
	u, err := url.Parse(r.URL)
	require.NoError(t, err)

	return u.Host
}
