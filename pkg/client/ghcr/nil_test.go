package ghcr

import (
	"encoding/json"
	"testing"

	"github.com/google/go-github/v70/github"
	"github.com/stretchr/testify/assert"
)

// The GitHub API can omit created_at, metadata or metadata.container.
func TestExtractImageTagsToleratesMissingFields(t *testing.T) {
	c := &Client{}
	versions := []*github.PackageVersion{
		{Name: github.Ptr("no-metadata")},
		{Name: github.Ptr("null-metadata"), Metadata: json.RawMessage(`null`)},
		{Name: github.Ptr("no-container"), Metadata: json.RawMessage(`{}`)},
		{Name: github.Ptr("no-created-at"), Metadata: json.RawMessage(`{"container":{"tags":["v1"]}}`)},
	}
	var tags []string
	assert.NotPanics(t, func() {
		for _, tag := range c.extractImageTags(versions) {
			tags = append(tags, tag.Tag)
		}
	})
	assert.Equal(t, []string{"v1"}, tags)
}
