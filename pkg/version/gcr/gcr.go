package gcr

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joshvanl/version-checker/pkg/api"
)

const (
	repoURL                = "https://gcr.io/v2/%s/tags/list"
	repoGoogleContainerURL = "https://gcr.io/v2/google-containers/%s/tags/list"

	// Some GCR images contain subdomains (k8s, gke etc.). These should be
	// treated as being part of the google-containers project
	imageWithSubDomainRegex = `^(\w+)\.gcr\.io/(.+)$`
	imagePrefix             = "gcr.io/"
)

var (
	regImageDomain = regexp.MustCompile(imageWithSubDomainRegex)
)

type Client struct {
	*http.Client
}

type Response struct {
	Manifest map[string]ManifestItem `json:"manifest"`
}

type ManifestItem struct {
	Tag         []string `json:"tag"`
	TimeCreated string   `json:"timeCreatedMs"`
}

func New() *Client {
	return &Client{
		Client: http.DefaultClient,
	}
}

func (c *Client) IsClient(imageURL string) bool {
	return strings.HasPrefix(imageURL, imagePrefix) || regImageDomain.MatchString(imageURL)
}

func (c *Client) Tags(ctx context.Context, imageURL string) ([]api.ImageTag, error) {
	// Check if google container.
	var url string
	if match := regImageDomain.FindStringSubmatch(imageURL); len(match) == 3 {
		url = fmt.Sprintf(repoGoogleContainerURL, match[2])
	} else {
		url = fmt.Sprintf(repoURL, strings.TrimPrefix(imageURL, imagePrefix))
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get docker image: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	var tags []api.ImageTag
	for sha, manifestItem := range response.Manifest {
		miliTimestamp, err := strconv.ParseInt(manifestItem.TimeCreated, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert timestamp string: %s", err)
		}

		timestamp := time.Unix(0, miliTimestamp*int64(1000000))

		// If no tag, add without and continue early.
		if len(manifestItem.Tag) == 0 {
			tags = append(tags, api.ImageTag{SHA: sha, Timestamp: timestamp})
			continue
		}

		for _, tag := range manifestItem.Tag {
			tags = append(tags, api.ImageTag{Tag: tag, SHA: sha, Timestamp: timestamp})
		}
	}

	return tags, nil
}
