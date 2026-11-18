package gcr

import (
	"regexp"
	"strings"
)

var (
	reg = regexp.MustCompile(`^(([a-zA-Z0-9-]+\.)?gcr\.io|([a-zA-Z0-9-]+\.)?k8s\.io|[a-zA-Z0-9-]+-docker\.pkg\.dev)$`)
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
