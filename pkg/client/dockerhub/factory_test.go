package dockerhub

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

	handler := new(Factory)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expIs, handler.IsHost(test.host))
		})
	}
}
