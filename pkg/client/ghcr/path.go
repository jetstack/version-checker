package ghcr

import (
	"regexp"
	"strings"
)

const (
	HostRegTempl = `^(containers\.[a-zA-Z0-9-]+\.ghe\.com|ghcr\.io)$`
)

var HostReg = regexp.MustCompile(HostRegTempl)

func (c *Client) IsHost(host string) bool {
	// Package API requires Authentication
	// This forces the Client to use the fallback method
	if c.opts.Token == "" {
		return false
	}
	// If we're using a custom hostname.
	if c.opts.Hostname != "" && c.opts.Hostname == host {
		return true
	}
	return HostReg.MatchString(host)
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
