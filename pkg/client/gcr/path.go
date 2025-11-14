package gcr

import (
	"regexp"
	"strings"
)

var (
	reg = regexp.MustCompile(`(^(.*\.)?gcr\.io$|^(.*\.)?k8s\.io$|^(.+)-docker\.pkg\.dev$)`)
)

func (c *Client) IsHost(host string) bool {
	return reg.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	split := strings.Split(path, "/")

	lenSplit := len(split)
	if lenSplit == 1 {
		return "google-containers", split[0]
	}

	if lenSplit > 1 {
		return strings.Join(split[:len(split)-1], "/"), split[lenSplit-1]
	}

	return path, ""
}
