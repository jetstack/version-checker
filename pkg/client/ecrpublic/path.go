package ecrpublic

import (
	"regexp"
	"strings"
)

var (
	// For public ECR, we only need to match the exact hostname "public.ecr.aws"
	ecrPublicPattern = regexp.MustCompile(`^public\.ecr\.aws$`)
)

func (c *Client) IsHost(host string) bool {
	return ecrPublicPattern.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	lastIndex := strings.LastIndex(path, "/")

	if lastIndex == -1 {
		return "", path
	}

	return path[:lastIndex], path[lastIndex+1:]
}
