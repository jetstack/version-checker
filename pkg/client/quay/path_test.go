package quay

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHost(t *testing.T) {
	tests := map[string]struct {
		host  string
		expIs bool
	}{
		"an empty host should be false": {
			host:  "",
			expIs: false,
		},
		"random string should be false": {
			host:  "foobar",
			expIs: false,
		},
		"random string with dots should be false": {
			host:  "foobar.foo",
			expIs: false,
		},
		"just quay.io should be true": {
			host:  "quay.io",
			expIs: true,
		},
		"quay.io with random sub domains should be true": {
			host:  "k8s.quay.io",
			expIs: true,
		},
		"foodquay.io should be false": {
			host:  "fooquay.io",
			expIs: false,
		},
		"quay.iofoo should be false": {
			host:  "quay.iofoo",
			expIs: false,
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expIs,
				handler.IsHost(test.host),
			)
		})
	}
}

func TestRepoImage(t *testing.T) {
	tests := map[string]struct {
		path              string
		expRepo, expImage string
	}{
		"single image should return empty image": {
			path:     "version-checker",
			expRepo:  "version-checker",
			expImage: "",
		},
		"two segments to path should return both": {
			path:     "jetstack/version-checker",
			expRepo:  "jetstack",
			expImage: "version-checker",
		},
		"multiple segments to path should return all in repo, last segment image": {
			path:     "k8s-artifacts-prod/ingress-nginx/nginx",
			expRepo:  "k8s-artifacts-prod/ingress-nginx",
			expImage: "nginx",
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo, image := handler.RepoImageFromPath(test.path)
			assert.Equal(t, test.expRepo, repo)
			assert.Equal(t, test.expImage, image)
		})
	}
}
