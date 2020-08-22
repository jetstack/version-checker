package docker

import (
	"regexp"
	"strings"
)

var (
	reg = regexp.MustCompile(`(^(.*\.)?docker.com$)|(^(.*\.)?docker.io$)`)
)

func (c *Client) IsHost(host string) bool {
	return reg.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string, error) {
	split := strings.Split(path, "/")

	lenSplit := len(split)
	if lenSplit == 1 {
		return "library", split[0], nil
	}

	return split[lenSplit-2], split[lenSplit-1], nil
}
