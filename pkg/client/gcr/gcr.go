package gcr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
)

const (
	lookupURL = "https://%s/v2/%s/tags/list"
)

// Ensure that we are an ImageClient
var _ api.ImageClient = (*Client)(nil)

type Options struct {
	Transporter http.RoundTripper
	Token       string
}

type Client struct {
	*http.Client
	Options
}

type Response struct {
	Manifest map[string]ManifestItem `json:"manifest"`
	Tags     []string                `json:"tags,omitempty"`
}

type ManifestItem struct {
	TimeCreated string   `json:"timeCreatedMs"`
	Tags        []string `json:"tag"`
}

func New(opts Options) *Client {
	return &Client{
		Options: opts,
		Client: &http.Client{
			Timeout:   time.Second * 5,
			Transport: opts.Transporter,
		},
	}
}

func (c *Client) Name() string {
	return "gcr"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	image = c.constructImageName(repo, image)
	url := fmt.Sprintf(lookupURL, host, image)

	req, err := c.buildRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get %q image: %w", c.Name(), err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return c.extractImageTags(response)
}

func (c *Client) constructImageName(repo, image string) string {
	if repo != "" {
		return fmt.Sprintf("%s/%s", repo, image)
	}
	return image
}

func (c *Client) buildRequest(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if len(c.Token) > 0 {
		req.SetBasicAuth("oauth2accesstoken", c.Token)
	}

	return req.WithContext(ctx), nil
}

func (c *Client) extractImageTags(response Response) ([]api.ImageTag, error) {
	tags := map[string]api.ImageTag{}
	for sha, manifestItem := range response.Manifest {
		timestamp, err := c.convertTimestamp(manifestItem.TimeCreated)
		if err != nil {
			return nil, fmt.Errorf("failed to convert timestamp string: %w", err)
		}

		// Base data shared across tags
		base := api.ImageTag{
			SHA:       sha,
			Timestamp: timestamp,
		}

		// If no tag, add without and continue early.
		if len(manifestItem.Tags) == 0 {
			tags[sha] = base
			continue
		}

		for _, tag := range manifestItem.Tags {
			current := base   // copy the base
			current.Tag = tag // set tag value

			// Already exists — add as child
			if parent, exists := tags[tag]; exists {
				parent.Children = append(parent.Children, &current)
				tags[tag] = parent
			} else {
				// First occurrence — assign as root
				tags[tag] = current
			}
		}
	}

	// Convert Map to Slice
	taglist := make([]api.ImageTag, 0, len(tags))
	for _, tag := range tags {
		taglist = append(taglist, tag)
	}

	return taglist, nil
}

func (c *Client) convertTimestamp(timeCreated string) (time.Time, error) {
	miliTimestamp, err := strconv.ParseInt(timeCreated, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, miliTimestamp*int64(1000000)), nil
}
