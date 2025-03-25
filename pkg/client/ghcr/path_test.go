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

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.token != "" {
				handler.opts.Token = test.token
			}
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

func TestRepoImage(t *testing.T) {
	tests := map[string]struct {
		path              string
		expRepo, expImage string
	}{
		"empty path should be interpreted as an empty repo and image": {
			path:     "",
			expRepo:  "",
			expImage: "",
		},
		"one segement should be interpreted as 'repo'": {
			path:     "jetstack-cre",
			expRepo:  "jetstack-cre",
			expImage: "",
		},
		"two segments to path should return both": {
			path:     "jetstack-cre/version-checker",
			expRepo:  "jetstack-cre",
			expImage: "version-checker",
		},
		"multiple segments to path should return first segment in repo, rest in image": {
			path:     "k8s-artifacts-prod/ingress-nginx/nginx",
			expRepo:  "k8s-artifacts-prod",
			expImage: "ingress-nginx/nginx",
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			handler.opts.Token = "fake-token"
			repo, image := handler.RepoImageFromPath(test.path)
			if repo != test.expRepo && image != test.expImage {
				t.Errorf("%s: unexpected repo/image, exp=%s/%s got=%s/%s",
					test.path, test.expRepo, test.expImage, repo, image)
			}
		})
	}
}
