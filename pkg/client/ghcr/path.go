package ghcr

import (
	"regexp"
	"strings"
)

var (
	reg = regexp.MustCompile(`^ghcr.io$`)
)

func (c *Client) IsHost(host string) bool {
	return reg.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	lastIndex := strings.LastIndex(path, "/")

	return path[:lastIndex], path[lastIndex+1:]
}
