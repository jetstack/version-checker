package client

import (
	"context"
	"strings"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"

	authn "github.com/google/go-containerregistry/pkg/authn"
	k8sauthn "github.com/google/go-containerregistry/pkg/authn/kubernetes"
	"github.com/google/go-containerregistry/pkg/name"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/acr"
	"github.com/jetstack/version-checker/pkg/client/dockerhub"
	"github.com/jetstack/version-checker/pkg/client/ecr"
	"github.com/jetstack/version-checker/pkg/client/gcr"
	"github.com/jetstack/version-checker/pkg/client/ghcr"
	"github.com/jetstack/version-checker/pkg/client/oci"
	"github.com/jetstack/version-checker/pkg/client/quay"
)

// ClientManager is a container image registry client manager to list tags of
// given image URLs.
func NewManager(ctx context.Context, log *logrus.Entry, k8sconfig *rest.Config, opts Options) (*ClientManager, error) {
	log = log.WithField("component", "client")

	// Create a list with a DefaultKeychain and prepend/append later.
	var keychains = []authn.Keychain{authn.DefaultKeychain}

	// Setup Transporters for all remaining clients (if one is set)
	if opts.Transport != nil {
		opts.Quay.Transporter = opts.Transport
		opts.ECR.Transporter = opts.Transport
		opts.GHCR.Transporter = opts.Transport
		opts.GCR.Transporter = opts.Transport
		log.Debug("registered custom Transport for Clients")
	}

	if k8sconfig != nil && opts.KeyChain.ServiceAccountName != "" && opts.KeyChain.Namespace != "" {
		log.WithField("Keychain", map[string]interface{}{
			"KeychainNamespace":          opts.KeyChain.Namespace,
			"KeychainServiceAccountName": opts.KeyChain.ServiceAccountName,
			"KeychainRefreshDuration":    opts.AuthRefreshDuration,
		}).Infof("Collecting Credentials")

		k8sclient, err := kubernetes.NewForConfig(k8sconfig)
		if err != nil {
			return nil, err
		}
		log.WithField("client", k8sclient).Debug("Successfully Created K8S Client")

		kc, err := k8sauthn.New(ctx, k8sclient, opts.KeyChain)
		if err != nil {
			return nil, err
		}
		log.WithField("opts", opts.KeyChain).Debug("Successfully Created K8S Keychain")
		// Prepend the Refreshing Keychain to list of keychains
		// We want a RefreshingKeychain as credentials in cluster could have rotated.
		keychains = append([]authn.Keychain{authn.RefreshingKeychain(kc, opts.AuthRefreshDuration)}, keychains...)
	}

	var factories = []api.ImageClientFactory{
		&dockerhub.Factory{},
		acr.NewFactory(opts.ACR),
		dockerhub.NewFactory(opts.Docker),
		ecr.NewFactory(opts.ECR),
		gcr.NewFactory(opts.GCR),
		ghcr.NewFactory(opts.GHCR),
		quay.NewFactory(opts.Quay),
		oci.NewFactory(opts.OCI),
		// &selfhosted.Factory{},
	}

	for _, factory := range factories {
		keychains = append(keychains, factory)
	}

	manager := &ClientManager{
		keychain:  authn.NewMultiKeychain(keychains...),
		cache:     cache.New(opts.AuthRefreshDuration, 3*opts.AuthRefreshDuration),
		log:       log,
		factories: factories,
	}

	for _, factory := range manager.factories {
		log.WithField("factory", factory.Name()).Debugf("registered factories")
	}

	return manager, nil
}

// Tags returns the full list of image tags available, for a given image URL.
func (c *ClientManager) Tags(ctx context.Context, imageURL string) ([]api.ImageTag, error) {
	client, host, path := c.fromImageURL(imageURL)

	c.log.Debugf("using client %q for image URL %q", client.Name(), imageURL)
	repo, image := client.RepoImageFromPath(path)

	return client.Tags(ctx, host, repo, image)
}

