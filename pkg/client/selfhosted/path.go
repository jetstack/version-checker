package selfhosted

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	regTemplate = (`(^(.*\.)?%s$)`)
)

func (c *Client) IsHost(host string) bool {

	u, err := url.Parse(c.Options.URL)
	if err != nil {
		//TODO
		panic(err)
	}

	urlRegex := fmt.Sprintf(regTemplate, u.Host)
	// fmt.Println(urlRegex)
	// fmt.Println(host)
	reg := regexp.MustCompile(urlRegex)
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