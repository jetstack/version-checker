package gitlab

import (
	"strings"
)

func (c *Client) IsHost(host string) bool {
	return host == "gitlab.com"
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	lastIndex := strings.LastIndex(path, "/")

	if lastIndex == -1 {
		return path, ""
	}

	return path[:lastIndex], path[lastIndex+1:]
}
