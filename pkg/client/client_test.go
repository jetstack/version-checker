package client

import (
	"context"
	"reflect"
	"testing"
)

func TestFromImageURL(t *testing.T) {
	handler, err := New(context.TODO(), Options{})
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		url       string
		expClient ImageClient
		expHost   string
		expPath   string
	}{
		"an empty image URL should be docker": {
			url:       "",
			expClient: handler.docker,
			expHost:   "",
			expPath:   "",
		},
		"single name should be docker": {
			url:       "nginx",
			expClient: handler.docker,
			expHost:   "",
			expPath:   "nginx",
		},
		"two names should be docker": {
			url:       "joshvanl/version-checker",
			expClient: handler.docker,
			expHost:   "",
			expPath:   "joshvanl/version-checker",
		},
		"docker.com should be docker": {
			url:       "docker.com/joshvanl/version-checker",
			expClient: handler.docker,
			expHost:   "docker.com",
			expPath:   "joshvanl/version-checker",
		},
		"docker.io should be docker": {
			url:       "docker.io/joshvanl/version-checker",
			expClient: handler.docker,
			expHost:   "docker.io",
			expPath:   "joshvanl/version-checker",
		},
		"docker.com with sub should be docker": {
			url:       "foo.docker.com/joshvanl/version-checker",
			expClient: handler.docker,
			expHost:   "foo.docker.com",
			expPath:   "joshvanl/version-checker",
		},
		"docker.io with sub should be docker": {
			url:       "bar.docker.io/registry/joshvanl/version-checker",
			expClient: handler.docker,
			expHost:   "bar.docker.io",
			expPath:   "registry/joshvanl/version-checker",
		},

		"gcr.io should be gcr": {
			url:       "gcr.io/jetstack-cre/version-checker",
			expClient: handler.gcr,
			expHost:   "gcr.io",
			expPath:   "jetstack-cre/version-checker",
		},
		"gcr.io with subdomain should be gcr": {
			url:       "us.gcr.io/k8s-artifacts-prod/ingress-nginx/nginx",
			expClient: handler.gcr,
			expHost:   "us.gcr.io",
			expPath:   "k8s-artifacts-prod/ingress-nginx/nginx",
		},

		"quay.io should be quay": {
			url:       "quay.io/jetstack/version-checker",
			expClient: handler.quay,
			expHost:   "quay.io",
			expPath:   "jetstack/version-checker",
		},
		"quay.io with subdomain should be quay": {
			url:       "us.quay.io/k8s-artifacts-prod/ingress-nginx/nginx",
			expClient: handler.quay,
			expHost:   "us.quay.io",
			expPath:   "k8s-artifacts-prod/ingress-nginx/nginx",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client, host, path := handler.fromImageURL(test.url)

			if client != test.expClient {
				t.Errorf("unexpected client, exp=%v got=%v",
					reflect.TypeOf(test.expClient), reflect.TypeOf(client))
			}

			if host != test.expHost {
				t.Errorf("unexpected host, exp=%v got=%v",
					test.expHost, host)
			}

			if path != test.expPath {
				t.Errorf("unexpected path, exp=%s got=%s",
					test.expPath, path)
			}
		})
	}
}
