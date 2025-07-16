package gcr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/sirupsen/logrus"
)

const (
	lookupURL = "https://%s/v2/%s/tags/list"
)

type Options struct {
	Token       string
	Transporter http.RoundTripper
}

var _ api.ImageClient = (*Client)(nil)

type Client struct {
	*http.Client
	Options
	*logrus.Entry
}

func NewClient(opts Options, _ *authn.AuthConfig, log *logrus.Entry) *Client {
	return &Client{
		Options: opts,
		Entry:   log.WithField("client", "gcr"),
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
	var tags []api.ImageTag
	for sha, manifestItem := range response.Manifest {
		timestamp, err := c.convertTimestamp(manifestItem.TimeCreated)
		if err != nil {
			return nil, fmt.Errorf("failed to convert timestamp string: %w", err)
		}

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

func (c *Client) convertTimestamp(timeCreated string) (time.Time, error) {
	miliTimestamp, err := strconv.ParseInt(timeCreated, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, miliTimestamp*int64(1000000)), nil
}

func (c *Client) RepoImageFromPath(path string) (string, string) {
	lastIndex := strings.LastIndex(path, "/")

	// If there's no slash, then its a "root" level image
	if lastIndex == -1 {
		return "", path
	}

	return path[:lastIndex], path[lastIndex+1:]
}
