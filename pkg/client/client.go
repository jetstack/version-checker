package client

import (
	"context"

	"github.com/joshvanl/version-checker/pkg/api"
	"github.com/joshvanl/version-checker/pkg/client/docker"
	"github.com/joshvanl/version-checker/pkg/client/gcr"
	"github.com/joshvanl/version-checker/pkg/client/quay"
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
	GCRAccessToken string `json:"gcr_access_token"`
}

func New(opts *Options) *Client {
	return &Client{
		quay:   quay.New(),
		docker: docker.New(),
		gcr:    gcr.New(opts.GCRAccessToken),
	}
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
