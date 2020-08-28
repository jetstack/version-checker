package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
)

const (
	loginURL  = "https://hub.docker.com/v2/users/login/"
	lookupURL = "https://registry.hub.docker.com/v2/repositories/%s/%s/tags"
)

type Options struct {
	Username string
	Password string
	Token    string
}

type Client struct {
	*http.Client
	Options
}

type AuthResponse struct {
	Token string `json:"token"`
}

type TagResponse struct {
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

func New(ctx context.Context, opts Options) (*Client, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

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
	}, nil
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

func (c *Client) doRequest(ctx context.Context, url string) (*TagResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.URL.Scheme = "https"
	req = req.WithContext(ctx)
	if len(c.Token) > 0 {
		req.Header.Add("Authorization", "Token "+c.Token)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get docker image: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
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

	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
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
