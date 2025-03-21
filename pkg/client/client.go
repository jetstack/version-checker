package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/acr"
	"github.com/jetstack/version-checker/pkg/client/docker"
	"github.com/jetstack/version-checker/pkg/client/ecr"
	"github.com/jetstack/version-checker/pkg/client/fallback"
	"github.com/jetstack/version-checker/pkg/client/gcr"
	"github.com/jetstack/version-checker/pkg/client/ghcr"
	"github.com/jetstack/version-checker/pkg/client/oci"
	"github.com/jetstack/version-checker/pkg/client/quay"
	"github.com/jetstack/version-checker/pkg/client/selfhosted"
)

// ImageClient represents a image registry client that can list available tags
// for image URLs.
type ImageClient interface {
	// Returns the name of the client
	Name() string

	// IsHost will return true if this client is appropriate for the given
	// host.
	IsHost(host string) bool

	// RepoImage will return the registries repository and image, from a given
	// URL path.
	RepoImageFromPath(path string) (string, string)

	// Tags will return the available tags for the given host, repo, and image
	// using that client.
	Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error)
}

// Client is a container image registry client to list tags of given image
// URLs.
type Client struct {
	clients        []ImageClient
	fallbackClient ImageClient
	log            *logrus.Entry
}

// Options used to configure client authentication.
type Options struct {
	ACR        acr.Options
	ECR        ecr.Options
	GCR        gcr.Options
	GHCR       ghcr.Options
	Docker     docker.Options
	Quay       quay.Options
	OCI        oci.Options
	Selfhosted map[string]*selfhosted.Options
	Transport  http.RoundTripper
}

func New(ctx context.Context, log *logrus.Entry, opts Options) (*Client, error) {
	acrClient, err := acr.New(opts.ACR)
	if err != nil {
		return nil, fmt.Errorf("failed to create acr client: %w", err)
	}
	dockerClient, err := docker.New(ctx, opts.Docker)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	var selfhostedClients []ImageClient
	for _, sOpts := range opts.Selfhosted {
		sClient, err := selfhosted.New(ctx, log, sOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to create selfhosted client %q: %w",
				sOpts.Host, err)
		}

		selfhostedClients = append(selfhostedClients, sClient)
	}

	fallbackClient, err := fallback.New(ctx, log, opts.Transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback client: %w", err)
	}

	c := &Client{
		clients: append(
			selfhostedClients,
			acrClient,
			ecr.New(opts.ECR),
			dockerClient,
			gcr.New(opts.GCR),
			ghcr.New(opts.GHCR),
			quay.New(opts.Quay, log),
		),
		fallbackClient: fallbackClient,
		log:            log.WithField("client", "registry"),
	}

	for _, client := range append(c.clients, fallbackClient) {
		log.WithField("client", client.Name()).Debugf("registered client")
	}

	return c, nil
}

// Tags returns the full list of image tags available, for a given image URL.
func (c *Client) Tags(ctx context.Context, imageURL string) ([]api.ImageTag, error) {
	client, host, path := c.fromImageURL(imageURL)

	c.log.Debugf("using client %q for image URL %q", client.Name(), imageURL)
	repo, image := client.RepoImageFromPath(path)

	return client.Tags(ctx, host, repo, image)
}

// fromImageURL will return the appropriate registry client for a given
// image URL, and the host + path to search.
func (c *Client) fromImageURL(imageURL string) (ImageClient, string, string) {
	var host, path string

	if strings.Contains(imageURL, ".") || strings.Contains(imageURL, ":") {
		split := strings.SplitN(imageURL, "/", 2)
		if len(split) < 2 {
			path = imageURL
		} else {
			host, path = split[0], split[1]
		}
	} else {
		path = imageURL
	}

	for _, client := range c.clients {
		if client.IsHost(host) {
			return client, host, path
		}
	}

	// fall back to selfhosted with no path split
	return c.fallbackClient, host, path
}
