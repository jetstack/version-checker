package selfhosted

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

var (
	regTemplate = (`(^(.*\.)?%s$)`)
)

func (c *Client) IsHost(host string) bool {

	u, err := url.Parse(c.Options.URL)
	if err != nil {
		// If we can't parse the host given by the options, it's useless to keep running
		log.Fatalf("failed parsing host: %s", c.Options.URL)
		return false
	}

	urlRegex := fmt.Sprintf(regTemplate, u.Host)
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
