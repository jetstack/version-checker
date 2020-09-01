package docker

import (
	"regexp"
	"strings"
)

var (
	dockerReg = regexp.MustCompile(`(^(.*\.)?docker.com$)|(^(.*\.)?docker.io$)`)
)

func (c *Client) IsHost(host string) bool {
	return host == "" || dockerReg.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	split := strings.Split(path, "/")

	lenSplit := len(split)
	if lenSplit == 1 {
		return "library", split[0]
	}

	return split[lenSplit-2], split[lenSplit-1]
}
