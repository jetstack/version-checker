package client

import (
	"context"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/acr"
	"github.com/jetstack/version-checker/pkg/client/dockerhub"
	"github.com/jetstack/version-checker/pkg/client/ecr"
	"github.com/jetstack/version-checker/pkg/client/gcr"
	"github.com/jetstack/version-checker/pkg/client/ghcr"
	"github.com/jetstack/version-checker/pkg/client/oci"
	"github.com/jetstack/version-checker/pkg/client/quay"
)

func TestFromImageURL(t *testing.T) {
	handler, err := NewManager(context.TODO(), logrus.NewEntry(logrus.New()), nil, Options{
		// Selfhosted: map[string]*selfhosted.Options{
		// 	"yourdomain": {
		// 		Host: "https://docker.repositories.yourdomain.com",
		// 	},
		// },
		GHCR: ghcr.Options{
			Token: "test-token",
		},
	})
	require.NoError(t, err)

	tests := map[string]struct {
		url       string
		expClient api.ImageClient
		expHost   string
		expPath   string
	}{
		"an empty image URL should be nil": {
			url:       "",
			expClient: nil,
			expHost:   "",
			expPath:   "",
		},
		"single name should be docker": {
			url:       "nginx",
			expClient: new(dockerhub.Client),
			expHost:   name.DefaultRegistry,
			expPath:   "library/nginx",
		},
		"two names should be docker": {
			url:       "joshvanl/version-checker",
			expClient: new(dockerhub.Client),
			expHost:   name.DefaultRegistry,
			expPath:   "joshvanl/version-checker",
		},
		"three names should be docker": {
			url:       "jetstack/joshvanl/version-checker",
			expClient: new(dockerhub.Client),
			expHost:   name.DefaultRegistry,
			expPath:   "jetstack/joshvanl/version-checker",
		},
		"docker.com should be docker": {
			url:       "docker.com/joshvanl/version-checker",
			expClient: new(dockerhub.Client),
			expHost:   "docker.com",
			expPath:   "joshvanl/version-checker",
		},
		"docker.io should be docker": {
			url:       "docker.io/joshvanl/version-checker",
			expClient: new(dockerhub.Client),
			expHost:   name.DefaultRegistry,
			expPath:   "joshvanl/version-checker",
		},
		"docker.com with sub should be docker": {
			url:       "foo.docker.com/joshvanl/version-checker",
			expClient: new(dockerhub.Client),
			expHost:   "foo.docker.com",
			expPath:   "joshvanl/version-checker",
		},
		"docker.io with sub should be docker": {
			url:       "bar.docker.io/registry/joshvanl/version-checker",
			expClient: new(dockerhub.Client),
			expHost:   "bar.docker.io",
			expPath:   "registry/joshvanl/version-checker",
		},

		// ACR
		"versionchecker.azurecr.io should be acr": {
			url:       "versionchecker.azurecr.io/jetstack-cre/version-checker",
			expClient: new(acr.Client),
			expHost:   "versionchecker.azurecr.io",
			expPath:   "jetstack-cre/version-checker",
		},
		"versionchecker.azurecr.io with single path should be acr": {
			url:       "versionchecker.azurecr.io/version-checker",
			expClient: new(acr.Client),
			expHost:   "versionchecker.azurecr.io",
			expPath:   "version-checker",
		},

		// ECR
		"123.dkr.foo.amazon.com should be ecr": {
			url:       "123.dkr.ecr.foo.amazonaws.com/version-checker",
			expClient: new(ecr.Client),
			expHost:   "123.dkr.ecr.foo.amazonaws.com",
			expPath:   "version-checker",
		},
		"hello.dkr.eu-west-1.amazon.com.cn should be ecr": {
			url:       "hello.dkr.ecr.eu-west-1.amazonaws.com.cn/jetstack/joshvanl/version-checker",
			expClient: new(ecr.Client),
			expHost:   "hello.dkr.ecr.eu-west-1.amazonaws.com.cn",
			expPath:   "jetstack/joshvanl/version-checker",
		},

		// GCR / GAR
		"gcr.io should be gcr": {
			url:       "gcr.io/jetstack-cre/version-checker",
			expClient: new(gcr.Client),
			expHost:   "gcr.io",
			expPath:   "jetstack-cre/version-checker",
		},
		"gcr.io with subdomain should be gcr": {
			url:       "us.gcr.io/k8s-artifacts-prod/ingress-nginx/nginx",
			expClient: new(gcr.Client),
			expHost:   "us.gcr.io",
			expPath:   "k8s-artifacts-prod/ingress-nginx/nginx",
		},
		"k8s.io should be gcr": {
			url:       "k8s.io/sig-storage/csi-node-driver-registrar",
			expClient: new(gcr.Client),
			expHost:   "k8s.io",
			expPath:   "sig-storage/csi-node-driver-registrar",
		},
		"k8s.io with subdomain should be gcr": {
			url:       "registry.k8s.io/sig-storage/csi-node-driver-registrar",
			expClient: new(gcr.Client),
			expHost:   "registry.k8s.io",
			expPath:   "sig-storage/csi-node-driver-registrar",
		},

		// GHCR
		"ghcr.io should be ghcr": {
			url:       "ghcr.io/jetstack/version-checker",
			expClient: new(ghcr.Client),
			expHost:   "ghcr.io",
			expPath:   "jetstack/version-checker",
		},
		"gcr.io with subdomain should be ghcr": {
			url:       "ghcr.io/k8s-artifacts-prod/ingress-nginx/nginx",
			expClient: new(ghcr.Client),
			expHost:   "ghcr.io",
			expPath:   "k8s-artifacts-prod/ingress-nginx/nginx",
		},

		// QUAY
		"quay.io should be quay": {
			url:       "quay.io/jetstack/version-checker",
			expClient: new(quay.Client),
			expHost:   "quay.io",
			expPath:   "jetstack/version-checker",
		},
		"quay.io with subdomain should be quay": {
			url:       "us.quay.io/k8s-artifacts-prod/ingress-nginx/nginx",
			expClient: new(quay.Client),
			expHost:   "us.quay.io",
			expPath:   "k8s-artifacts-prod/ingress-nginx/nginx",
		},

		// OCI / Self Hosted
		"selfhosted should be selfhosted": {
			url:       "docker.repositories.yourdomain.com/ingress-nginx/nginx",
			expClient: new(oci.Client),
			expHost:   "docker.repositories.yourdomain.com",
			expPath:   "ingress-nginx/nginx",
		},
		"selfhosted with different domain should be fallback": {
			url:       "registry.opensource.zalan.do/teapot/external-dns",
			expClient: new(oci.Client),
			expHost:   "registry.opensource.zalan.do",
			expPath:   "teapot/external-dns",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			client, host, path := handler.fromImageURL(test.url)

			if test.expClient != nil {
				require.NotNil(t, client)
			}
			require.IsType(t, test.expClient, client)
			assert.Equal(t, test.expHost, host)
			assert.Equal(t, test.expPath, path)
		})
	}
}
