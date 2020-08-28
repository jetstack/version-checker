package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/acr"
	"github.com/jetstack/version-checker/pkg/client/docker"
	"github.com/jetstack/version-checker/pkg/client/ecr"
	"github.com/jetstack/version-checker/pkg/client/gcr"
	"github.com/jetstack/version-checker/pkg/client/quay"
	"github.com/jetstack/version-checker/pkg/client/selfhosted"
)

// ImageClient represents a image registry client that can list available tags
// for image URLs.
type ImageClient interface {
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
	acr        *acr.Client
	ecr        *ecr.Client
	gcr        *gcr.Client
	docker     *docker.Client
	quay       *quay.Client
	selfhosted *selfhosted.Client
}

// Options used to configure client authentication.
type Options struct {
	ACR        acr.Options
	ECR        ecr.Options
	GCR        gcr.Options
	Docker     docker.Options
	Quay       quay.Options
	Selfhosted selfhosted.Options
}

func New(ctx context.Context, log *logrus.Entry, opts Options) (*Client, error) {
	acrClient, err := acr.New(opts.ACR)
	if err != nil {
		return nil, fmt.Errorf("failed to create acr client: %s", err)
	}
	dockerClient, err := docker.New(ctx, opts.Docker)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %s", err)
	}

	selfhostedClient, err := selfhosted.New(ctx, log, opts.Selfhosted)
	if err != nil {
		return nil, fmt.Errorf("failed to create selfhosted client: %s", err)
	}

	return &Client{
		acr:        acrClient,
		ecr:        ecr.New(opts.ECR),
		docker:     dockerClient,
		gcr:        gcr.New(opts.GCR),
		quay:       quay.New(opts.Quay),
		selfhosted: selfhostedClient,
	}, nil
}

// Tags returns the full list of image tags available, for a given image URL.
func (c *Client) Tags(ctx context.Context, imageURL string) ([]api.ImageTag, error) {
	client, host, path := c.fromImageURL(imageURL)
	repo, image := client.RepoImageFromPath(path)
	return client.Tags(ctx, host, repo, image)
}

// fromImageURL will return the appropriate registry client for a given
// image URL, and the host + path to search
func (c *Client) fromImageURL(imageURL string) (ImageClient, string, string) {
	split := strings.SplitN(imageURL, "/", 2)
	if len(split) < 2 {
		return c.docker, "", imageURL
	}

	host, path := split[0], split[1]

	switch {
	case c.acr.IsHost(host):
		return c.acr, host, path
	case c.docker.IsHost(host):
		return c.docker, host, path
	case c.ecr.IsHost(host):
		return c.ecr, host, path
	case c.gcr.IsHost(host):
		return c.gcr, host, path
	case c.quay.IsHost(host):
		return c.quay, host, path
	case c.selfhosted.URL != "" && c.selfhosted.IsHost(host):
		return c.selfhosted, host, path
	}

	// fall back to docker with no path split
	return c.docker, "", imageURL
}
