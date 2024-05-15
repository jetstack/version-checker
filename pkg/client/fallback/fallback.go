package fallback

import (
	"context"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/oci"
	"github.com/jetstack/version-checker/pkg/client/selfhosted"
	"github.com/sirupsen/logrus"
)

type Client struct {
	SelfHosted *selfhosted.Client
	OCI        *oci.Client
}

func New(ctx context.Context, log *logrus.Entry) (*Client, error) {
	sh, err := selfhosted.New(ctx, log, new(selfhosted.Options))
	if err != nil {
		return nil, err
	}
	oci, err := oci.New()
	if err != nil {
		return nil, err
	}

	return &Client{
		SelfHosted: sh,
		OCI:        oci,
	}, nil
}

func (c *Client) Name() string {
	return "fallback"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	// TODO: Cache selfhosted/oci by host
	if tags, err := c.SelfHosted.Tags(ctx, host, repo, image); err == nil {
		return tags, err
	}
	return c.OCI.Tags(ctx, host, repo, image)
}

func (c *Client) IsHost(host string) bool {
	return true
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	return c.SelfHosted.RepoImageFromPath(path)
}
