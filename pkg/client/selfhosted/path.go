package selfhosted

import (
	"fmt"
	"strings"
)

func (c *Client) IsHost(host string) bool {
	fmt.Println(c.hostRegex)
	return c.hostRegex.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	split := strings.Split(path, "/")

	lenSplit := len(split)

	if lenSplit == 1 {
		return "", split[0]
	}

	if lenSplit > 1 {
		return split[lenSplit-2], split[lenSplit-1]
	}

	return path, ""
}
