package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	authn "github.com/google/go-containerregistry/pkg/authn"
	k8sauthn "github.com/google/go-containerregistry/pkg/authn/kubernetes"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"

	"github.com/patrickmn/go-cache"

	"github.com/jetstack/version-checker/pkg/client/acr"
	"github.com/jetstack/version-checker/pkg/client/dockerhub"
	"github.com/jetstack/version-checker/pkg/client/ecr"
	"github.com/jetstack/version-checker/pkg/client/gcr"
	"github.com/jetstack/version-checker/pkg/client/ghcr"
	"github.com/jetstack/version-checker/pkg/client/oci"
	"github.com/jetstack/version-checker/pkg/client/quay"
)

// Used for testing/mocking purposes
type ClientHandler interface {
	Tags(ctx context.Context, imageURL string) ([]api.ImageTag, error)
}

// Client is a container image registry client to list tags of given image
// URLs.
type ClientManager struct {
	keychain  authn.Keychain
	factories []api.ImageClientFactory

	fallbackClient api.ImageClient

	cache *cache.Cache

	log *logrus.Entry
}

// Options used to configure client authentication.
type Options struct {
	ACR    acr.Options
	ECR    ecr.Options
	GCR    gcr.Options
	GHCR   ghcr.Options
	Docker dockerhub.Options
	Quay   quay.Options
	OCI    oci.Options
	// Selfhosted map[string]*selfhosted.Options

	// Kubernetes Authentication Options
	KeyChain            k8sauthn.Options
	AuthRefreshDuration time.Duration

	Transport http.RoundTripper
}

func (m *ClientManager) newClientForHost(host string, authcfg *authn.AuthConfig) (api.ImageClient, error) {
	for _, factory := range m.factories {
		if factory.IsHost(host) {
			return factory.NewClient(authcfg, m.log)
		}
	}
	return nil, fmt.Errorf("no client found for host: %s", host)
}
