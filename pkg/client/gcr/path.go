package gcr

import (
	"regexp"
	"strings"
)

var (
	reg = regexp.MustCompile(`(^(.*\.)?gcr.io$|^(.*\.)?k8s.io$|^(.+)-docker.pkg.dev$)`)
)

func (c *Client) IsHost(host string) bool {
	return reg.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	lastIndex := strings.LastIndex(path, "/")

	if lastIndex == -1 {
		return "google-containers", path
	}

	return path[:lastIndex], path[lastIndex+1:]
}
