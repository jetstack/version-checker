package docker

import "testing"

func TestIsHost(t *testing.T) {
	tests := map[string]struct {
		host  string
		expIs bool
	}{
		"an empty host should be false": {
			host:  "",
			expIs: true,
		},
		"random string should be false": {
			host:  "foobar",
			expIs: false,
		},
		"path with two segments should be false": {
			host:  "joshvanl/version-checker",
			expIs: false,
		},
		"path with three segments should be false": {
			host:  "jetstack/joshvanl/version-checker",
			expIs: false,
		},
		"random string with dots should be false": {
			host:  "foobar.foo",
			expIs: false,
		},
		"just docker.io should be true": {
			host:  "docker.io",
			expIs: true,
		},
		"just docker.com should be true": {
			host:  "docker.com",
			expIs: true,
		},
		"docker.com with random sub domains should be true": {
			host:  "foo.bar.docker.com",
			expIs: true,
		},
		"docker.io with random sub domains should be true": {
			host:  "foo.bar.docker.io",
			expIs: true,
		},
		"foodocker.com should be false": {
			host:  "foodocker.com",
			expIs: false,
		},
		"foodocker.io should be false": {
			host:  "foodocker.io",
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
			if repo != test.expRepo && image != test.expImage {
				t.Errorf("%s: unexpected repo/image, exp=%s/%s got=%s/%s",
					test.path, test.expRepo, test.expImage, repo, image)
			}
		})
	}
}
