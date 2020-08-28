package ecr

import (
	"regexp"
	"strings"
)

var (
	ecrPattern = regexp.MustCompile(`(^[a-zA-Z0-9][a-zA-Z0-9-_]*)\.dkr\.ecr(\-fips)?\.([a-zA-Z0-9][a-zA-Z0-9-_]*)\.amazonaws\.com(\.cn)?$`)
)

func (c *Client) IsHost(host string) bool {
	return ecrPattern.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	lastIndex := strings.LastIndex(path, "/")

	if lastIndex == -1 {
		return "", path
	}

	return path[:lastIndex], path[lastIndex+1:]
}
