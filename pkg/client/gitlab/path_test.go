package gitlab

import (
	"testing"
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
		"just docker.io should be false": {
			host:  "docker.io",
			expIs: false,
		},
		"just docker.com should be false": {
			host:  "docker.com",
			expIs: false,
		},
		"docker.com with random sub domains should be false": {
			host:  "foo.bar.docker.com",
			expIs: false,
		},
		"docker.io with random sub domains should be false": {
			host:  "foo.bar.docker.io",
			expIs: false,
		},
		"docker.comfoo should be false": {
			host:  "docker.iofoo",
			expIs: false,
		},
		"docker.iofoo should be false": {
			host:  "ocker.iofoo",
			expIs: false,
		},
		"gitlab.com should be true": {
			host:  "gitlab.com",
			expIs: true,
		},
		"gitlab.com with subdomains should be false": {
			host:  "sub.gitlab.com",
			expIs: false,
		},
		"gitlab.com with path should be false": {
			host:  "gitlab.com/path",
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

func TestRepoImageFromPath(t *testing.T) {
	tests := map[string]struct {
		path              string
		expRepo, expImage string
	}{
		"single image should return the image": {
			path:     "nginx",
			expRepo:  "nginx",
			expImage: "",
		},
		"two segments to path should return both": {
			path:     "joshvanl/version-checker",
			expRepo:  "joshvanl",
			expImage: "version-checker",
		},
		"multiple segments to path should return last two": {
			path:     "registry/joshvanl/version-checker",
			expRepo:  "registry/joshvanl",
			expImage: "version-checker",
		},
		"path with trailing slash should return empty image": {
			path:     "joshvanl/version-checker/",
			expRepo:  "joshvanl/version-checker",
			expImage: "",
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo, image := handler.RepoImageFromPath(test.path)
			if repo != test.expRepo || image != test.expImage {
				t.Errorf("%s: unexpected repo/image, exp=%s/%s got=%s/%s",
					test.path, test.expRepo, test.expImage, repo, image)
			}
		})
	}
}
