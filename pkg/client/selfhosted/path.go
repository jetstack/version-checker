package selfhosted

import (
	"errors"
	"regexp"
	"strings"
)

func (c *Client) IsHost(host string) bool {
	reg := regexp.MustCompile(c.Options.HostRegex)
	return reg.MatchString(host)
}

func (c *Client) RepoImageFromPath(path string) (string, string, error) {
	split := strings.Split(path, "/")

	lenSplit := len(split)

	if lenSplit >= 2 {
		return split[lenSplit-2], split[lenSplit-1], nil
	}

	return "", "", errors.New("bad split path length")

}
