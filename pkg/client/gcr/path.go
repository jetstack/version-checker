package gcr

import (
	"regexp"
	"strings"
)

var (
	reg = regexp.MustCompile(`(^(.*\.)?gcr.io$)`)
)

func (c *Client) IsHost(host string) bool {
	return reg.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string, error) {
	lastIndex := strings.LastIndex(path, "/")

	if lastIndex == -1 {
		return "google-containers", path, nil
	}

	return path[:lastIndex], path[lastIndex+1:], nil
}
