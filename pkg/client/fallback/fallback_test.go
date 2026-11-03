package fallback

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jetstack/version-checker/pkg/api"
)

// stubClient is a controllable api.ImageClient for exercising fallback behaviour.
type stubClient struct {
	name  string
	tags  []api.ImageTag
	err   error
	calls int
}

func (s *stubClient) Name() string { return s.name }

func (s *stubClient) IsHost(_ string) bool { return true }

func (s *stubClient) RepoImageFromPath(_ string) (string, string) { return "", "" }

func (s *stubClient) Tags(_ context.Context, _, _, _ string) ([]api.ImageTag, error) {
	s.calls++
	return s.tags, s.err
}

func newTestClient(t *testing.T, clients ...api.ImageClient) *Client {
	t.Helper()
	c, err := New(context.Background(), logrus.NewEntry(logrus.New()), clients)
	require.NoError(t, err)
	return c
}

func TestName(t *testing.T) {
	c := newTestClient(t)
	assert.Equal(t, "fallback", c.Name())
}

func TestIsHost(t *testing.T) {
	c := newTestClient(t)
	assert.True(t, c.IsHost("anything.example.com"))
}

func TestRepoImageFromPath(t *testing.T) {
	c := newTestClient(t)
	repo, image := c.RepoImageFromPath("some/path")
	assert.Empty(t, repo)
	assert.Empty(t, image)
}

func TestTagsFallsBackToNextClient(t *testing.T) {
	want := []api.ImageTag{{Tag: "v1.2.3", SHA: "sha256:abc"}}
	first := &stubClient{name: "first", err: errors.New("boom")}
	second := &stubClient{name: "second", tags: want}

	c := newTestClient(t, first, second)
	tags, err := c.Tags(context.Background(), "example.com", "repo", "image")

	require.NoError(t, err)
	assert.Equal(t, want, tags)
	assert.Equal(t, 1, first.calls)
	assert.Equal(t, 1, second.calls)
}

func TestTagsCachesSuccessfulClient(t *testing.T) {
	want := []api.ImageTag{{Tag: "v1.2.3", SHA: "sha256:abc"}}
	first := &stubClient{name: "first", err: errors.New("boom")}
	second := &stubClient{name: "second", tags: want}

	c := newTestClient(t, first, second)

	_, err := c.Tags(context.Background(), "example.com", "repo", "image")
	require.NoError(t, err)

	_, err = c.Tags(context.Background(), "example.com", "repo", "image")
	require.NoError(t, err)
	assert.Equal(t, 1, first.calls)
	assert.Equal(t, 2, second.calls)
}

func TestTagsCacheMissFallsBackAgain(t *testing.T) {
	want := []api.ImageTag{{Tag: "v9.9.9", SHA: "sha256:def"}}
	first := &stubClient{name: "first", err: errors.New("boom")}
	second := &stubClient{name: "second", tags: []api.ImageTag{{Tag: "v1.0.0", SHA: "sha256:aaa"}}}

	c := newTestClient(t, first, second)

	_, err := c.Tags(context.Background(), "example.com", "repo", "image")
	require.NoError(t, err)

	second.err = errors.New("boom")
	first.err = nil
	first.tags = want

	tags, err := c.Tags(context.Background(), "example.com", "repo", "image")
	require.NoError(t, err)
	assert.Equal(t, want, tags)
	assert.Equal(t, 2, second.calls)
	assert.Equal(t, 2, first.calls)
}

func TestTagsAllClientsFail(t *testing.T) {
	first := &stubClient{name: "first", err: errors.New("boom")}
	second := &stubClient{name: "second", err: errors.New("bang")}

	c := newTestClient(t, first, second)
	tags, err := c.Tags(context.Background(), "example.com", "repo", "image")

	require.Error(t, err)
	assert.Nil(t, tags)
	assert.Contains(t, err.Error(), "failed to fetch tags")
}
