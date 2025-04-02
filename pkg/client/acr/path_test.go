package acr

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
		"just azurecr.io should be false": {
			host:  "azurecr.io",
			expIs: false,
		},
		"azurecr.io with random sub domains should be true": {
			host:  "versionchecker.azurecr.io",
			expIs: true,
		},
		"azurecr.cn with random sub domains should be true": {
			host:  "versionchecker.azurecr.cn",
			expIs: true,
		},
		"azurecr.de with random sub domains should be true": {
			host:  "versionchecker.azurecr.de",
			expIs: true,
		},
		"azurecr.us with random sub domains should be true": {
			host:  "versionchecker.azurecr.us",
			expIs: true,
		},
		"foodazurecr.io should be false": {
			host:  "fooazurecr.io",
			expIs: false,
		},
		"azurecr.iofoo should be false": {
			host:  "azurecr.iofoo",
			expIs: false,
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if isHost := handler.IsHost(test.host); isHost != test.expIs {
				t.Errorf("%s: unexpected IsHost, exp=%t got=%t",
					test.host, test.expIs, isHost)
			}
		})
	}
}

func TestRepoImage(t *testing.T) {
	tests := map[string]struct {
		path              string
		expRepo, expImage string
	}{
		"single image should return google-containers": {
			path:     "kube-scheduler",
			expRepo:  "",
			expImage: "kube-scheduler",
		},
		"two segments to path should return both": {
			path:     "jetstack-cre/version-checker",
			expRepo:  "jetstack-cre",
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
			assert.Equal(t, repo, test.expRepo)
			assert.Equal(t, image, test.expImage)
		})
	}
}