// fromImageURL will return the appropriate registry client for a given
// image URL, and the host + path to search.
func (c *ClientManager) fromImageURL(imageURL string) (api.ImageClient, string, string) {
	var host, path string

	if imageURL == "" {
		return nil, "", ""
	}

	repo, err := name.NewRepository(imageURL, name.WeakValidation)
	if err != nil {
		c.log.Errorf("parsing repository: %s", err)
		return nil, host, path
	}
	host = repo.RegistryStr()
	path = strings.TrimPrefix(
		strings.TrimPrefix(repo.String(), host),
		"/",
	)

	auth, err := c.keychain.Resolve(repo)
	if err != nil {
		c.log.Errorf("Failed to resolve keychain for %q: %s", host, err)
		return nil, host, path
	}
	authconfig, err := auth.Authorization()
	if err != nil {
		c.log.Errorf("Failed to resolve keychain for %q: %s", host, err)
		return nil, host, path
	}

	// Check if we have a cached client for this host
	if cl, ok := c.cache.Get(host); ok {
		c.log.Debugf("Using cached client for host %q", host)
		return cl.(api.ImageClient), host, path
	}

	cl, err := c.newClientForHost(host, authconfig)
	if err != nil {
		c.log.Errorf("Failed to create client for host %q: %v", host, err)
		// We don't return an error here, as we want to fall back to the
		// fallback client if no specific client is found.
	}

	if cl != nil {
		c.log.Debugf("Found client %q for host %q", cl.Name(), host)
		c.cache.SetDefault(host, cl)
		return cl, host, path
	}

	// fall back to the fallback client if no specific client is found
	return c.fallbackClient, host, path
}

// func setupSelfHosted(ctx context.Context, log *logrus.Entry, opts Options) ([]api.ImageClient, error) {
// 	var selfhostedClients []api.ImageClient

// 	for _, sOpts := range opts.Selfhosted {
// 		if keychain != nil {
// 			repo, _ := name.NewRepository("fake/image", name.WeakValidation, name.WithDefaultRegistry(sOpts.Host))
// 			log.Debug("Repo:", repo)

// 			kcauth, err := keychain.Resolve(repo)
// 			log.Debug("kcauth:", kcauth)

// 			if kcauth == authn.Anonymous {
// 				log.Warnf("Using Anonymous Authentication for host %s", sOpts.Host)
// 			}
// 			if err == nil && kcauth != authn.Anonymous {
// 				authconfig, err := kcauth.Authorization()
// 				if err == nil {
// 					log.WithFields(logrus.Fields{
// 						"client": "selfhosted",
// 						"host":   sOpts.Host,
// 						"config": authconfig,
// 					}).Infof("Using Authentication from keychain")

// 					sOpts.Username = authconfig.Username
// 					sOpts.Password = authconfig.Password
// 					sOpts.Bearer = authconfig.RegistryToken

// 					// TODO: Remove Output when done
// 					// spew.Dump()
// 					// spew.Dump(authconfig)
// 					// spew.Dump(sOpts)
// 				} else {
// 					log.Errorf("Unable to retrieve authentication for host %s: %v", sOpts.Host, err)
// 				}
// 			} else {
// 				log.Errorf("Unable to resolve credentials for host %s: %v", sOpts.Host, err)
// 			}
// 		} else {
// 			log.Infof("Not using keychain for selfhosted client %s", sOpts.Host)
// 		}
// 		// If we don't have a prefix, lets assume https
// 		if !strings.HasPrefix(sOpts.Host, "http://") || !strings.HasPrefix(sOpts.Host, "https://") {
// 			sOpts.Host = "https://" + sOpts.Host
// 		}

// 		sClient, err := selfhosted.New(ctx, log, sOpts)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to create selfhosted client %q: %w",
// 				sOpts.Host, err)
// 		}

// 		selfhostedClients = append(selfhostedClients, sClient)
// 	}

// 	return selfhostedClients, nil
// }
