package ghcr

import "testing"

func TestIsHost(t *testing.T) {
	tests := map[string]struct {
		token      string
		host       string
		customhost *string
		expIs      bool
	}{
		"an empty token should be false": {
			token: "test-token",
			host:  "",
			expIs: false,
		},
		"an empty host and token should be false": {
			token: "",
			host:  "",
			expIs: false,
		},
		"an empty host  should be false": {
			token: "test-token",
			host:  "",
			expIs: false,
		},
		"random string should be false": {
			token: "test-token",
			host:  "foobar",
			expIs: false,
		},
		"random string with dots should be false": {
			token: "test-token",
			host:  "foobar.foo",
			expIs: false,
		},
		"just ghcr.io should be true": {
			token: "test-token",
			host:  "ghcr.io",
			expIs: true,
		},
		"gcr.io with random sub domains should be false": {
			token: "test-token",
			host:  "ghcr.gcr.io",
			expIs: false,
		},
		"foodghcr.io should be false": {
			token: "test-token",
			host:  "foodghcr.io",
			expIs: false,
		},
		"ghcr.iofoo should be false": {
			token: "test-token",
			host:  "ghcr.iofoo",
			expIs: false,
		},

		// Support for GHE Cloud
		"containers.yourdomain.ghe.com should be true": {
			token: "test-token",
			host:  "containers.yourdomain.ghe.com",
			expIs: true,
		},
		"containers.jetstack.ghe.com should be true": {
			token: "test-token",
			host:  "containers.jetstack.ghe.com",
			expIs: true,
		},
		"customhostname.ghe.internal should be true": {
			token:      "test-token",
			host:       "customhostname.ghe.internal",
			customhost: strPtr("customhostname.ghe.internal"),
			expIs:      true,
		},
		"not-my-customhostname.ghe.internal should be false": {
			token:      "test-token",
			host:       "not-my-customhostname.ghe.internal",
			customhost: strPtr("customhostname.ghe.internal"),
			expIs:      false,
		},
	}

	handler := new(Factory)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.customhost != nil {
				handler.opts.Hostname = *test.customhost
			}
			if isHost := handler.IsHost(test.host); isHost != test.expIs {
				t.Errorf("%s: unexpected IsHost, exp=%t got=%t",
					test.host, test.expIs, isHost)
			}
		})
	}
}

func strPtr(str string) *string {
	return &str
}
