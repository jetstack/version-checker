package ecrpublic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

const (
	ecrPublicLookupURL = "https://public.ecr.aws/v2/%s/%s/tags/list"
	loginURL           = "https://public.ecr.aws/token/?service=ecr-public"
)

type Options struct {
	Username    string
	Password    string
	Transporter http.RoundTripper
}

type Client struct {
	*http.Client
	Options
}

func New(opts Options, log *logrus.Entry) (*Client, error) {
	retryclient := retryablehttp.NewClient()
	if opts.Transporter != nil {
		retryclient.HTTPClient.Transport = opts.Transporter
	}
	retryclient.HTTPClient.Timeout = 10 * time.Second
	retryclient.RetryMax = 10
	retryclient.RetryWaitMax = 10 * time.Minute
	retryclient.RetryWaitMin = 1 * time.Second
	// This custom backoff will fail requests that have a max wait of the RetryWaitMax
	retryclient.Backoff = util.HTTPBackOff
	retryclient.Logger = log.WithField("client", "ecrpublic")
	client := retryclient.StandardClient()

	return &Client{
		Options: opts,
		Client:  client,
	}, nil
}

func (c *Client) Name() string {
	return "ecrpublic"
}

func (c *Client) Tags(ctx context.Context, _, repo, image string) ([]api.ImageTag, error) {
	url := fmt.Sprintf(ecrPublicLookupURL, repo, image)

	var tags []api.ImageTag
	for url != "" {
		response, err := c.doRequest(ctx, url)
		if err != nil {
			return nil, err
		}

		for _, tag := range response.Tags {
			// No images in this result, so continue early
			if len(tag) == 0 {
				continue
			}

			tags = append(tags, api.ImageTag{
				Tag: tag,
			})
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

	// Always get a token for ECR Public
	token, err := getAnonymousToken(ctx, c.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to get anonymous token: %s", err)
	}

	req.URL.Scheme = "https"
	req = req.WithContext(ctx)
	if len(token) > 0 {
		req.Header.Add("Authorization", "Bearer "+token)
	}

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

func getAnonymousToken(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequest(http.MethodGet, loginURL, nil)
	if err != nil {
		return "", err
	}

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
