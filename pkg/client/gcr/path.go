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

	// If there's no slash, then its a "root" level image
	if lastIndex == -1 {
		return "", path
	}

	return path[:lastIndex], path[lastIndex+1:]
}
