package selfhosted

import "testing"

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
		"docker.repositories.yourdomain.ext should be true": {
			host:  "docker.repositories.yourdomain.ext",
			expIs: true,
		},
		"docker.repositories.yourdomain.ext/testing should be false": {
			host:  "docker.repositories.yourdomain.ext/testing",
			expIs: false,
		},
		"docker.repositories.yourdomain.ext/artifactory/v2 should be false": {
			host:  "docker.repositories.yourdomain.ext/artifactory/v2",
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
	handler.Options.URL = "https://docker.repositories.yourdomain.ext/artifactory/v2"
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
		expError          bool
	}{
		"single image should return error": {
			path:     "nginx",
			expRepo:  "",
			expImage: "",
			expError: true,
		},
		"two segments to path should return both": {
			path:     "joshvanl/version-checker",
			expRepo:  "joshvanl",
			expImage: "version-checker",
			expError: false,
		},
		"multiple segments to path should return last two": {
			path:     "registry/joshvanl/version-checker",
			expRepo:  "joshvanl",
			expImage: "version-checker",
			expError: false,
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			repo, image, err := handler.RepoImageFromPath(test.path)

			if test.expError && err == nil {
				t.Errorf("%s: Expected error, exp=%s/%s/error got=%s/%s/nil",
					test.path, test.expRepo, test.expImage, repo, image)
			} else if repo != test.expRepo && image != test.expImage {
				t.Errorf("%s: unexpected repo/image, exp=%s/%s got=%s/%s",
					test.path, test.expRepo, test.expImage, repo, image)
			}

		})
	}
}
