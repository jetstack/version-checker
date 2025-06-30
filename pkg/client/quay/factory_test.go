package quay

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

	handler := new(Factory)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if isHost := handler.IsHost(test.host); isHost != test.expIs {
				t.Errorf("%s: unexpected IsHost, exp=%t got=%t",
					test.host, test.expIs, isHost)
			}
		})
	}
}
