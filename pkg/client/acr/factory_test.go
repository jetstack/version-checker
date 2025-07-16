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

	handler := new(Factory)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expIs, handler.IsHost(test.host))
		})
	}
}
