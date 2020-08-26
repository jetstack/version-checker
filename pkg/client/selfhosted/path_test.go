package selfhosted

import (
	"fmt"
	"regexp"
	"testing"
)

const (
	URL        = "https://docker.repositories.yourdomain.ext/artifactory/v2"
	ParsedHost = "docker.repositories.yourdomain.ext"
)

func TestIsHost(t *testing.T) {
	tests := map[string]struct {
		host       string
		expIs      bool
		parsedHost string
	}{
		"an empty host should be false": {
			host:       "",
			expIs:      false,
			parsedHost: ParsedHost,
		},
		"random string should be false": {
			host:       "foobar",
			expIs:      false,
			parsedHost: ParsedHost,
		},
		"random string with dots should be false": {
			host:       "foobar.foo",
			expIs:      false,
			parsedHost: ParsedHost,
		},
		"just docker.io should be false": {
			host:       "docker.io",
			expIs:      false,
			parsedHost: ParsedHost,
		},
		"just docker.com should be false": {
			host:       "docker.com",
			expIs:      false,
			parsedHost: ParsedHost,
		},
		"docker.com with random sub domains should be false": {
			host:       "foo.bar.docker.com",
			expIs:      false,
			parsedHost: ParsedHost,
		},
		"docker.io with random sub domains should be false": {
			host:       "foo.bar.docker.io",
			expIs:      false,
			parsedHost: ParsedHost,
		},
		"docker.repositories.yourdomain.ext should be true": {
			host:       "docker.repositories.yourdomain.ext",
			expIs:      true,
			parsedHost: ParsedHost,
		},
		"docker.repositories.yourdomain.ext should be false": {
			host:       "docker.repositories.yourdomain.ext",
			expIs:      false,
			parsedHost: "docker.repositories.yourdomain.fail",
		},
		"docker.yourdomain.ext with only the root domain parsed should be true": {
			host:       "docker.yourdomain.ext",
			expIs:      true,
			parsedHost: "yourdomain.ext",
		},
		"docker.yourdomain.ext should be true": {
			host:       "docker.yourdomain.ext",
			expIs:      true,
			parsedHost: "docker.yourdomain.ext",
		},
		"docker.repositories.yourdomain.ext/testing should be false": {
			host:       "docker.repositories.yourdomain.ext/testing",
			expIs:      false,
			parsedHost: ParsedHost,
		},
		"docker.repositories.yourdomain.ext/artifactory/v2 should be false": {
			host:       "docker.repositories.yourdomain.ext/artifactory/v2",
			expIs:      false,
			parsedHost: ParsedHost,
		},
		"docker.comfoo should be false": {
			host:       "docker.iofoo",
			expIs:      false,
			parsedHost: ParsedHost,
		},
		"docker.iofoo should be false": {
			host:       "ocker.iofoo",
			expIs:      false,
			parsedHost: ParsedHost,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			handler := new(Client)
			handler.Options.URL = URL
			hostRegex, err := regexp.Compile(fmt.Sprintf(regTemplate, test.parsedHost))
			if err != nil {
				t.Errorf("%s: unexpected error parsedHost=%s", test.host, test.parsedHost)
			}
			handler.hostRegex = hostRegex

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
