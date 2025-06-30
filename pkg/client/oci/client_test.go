package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var logger = logrus.NewEntry(&logrus.Logger{Out: io.Discard})
var anonAuth, _ = authn.Anonymous.Authorization()
var emptyDigest, _ = empty.Image.Digest()

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
						SHA: emptyDigest.String(),
					},
					{
						Tag: "b",
						SHA: emptyDigest.String(),
					},
					{
						Tag: "c",
						SHA: emptyDigest.String(),
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", host, tc.repo, tc.img))
			require.NoError(t, err)

			for _, tag := range tc.wantTags {
				err := remote.Write(repo.Tag(tag.Tag), empty.Image)
				require.NoError(t, err)
			}
			return tc
		},
		"should list expected tags within a root repository": func(t *testing.T, host string) *testCase {
			tc := &testCase{
				img: "foo",
				wantTags: []api.ImageTag{
					{
						Tag: "a",
						SHA: emptyDigest.String(),
					},
					{
						Tag: "b",
						SHA: emptyDigest.String(),
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s", host, tc.img))
			require.NoError(t, err)

			for _, tag := range tc.wantTags {
				err := remote.Write(repo.Tag(tag.Tag), empty.Image)
				require.NoError(t, err)
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
						SHA: emptyDigest.String(),
					},
					{
						Tag: "indx",
						SHA: emptyDigest.String(),
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", host, tc.repo, tc.img))
			require.NoError(t, err)

			for _, tag := range tc.wantTags {
				err := remote.Write(repo.Tag(tag.Tag), empty.Image)
				require.NoError(t, err)
			}
			return tc
		},
		"should return all Tags when Image contains Mutiple Arch/Os": func(t *testing.T, host string) *testCase {
			tc := &testCase{
				repo: "foo/bar",
				img:  "baz",
				wantTags: []api.ImageTag{
					{
						Tag:          "a",
						SHA:          emptyDigest.String(),
						OS:           "linux",
						Architecture: "arm64",
					},
					{
						Tag:          "a",
						SHA:          emptyDigest.String(),
						OS:           "linux",
						Architecture: "amd64",
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", host, tc.repo, tc.img))
			require.NoError(t, err)

			index := mutate.IndexMediaType(empty.Index, types.OCIImageIndex)
			index = mutate.AppendManifests(index,
				mutate.IndexAddendum{
					Add: empty.Image,
					Descriptor: v1.Descriptor{
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "arm64",
						},
					},
				},
				mutate.IndexAddendum{
					Add: empty.Image,
					Descriptor: v1.Descriptor{
						Platform: &v1.Platform{
							OS:           "linux",
							Architecture: "amd64",
						},
					},
				})

			imgTag := repo.Tag("a")
			err = remote.Write(imgTag, empty.Image)
			require.NoError(t, err)

			err = remote.WriteIndex(imgTag, index)
			require.NoError(t, err)

			return tc
		},
		"should return all Tags when Image contains attestations": func(t *testing.T, host string) *testCase {
			tc := &testCase{
				repo: "foo/bar",
				img:  "baz",
				wantTags: []api.ImageTag{
					{
						Tag:          "a",
						SHA:          emptyDigest.String(),
						OS:           "linux",
						Architecture: "amd64",
					},
				},
			}
			repo, err := name.NewRepository(fmt.Sprintf("%s/%s/%s", host, tc.repo, tc.img))
			require.NoError(t, err)

			for _, tag := range tc.wantTags {
				// Lets push the base image....
				imgTag := repo.Tag(tag.Tag)
				err := remote.Write(imgTag, empty.Image)
				require.NoError(t, err)

				subjectDigest := emptyDigest

				payload := map[string]any{
					"_type":         "https://in-toto.io/Statement/v0.1",
					"predicateType": "https://example.com/custom-attestation",
					"predicate":     map[string]string{"builder": "test"},
				}
				payloadBytes, _ := json.Marshal(payload)
				attConfig := &v1.ConfigFile{
					OS:           "unknown",
					Architecture: "unknown",
					Config: v1.Config{
						Labels: map[string]string{
							"dev.cosignproject.attestation": string(payloadBytes),
						},
					},
				}
				attImg, err := mutate.ConfigFile(empty.Image, attConfig)
				require.NoError(t, err)
				attImg = mutate.MediaType(attImg, types.OCIManifestSchema1)

				idx := mutate.AppendManifests(empty.Index,
					mutate.IndexAddendum{
						Add: empty.Image,
						Descriptor: v1.Descriptor{
							MediaType: types.OCIManifestSchema1,
							Platform: &v1.Platform{
								OS:           "linux",
								Architecture: "amd64",
							},
						},
					},
					mutate.IndexAddendum{
						Add: attImg,
						Descriptor: v1.Descriptor{
							MediaType: types.OCIManifestSchema1,
							Annotations: map[string]string{
								"vnd.docker.reference.digest": subjectDigest.String(),
								"vnd.docker.reference.type":   "attestation-manifest",
							},
							Platform: &v1.Platform{
								OS:           "unknown",
								Architecture: "unknown",
							},
						},
					},
				)

				err = remote.WriteIndex(imgTag, idx)
				require.NoError(t, err)

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
			err = remote.Write(repo.Tag("latest"), empty.Image)
			require.NoError(t, err)
			err = remote.Delete(repo.Tag("latest"))
			require.NoError(t, err)

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

			c, err := NewClient(new(Options), anonAuth, logger)
			require.NoError(t, err)

			tc := fn(t, host)

			gotTags, err := c.Tags(ctx, host, tc.repo, tc.img)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Exactly(t, tc.wantTags, gotTags)
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

	c, err := NewClient(new(Options), anonAuth, logger)
	if err != nil {
		t.Fatalf("unexpected error creating client: %s", err)
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo, image := c.RepoImageFromPath(test.path)
			if repo != test.expRepo && image != test.expImage {
				t.Errorf("%s: unexpected repo/image, exp=%s/%s got=%s/%s",
					test.path, test.expRepo, test.expImage, repo, image)
			}
		})
	}
}

func setupRegistry(t *testing.T) string {
	r := httptest.NewServer(registry.New())
	t.Cleanup(r.Close)
	u, err := url.Parse(r.URL)
	if err != nil {
		t.Fatalf("unexpected error parsing registry url: %s", err)
	}
	return u.Host
}
