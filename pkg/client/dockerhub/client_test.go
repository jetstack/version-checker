package dockerhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoImage(t *testing.T) {
	tests := map[string]struct {
		path              string
		expRepo, expImage string
	}{
		"single image should return library": {
			path:     "nginx",
			expRepo:  "library",
			expImage: "nginx",
		},
		"two segments to path should return both": {
			path:     "joshvanl/version-checker",
			expRepo:  "joshvanl",
			expImage: "version-checker",
		},
		"multiple segments to path should return last two": {
			path:     "registry/joshvanl/version-checker",
			expRepo:  "joshvanl",
			expImage: "version-checker",
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo, image := handler.RepoImageFromPath(test.path)
			assert.Equal(t, test.expImage, image)
			assert.Equal(t, test.expRepo, repo)
		})
	}
}
