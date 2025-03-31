package fallback

import (
	"context"
	"fmt"
	"time"

	"github.com/jetstack/version-checker/pkg/api"

	"github.com/patrickmn/go-cache"

	"github.com/sirupsen/logrus"
)

type Client struct {
	clients []api.ImageClient

	log       *logrus.Entry
	hostCache *cache.Cache
}

func New(ctx context.Context, log *logrus.Entry, clients []api.ImageClient) (*Client, error) {
	return &Client{
		clients:   clients,
		hostCache: cache.New(5*time.Hour, 10*time.Hour),
		log:       log.WithField("client", "fallback"),
	}, nil
}

func (c *Client) Name() string {
	return "fallback"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) (tags []api.ImageTag, err error) {
	// Check if we have a cached client for the host
	if client, found := c.hostCache.Get(host); found {
		c.log.Infof("Found client for host %s in cache", host)
		if client, ok := client.(api.ImageClient); ok {
			if tags, err := client.Tags(ctx, host, repo, image); err == nil {
				return tags, nil
			}
		} else {
			c.log.Errorf("Unable to fetch from cache for host %s...", host)
		}
	}
	c.log.Debugf("no client for host %s in cache, continuing fallback", host)

	// Try clients, one by one until we have none left..
	for i, client := range c.clients {
		if tags, err := client.Tags(ctx, host, repo, image); err == nil {
			c.hostCache.SetDefault(host, client)
			return tags, nil
		}

		remaining := len(c.clients) - i - 1
		if remaining == 0 {
			c.log.Debugf("failed to lookup via %q, Giving up, no more clients", client.Name())
		} else {
			c.log.Debugf("failed to lookup via %q, continuing to search with %v clients remaining", client.Name(), remaining)
		}
	}

	// If both clients fail, return an error
	return nil, fmt.Errorf("failed to fetch tags for host: %s, repo: %s, image: %s", host, repo, image)
}

func (c *Client) IsHost(_ string) bool {
	return true
}

// Function only added to match ImageClient Interface
func (c *Client) RepoImageFromPath(path string) (string, string) {
	return "", ""
}
