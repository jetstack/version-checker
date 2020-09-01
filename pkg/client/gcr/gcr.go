package gcr

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
)

const (
	lookupURL = "https://%s/v2/%s/%s/tags/list"
)

type Options struct {
	Token string
}

type Client struct {
	*http.Client
	Options
}

type Response struct {
	Manifest map[string]ManifestItem `json:"manifest"`
}

type ManifestItem struct {
	Tag         []string `json:"tag"`
	TimeCreated string   `json:"timeCreatedMs"`
}

func New(opts Options) *Client {
	return &Client{
		Options: opts,
		Client: &http.Client{
			Timeout: time.Second * 5,
		},
	}
}

func (c *Client) Name() string {
	return "gcr"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	if repo == "google-containers" {
		host = "gcr.io"
	}

	url := fmt.Sprintf(lookupURL, host, repo, image)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if len(c.Token) > 0 {
		req.SetBasicAuth("oauth2accesstoken", c.Token)
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
