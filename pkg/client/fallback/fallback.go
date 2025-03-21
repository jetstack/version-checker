package fallback

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/oci"
	"github.com/jetstack/version-checker/pkg/client/selfhosted"

	"github.com/patrickmn/go-cache"

	"github.com/sirupsen/logrus"
)

type Client struct {
	SelfHosted *selfhosted.Client
	OCI        *oci.Client
	log        *logrus.Entry
	hostCache  *cache.Cache
}

func New(ctx context.Context, log *logrus.Entry, transporter http.RoundTripper) (*Client, error) {
	sh, err := selfhosted.New(ctx, log, &selfhosted.Options{Transporter: transporter})
	if err != nil {
		return nil, err
	}
	oci, err := oci.New(&oci.Options{Transporter: transporter})
	if err != nil {
		return nil, err
	}

	return &Client{
		SelfHosted: sh,
		OCI:        oci,
		hostCache:  cache.New(5*time.Hour, 10*time.Hour),
		log:        log.WithField("client", "fallback"),
	}, nil
}

func (c *Client) Name() string {
	return "fallback"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) (tags []api.ImageTag, err error) {
	// Check if we have a cached client for the host
	if client, found := c.hostCache.Get(host); found {
		c.log.Infof("Found client for host %s in cache", host)
		if tags, err := client.Tags(ctx, host, repo, image); err == nil {
			return tags, nil
		}
	}
	c.log.Debugf("no client for host %s in cache, continuing fallback", host)

	// Try selfhosted client first
	if tags, err := c.SelfHosted.Tags(ctx, host, repo, image); err == nil {
		c.hostCache.SetDefault(host, c.SelfHosted)
		return tags, nil
	}
	c.log.Debug("failed to lookup via SelfHosted, looking up via OCI")

	// Fallback to OCI client
	if tags, err := c.OCI.Tags(ctx, host, repo, image); err == nil {
		c.hostCache.SetDefault(host, c.OCI)
		return tags, nil
	}

	// If both clients fail, return an error
	return nil, fmt.Errorf("failed to fetch tags for host: %s, repo: %s, image: %s", host, repo, image)
}

func (c *Client) IsHost(_ string) bool {
	return true
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	return c.SelfHosted.RepoImageFromPath(path)
}
