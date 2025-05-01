package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/sirupsen/logrus"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

// Ensure that we are an ImageClient
var _ api.ImageClient = (*Client)(nil)

// Values taken from: https://docs.docker.com/docker-hub/usage/#abuse-rate-limit
const (
	windowDuration = time.Minute
	APIRateLimit   = 500
	maxWait        = time.Hour
)

const (
	loginURL  = "https://hub.docker.com/v2/users/login/"
	lookupURL = "https://registry.hub.docker.com/v2/repositories/%s/%s/tags?page_size=100"
)

type Options struct {
	Transporter http.RoundTripper
	Username    string
	Password    string
	Token       string
}

type Client struct {
	*http.Client
	Options

	log     *logrus.Entry
	limiter *rate.Limiter
}

func New(opts Options, log *logrus.Entry) (*Client, error) {
	ctx := context.Background()

	limiter := rate.NewLimiter(
		rate.Every(windowDuration/APIRateLimit),
		1,
	)
	log = log.WithField("client", "docker")

	retryclient := retryablehttp.NewClient()
	if opts.Transporter != nil {
		retryclient.HTTPClient.Transport = opts.Transporter
	}
	retryclient.Backoff = util.RateLimitedBackoffLimiter(log, limiter, maxWait)
	retryclient.HTTPClient.Timeout = 10 * time.Second
	retryclient.RetryMax = 10
	retryclient.RetryWaitMax = 10 * time.Minute
	retryclient.RetryWaitMin = 1 * time.Second
	// This custom backoff will fail requests that have a max wait of the RetryWaitMax
	retryclient.Backoff = util.HTTPBackOff
	retryclient.Logger = log.WithField("client", "docker")
	client := retryclient.StandardClient()

	// Setup Auth if username and password used.
	if len(opts.Username) > 0 || len(opts.Password) > 0 {
		if len(opts.Token) > 0 {
			return nil, errors.New("cannot specify Token as well as username/password")
		}

		token, err := basicAuthSetup(ctx, client, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to setup auth: %s", err)
		}
		opts.Token = token
	}

	return &Client{
		Options: opts,
		Client:  client,
		log:     log,
		limiter: limiter,
	}, nil
}

func (c *Client) Name() string {
	return "dockerhub"
}

func (c *Client) Tags(ctx context.Context, _, repo, image string) ([]api.ImageTag, error) {
	url := fmt.Sprintf(lookupURL, repo, image)

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

			var timestamp time.Time
			if len(result.Timestamp) > 0 {
				timestamp, err = time.Parse(time.RFC3339Nano, result.Timestamp)
				if err != nil {
					return nil, fmt.Errorf("failed to parse image timestamp: %s", err)
				}
			}

			tag := api.ImageTag{
				Tag:       result.Name,
				Timestamp: timestamp,
			}

			// If we have a Digest, lets set it..
			if result.Digest != "" {
				tag.SHA = result.Digest
			}

			for _, image := range result.Images {
				// Image without digest contains no real image.
				if len(image.Digest) == 0 {
					continue
				}

				tag.Children = append(tag.Children, &api.ImageTag{
					Tag:          result.Name,
					SHA:          image.Digest,
					Timestamp:    timestamp,
					OS:           image.OS,
					Architecture: image.Architecture,
				})
			}

			// If we only have one child, and it has a SHA, then lets use that in the parent
			if tag.SHA == "" && len(tag.Children) == 1 && tag.Children[0].SHA != "" {
				tag.SHA = tag.Children[0].SHA
			}

			// Append our Tag at the end...
			tags = append(tags, tag)
		}

		url = response.Next
	}

	return tags, nil
}

func (c *Client) doRequest(ctx context.Context, url string) (*TagResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.URL.Scheme = "https"
	req = req.WithContext(ctx)
	if len(c.Token) > 0 {
		req.Header.Add("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("User-Agent", "version-checker/docker")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get %q image: %s", c.Name(), err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := new(TagResponse)
	if err := json.Unmarshal(body, response); err != nil {
		return nil, fmt.Errorf("unexpected image tags response: %s", body)
	}

	return response, nil
}

func basicAuthSetup(ctx context.Context, client *http.Client, opts Options) (string, error) {
	upReader := strings.NewReader(
		fmt.Sprintf(`{"username": "%s", "password": "%s"}`,
			opts.Username, opts.Password,
		),
	)

	req, err := http.NewRequest(http.MethodPost, loginURL, upReader)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "version-checker/docker")
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(string(body))
	}

	response := new(AuthResponse)
	if err := json.Unmarshal(body, response); err != nil {
		return "", err
	}

	return response.Token, nil
}
