package ecrpublic

import (
	"regexp"
	"strings"
)

var (
	ecrPublicPattern = regexp.MustCompile(`^public\.ecr\.aws$`)
)

func (c *Client) IsHost(host string) bool {
	return ecrPublicPattern.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	parts := strings.Split(path, "/")

	return parts[0], strings.Join(parts[1:], "/")
}
