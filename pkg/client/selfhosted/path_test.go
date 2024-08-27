package selfhosted

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
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
		"docker.repositories.yourdomain.ext should be true": {
			host:  "docker.repositories.yourdomain.ext",
			expIs: true,
		},
		"docker.repositories.yourdomain.ext with a wrong URL should be false": {
			host:  "foo.repositories.yourdomain.ext",
			expIs: false,
		},
		"docker.repositories.yourdomain.ext with a URL and PATH should be false": {
			host:  "docker.repositories.yourdomain.ext/hello-world",
			expIs: false,
		},
		"docker.repositories.yourdomain.ext with sub domain should be true": {
			host:  "foo.docker.repositories.yourdomain.ext",
			expIs: true,
		},
	}

	options := &Options{
		Host: "https://docker.repositories.yourdomain.ext",
	}

	handler, err := New(context.TODO(), logrus.NewEntry(logrus.New()), options)
	if err != nil {
		t.Fatal(err)
	}

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
		"single image should return error": {
			path:     "nginx",
			expRepo:  "",
			expImage: "",
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
