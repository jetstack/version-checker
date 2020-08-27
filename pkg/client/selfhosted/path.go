package selfhosted

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	// Regex template to be used to check "isHost"
	hostRegTemplate = `^.*%s$`
)

func (c *Client) IsHost(host string) bool {
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

func parseURL(rawurl string) (*regexp.Regexp, string, error) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return nil, "", fmt.Errorf("failed parsing host %q: %s", rawurl, err)
	}

	hostRegTemplate := fmt.Sprintf(hostRegTemplate, parsedURL.Host)
	hostRegex, err := regexp.Compile(hostRegTemplate)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse regex: %s for host %q: %s",
			hostRegTemplate, parsedURL.Host, err)
	}

	return hostRegex, parsedURL.Scheme, nil
}

func joinRepoImage(repo, image string) string {
	if len(repo) == 0 {
		return image
	}
	if len(image) == 0 {
		return repo
	}

	return repo + "/" + image
}
