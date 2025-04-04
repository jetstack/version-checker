package oci

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"

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
					},
					{
						Tag: "b",
					},
					{
						Tag: "c",
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", host, tc.repo, tc.img))
			if err != nil {
				t.Fatalf("unexpected error parsing repo: %s", err)
			}
			for _, tag := range tc.wantTags {
				if err := remote.Write(repo.Tag(tag.Tag), empty.Image); err != nil {
					t.Fatalf("unexpected error writing image to tag: %s", err)
				}
			}
			return tc
		},
		"should list expected tags within a root repository": func(t *testing.T, host string) *testCase {
			tc := &testCase{
				img: "foo",
				wantTags: []api.ImageTag{
					{
						Tag: "a",
					},
					{
						Tag: "b",
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s", host, tc.img))
			if err != nil {
				t.Fatalf("unexpected error parsing repo: %s", err)
			}
			for _, tag := range tc.wantTags {
				if err := remote.Write(repo.Tag(tag.Tag), empty.Image); err != nil {
					t.Fatalf("unexpected error writing image to tag: %s", err)
				}
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
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", host, tc.repo, tc.img))
			if err != nil {
				t.Fatalf("unexpected error parsing repo: %s", err)
			}
			for _, tag := range tc.wantTags {
				if err := remote.Write(repo.Tag(tag.Tag), empty.Image); err != nil {
					t.Fatalf("unexpected error writing image to tag: %s", err)
				}
			}
			return tc
		},
		"should return an empty list and no error for a repository with no tags": func(t *testing.T, host string) *testCase {
			tc := &testCase{
				repo: "foo",
				img:  "bar",
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", host, tc.repo, tc.img))
			if err != nil {
				t.Fatalf("unexpected error parsing repo: %s", err)
			}

			// Write a tag but then delete it so the repository
			// exists but it has no tags
			if err := remote.Write(repo.Tag("latest"), empty.Image); err != nil {
				t.Fatalf("unexpected error writing image to tag: %s", err)
			}
			if err := remote.Delete(repo.Tag("latest")); err != nil {
				t.Fatalf("unexpected error writing image to tag: %s", err)
			}
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

			c, err := New(new(Options))
			if err != nil {
				t.Fatalf("unexpected error creating client: %s", err)
			}

			tc := fn(t, host)

			gotTags, err := c.Tags(ctx, host, tc.repo, tc.img)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantTags, gotTags)
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

	c, err := New(new(Options))
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
	r := httptest.NewServer(registry.New())
	t.Cleanup(r.Close)
	u, err := url.Parse(r.URL)
	require.NoError(t, err)

	return u.Host
}
