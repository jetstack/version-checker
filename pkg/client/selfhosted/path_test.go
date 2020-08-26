package selfhosted

import (
	"testing"
)

const (
	URL = "https://docker.repositories.yourdomain.ext/"
)

func TestIsHost(t *testing.T) {
	tests := map[string]struct {
		host   string
		expIs  bool
		URL    string
		expErr bool
	}{
		"an empty host should be false": {
			host:   "",
			expIs:  false,
			URL:    URL,
			expErr: false,
		},
		"random string should be false": {
			host:   "foobar",
			expIs:  false,
			URL:    URL,
			expErr: false,
		},
		"random string with dots should be false": {
			host:   "foobar.foo",
			expIs:  false,
			URL:    URL,
			expErr: false,
		},
		"just docker.io should be false": {
			host:   "docker.io",
			expIs:  false,
			URL:    URL,
			expErr: false,
		},
		"just docker.com should be false": {
			host:   "docker.com",
			expIs:  false,
			URL:    URL,
			expErr: false,
		},
		"docker.com with random sub domains should be false": {
			host:   "foo.bar.docker.com",
			expIs:  false,
			URL:    URL,
			expErr: false,
		},
		"docker.io with random sub domains should be false": {
			host:   "foo.bar.docker.io",
			expIs:  false,
			URL:    URL,
			expErr: false,
		},
		"docker.comfoo should be false": {
			host:   "docker.iofoo",
			expIs:  false,
			URL:    URL,
			expErr: false,
		},
		"docker.iofoo should be false": {
			host:   "ocker.iofoo",
			expIs:  false,
			URL:    URL,
			expErr: false,
		},
		"docker.repositories.yourdomain.ext should be true": {
			host:   "docker.repositories.yourdomain.ext",
			expIs:  true,
			URL:    URL,
			expErr: false,
		},
		"docker.repositories.yourdomain.ext with a wrong URL should be false": {
			host:   "docker.repositories.yourdomain.ext",
			expIs:  false,
			URL:    "http://something.wrong",
			expErr: false,
		},
		"docker.repositories.yourdomain.ext with a URL and PATH should be true": {
			host:   "docker.repositories.yourdomain.ext",
			expIs:  true,
			URL:    URL + "/artifactory",
			expErr: false,
		},
		"docker.repositories.yourdomain.ext with a bad URL should be error": {
			host:   "docker.repositories.yourdomain.ext",
			expIs:  false,
			URL:    "something.bad",
			expErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			options := &Options{URL: test.URL}
			regex, parsedURL, err := parseURL(*options)

			if err != nil {
				if !test.expErr {
					t.Errorf("%s: unexpected parseErr got=%s exp=%t", test.host, err, test.expErr)
				}
			} else {
				handler := &Client{
					Options:   *options,
					parsedURL: parsedURL,
					hostRegex: regex,
				}

				if isHost := handler.IsHost(test.host); isHost != test.expIs {
					t.Errorf("%s: unexpected IsHost, exp=%t got=%t",
						test.host, test.expIs, isHost)
				}
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
