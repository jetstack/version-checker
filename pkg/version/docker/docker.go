package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/joshvanl/version-checker/pkg/api"
)

const (
	repoURL        = "https://registry.hub.docker.com/v2/repositories/%s/tags"
	imagePrefix    = "docker.io/"
	imagePrefixHub = "registry.hub.docker.com/"
)

type Client struct {
	*http.Client
}

type Response struct {
	Next    string   `json:"next"`
	Results []Result `json:"results"`
}

type Result struct {
	Name      string  `json:"name"`
	Timestamp string  `json:"last_updated"`
	Images    []Image `json:"images"`
}

type Image struct {
	Digest       string `json:"digest"`
	OS           string `json:"os"`
	Architecture string `json:"Architecture"`
}

func New() *Client {
	return &Client{
		Client: http.DefaultClient,
	}
}

func (c *Client) IsClient(imageURL string) bool {
	return strings.HasPrefix(imageURL, imagePrefix) ||
		strings.HasPrefix(imageURL, imagePrefixHub)
}

func (c *Client) Tags(ctx context.Context, imageURL string) ([]api.ImageTag, error) {
	if strings.HasPrefix(imageURL, imagePrefix) {
		imageURL = strings.TrimPrefix(imageURL, imagePrefix)
	}

	if strings.HasPrefix(imageURL, imagePrefixHub) {
		imageURL = strings.TrimPrefix(imageURL, imagePrefixHub)
	}

	if len(strings.Split(imageURL, "/")) == 1 {
		imageURL = fmt.Sprintf("library/%s", imageURL)
	}

	url := fmt.Sprintf(repoURL, imageURL)

	var tags []api.ImageTag
	for url != "" {
		response, err := c.doRequest(ctx, url)
		if err != nil {
			return nil, err
		}

		for _, result := range response.Results {
			// No images in this result, so continue early
			if len(result.Images) == 0 {
				continue
			}

			timestamp, err := time.Parse(time.RFC3339Nano, result.Timestamp)
			if err != nil {
				return nil, fmt.Errorf("failed to parse image timestamp: %s", err)
			}

			for _, image := range result.Images {
				// Image without digest contains no real image.
				if len(image.Digest) == 0 {
					continue
				}

				tags = append(tags, api.ImageTag{
					Tag:          result.Name,
					SHA:          image.Digest,
					Timestamp:    timestamp,
					OS:           image.OS,
					Architecture: image.Architecture,
				})
			}
		}

		url = response.Next
	}

	return tags, nil
}

func (c *Client) doRequest(ctx context.Context, url string) (*Response, error) {
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

	response := new(Response)
	if err := json.Unmarshal(body, response); err != nil {
		return nil, err
	}

	return response, nil
}
