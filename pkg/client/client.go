package client

import (
	"context"
	"fmt"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/docker"
	"github.com/jetstack/version-checker/pkg/client/gcr"
	"github.com/jetstack/version-checker/pkg/client/quay"
)

type ImageClient interface {
	// IsClient will return true if this client is appropriate for the given
	// image URL.
	IsClient(imageURL string) bool

	// Tags will return the available tags for the given image URL at the remote
	// repository.
	Tags(ctx context.Context, imageURL string) ([]api.ImageTag, error)
}

// Client is a container image registry client to list tags of given image
// URLs.
type Client struct {
	quay   *quay.Client
	docker *docker.Client
	gcr    *gcr.Client
}

// Options used to configure client authentication.
type Options struct {
	Docker docker.Options
	GCR    gcr.Options
	Quay   quay.Options
}

func New(ctx context.Context, opts Options) (*Client, error) {
	dockerClient, err := docker.New(ctx, opts.Docker)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %s", err)
	}

	return &Client{
		quay:   quay.New(opts.Quay),
		docker: dockerClient,
		gcr:    gcr.New(opts.GCR),
	}, nil
}

func (c *Client) Tags(ctx context.Context, imageURL string) ([]api.ImageTag, error) {
	return c.fromImageURL(imageURL).Tags(ctx, imageURL)
}

// ClientFromImage will return the appropriate registry client for a given
// image URL.
func (c *Client) fromImageURL(imageURL string) ImageClient {
	switch {
	case c.quay.IsClient(imageURL):
		return c.quay
	case c.gcr.IsClient(imageURL):
		return c.gcr
	case c.docker.IsClient(imageURL):
		return c.docker
	default:
		// Fall back to docker if we can't determine the registry
		return c.docker
	}
}
