package ghcr

import (
	"strings"
)

func (c *Client) IsHost(host string) bool {
	// Package API requires Authentication
	// This forces the Client to use the fallback method
	if c.opts.Token == "" {
		return false
	}
	return host == "ghcr.io"
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	var owner, pkg string
	parts := strings.SplitN(path, "/", 2)
	if len(parts) > 0 {
		owner = parts[0]
	}
	if len(parts) > 1 {
		pkg = parts[1]
	}
	return owner, pkg
}
