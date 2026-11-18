package selfhosted

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	// Regex template to be used to check "isHost".
	// The optional prefix is restricted to hostname characters: `.*` would
	// match URL metacharacters (#, ?, @) and route credentials to another host.
	hostRegTemplate = `^([A-Za-z0-9-]+\.)*%s$`
)

func (c *Client) IsHost(host string) bool {
	if c.hostRegex == nil {
		return c.Host == host
	}
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

	if parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, "", fmt.Errorf("host %q must be a http(s) URL such as https://registry.example.com", rawurl)
	}

	hostRegTemplate := fmt.Sprintf(hostRegTemplate, regexp.QuoteMeta(parsedURL.Host))
	hostRegex, err := regexp.Compile(hostRegTemplate)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse regex: %s for host %q: %s",
			hostRegTemplate, parsedURL.Host, err)
	}

	return hostRegex, parsedURL.Scheme, nil
}
