package generic

/*
	Generic client for container registries
*/

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
	genericerrors "github.com/jetstack/version-checker/pkg/client/generic/errors"
	"github.com/jetstack/version-checker/pkg/client/util"
	"github.com/sirupsen/logrus"
)

const (
	// {host}/v2/{repo/image}/tags/list?n=500
	tagsPath = "%s/v2/%s/tags/list?n=500"
	// /v2/{repo/image}/manifests/{tag}
	manifestPath = "%s/v2/%s/manifests/%s"
	// Token endpoint
	tokenPath = "/v2/token"

	// HTTP headers to request API version
	dockerAPIv1Header = "application/vnd.docker.distribution.manifest.v1+json"
	dockerAPIv2Header = "application/vnd.docker.distribution.manifest.v2+json"
)

type Options struct {
	Host     string
	Username string
	Password string
	Bearer   string
}

type Client struct {
	*http.Client
	*Options

	log *logrus.Entry

	hostRegex  *regexp.Regexp
	httpScheme string
}

type AuthResponse struct {
	Token string `json:"token"`
}

type TagResponse struct {
	Tags []string `json:"tags"`
}

type ManifestResponse struct {
	Digest       string
	Architecture string    `json:"architecture"`
	History      []History `json:"history"`
}

type History struct {
	V1Compatibility string `json:"v1Compatibility"`
}

type V1Compatibility struct {
	Created time.Time `json:"created,omitempty"`
}

func New(ctx context.Context, c *http.Client, log *logrus.Entry, opts *Options) (*Client, error) {
	if c == nil {
		// default http client
		c = &http.Client{
			Timeout: time.Second * 10,
		}
	}

	client := &Client{
		Client:  c,
		Options: opts,
		log:     log.WithField("client", opts.Host),
	}

	// Set up client with host matching if set
	if opts.Host != "" {
		hostRegex, scheme, err := parseURL(opts.Host)
		if err != nil {
			return nil, fmt.Errorf("failed parsing url: %s", err)
		}
		client.hostRegex = hostRegex
		client.httpScheme = scheme

		// Setup Auth if username and password used.
		if len(opts.Username) > 0 || len(opts.Password) > 0 {
			if len(opts.Bearer) > 0 {
				return nil, errors.New("cannot specify Bearer token as well as username/password")
			}

			token, err := client.setupBasicAuth(ctx, fmt.Sprintf("%s%s", opts.Host, tokenPath))
			if httpErr, ok := genericerrors.IsHTTPError(err); ok {
				return nil, fmt.Errorf("failed to setup token auth (%d): %s",
					httpErr.StatusCode, httpErr.Body)
			}

			if err != nil {
				return nil, fmt.Errorf("failed to setup token auth: %s", err)
			}
			client.Bearer = token
		}
	}

	// Default to https if no scheme set
	if client.httpScheme == "" {
		client.httpScheme = "https"
	}

	return client, nil
}

// Name returns the name of the host URL for the selfhosted client
func (c *Client) Name() string {
	if len(c.Host) == 0 {
		return "dockerapi"
	}

	return c.Host
}

// Tags will fetch the image tags from a given image URL. It must first query
// the tags that are available, then query the 2.1 and 2.2 API endpoints to
// gather the image digest and created time.
func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	path := util.JoinRepoImage(repo, image)
	tagURL := fmt.Sprintf(tagsPath, host, path)

	var tagResponse TagResponse
	tagRequest, err := c.buildRequest(ctx, tagURL, nil)
	if err != nil {
		return nil, err
	}
	if _, err := c.doRequest(tagRequest, &tagResponse); err != nil {
		return nil, err
	}

	var tags []api.ImageTag
	for _, tag := range tagResponse.Tags {
		manifestURL := fmt.Sprintf(manifestPath, host, path, tag)

		var manifestResponse ManifestResponse
		manifestV1Request, err := c.buildRequest(ctx, manifestURL, map[string]string{"Accept": dockerAPIv1Header})
		if err != nil {
			return nil, err
		}

		_, err = c.doRequest(manifestV1Request, &manifestResponse)
		if httpErr, ok := genericerrors.IsHTTPError(err); ok {
			c.log.Errorf("%s: failed to get manifest response for tag, skipping (%d): %s",
				manifestURL, httpErr.StatusCode, httpErr.Body)
			continue
		}
		if err != nil {
			return nil, err
		}

		var timestamp time.Time
		for _, v1History := range manifestResponse.History {
			data := V1Compatibility{}
			if err := json.Unmarshal([]byte(v1History.V1Compatibility), &data); err != nil {
				return nil, err
			}

			if !data.Created.IsZero() {
				timestamp = data.Created
				// Each layer has its own created timestamp. We just want a general reference.
				// Take the first and step out the loop
				break
			}
		}

		manifestV2Request, err := c.buildRequest(ctx, manifestURL, map[string]string{"Accept": dockerAPIv2Header})
		if err != nil {
			return nil, err
		}
		header, err := c.doRequest(manifestV2Request, new(ManifestResponse))
		if httpErr, ok := genericerrors.IsHTTPError(err); ok {
			c.log.Errorf("%s: failed to get manifest sha response for tag, skipping (%d): %s",
				manifestURL, httpErr.StatusCode, httpErr.Body)
			continue
		}
		if err != nil {
			return nil, err
		}

		tags = append(tags, api.ImageTag{
			Tag:          tag,
			SHA:          header.Get("Docker-Content-Digest"),
			Timestamp:    timestamp,
			Architecture: manifestResponse.Architecture,
		})
	}

	return tags, nil
}

func (c *Client) buildRequest(ctx context.Context, url string, headers map[string]string) (*http.Request, error) {
	// assuming the caller is passing a complete and valid url
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for header, value := range headers {
		// key in the headers is case insensitive
		req.Header.Set(header, value)
	}
	if len(req.Header.Get("Authorization")) == 0 && len(c.Bearer) > 0 {
		req.Header.Add("Authorization", "Bearer "+c.Bearer)
	}
	req = req.WithContext(ctx)
	return req, nil
}

func (c *Client) doRequest(request *http.Request, obj interface{}) (http.Header, error) {
	resp, err := c.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get docker image: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, genericerrors.NewHTTPError(resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, obj); err != nil {
		return nil, fmt.Errorf("unexpected %s response: %s", request.URL, body)
	}

	return resp.Header, nil
}

func (c *Client) setupBasicAuth(ctx context.Context, url string) (string, error) {
	upReader := strings.NewReader(
		fmt.Sprintf(`{"username": "%s", "password": "%s"}`,
			c.Username, c.Password,
		),
	)

	req, err := http.NewRequest(http.MethodPost, url, upReader)
	if err != nil {
		return "", fmt.Errorf("failed to create basic auth request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send basic auth request %q: %s",
			req.URL, err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", genericerrors.NewHTTPError(resp.StatusCode, body)
	}

	response := new(AuthResponse)
	if err := json.Unmarshal(body, response); err != nil {
		return "", err
	}

	return response.Token, nil
}
