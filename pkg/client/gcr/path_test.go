package gcr

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
		"just gcr.io should be true": {
			host:  "gcr.io",
			expIs: true,
		},
		"gcr.io with random sub domains should be true": {
			host:  "k8s.gcr.io",
			expIs: true,
		},
		"foodgcr.io should be false": {
			host:  "foogcr.io",
			expIs: false,
		},
		"gcr.iofoo should be false": {
			host:  "gcr.iofoo",
			expIs: false,
		},
		"just pkg.dev should be false": {
			host:  "pkg.dev",
			expIs: false,
		},
		"docker.pkg.dev with random sub domains should be false": {
			host:  "docker.pkg.dev",
			expIs: false,
		},
		"eu-docker.pkg.dev with random sub domains should be true": {
			host:  "eu-docker.pkg.dev",
			expIs: true,
		},
		"k8s.io should be true": {
			host:  "k8s.io",
			expIs: true,
		},
		"registry.k8s.io should be true": {
			host:  "registry.k8s.io",
			expIs: true,
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
		"single image should return google-containers": {
			path:     "kube-scheduler",
			expRepo:  "google-containers",
			expImage: "kube-scheduler",
		},
		"two segments to path should return both": {
			path:     "jetstack-cre/version-checker",
			expRepo:  "jetstack-cre",
			expImage: "version-checker",
		},
		"multiple segments to path should return all in repo, last segment image": {
			path:     "k8s-artifacts-prod/ingress-nginx/nginx",
			expRepo:  "k8s-artifacts-prod/ingress-nginx",
			expImage: "nginx",
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
