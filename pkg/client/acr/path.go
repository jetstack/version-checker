package acr

import (
	"regexp"
	"strings"
)

var (
	reg = regexp.MustCompile(`^[a-zA-Z0-9-]+\.azurecr\.(io|cn|de|us)$`)
)

func (c *Client) IsHost(host string) bool {
	return reg.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	lastIndex := strings.LastIndex(path, "/")

	if lastIndex == -1 {
		return "", path
	}

	return path[:lastIndex], path[lastIndex+1:]
}
