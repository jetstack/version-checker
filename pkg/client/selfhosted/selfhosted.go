package selfhosted

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
)

const (
	// This n=500 is a temporary work around until pagination is properly tested
	// Not all versions support pagination AND not all Artifactory versions are handling the "latest" argument
	lookupURL   = "%s/%s/%s/tags/list?n=500"
	manifestURL = "%s/%s/%s/manifests/"
	regTemplate = `(^(.*\.)?%s$)`
)

type Options struct {
	URL      string
	LoginURL string
	Username string
	Password string
	Bearer   string
}

type Client struct {
	*http.Client
	Options
	hostRegex *regexp.Regexp
}

type AuthResponse struct {
	Token string `json:"token"`
}

type TagResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
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

func New(ctx context.Context, opts Options) (*Client, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	parsedURL, err := url.Parse(opts.URL)
	if err != nil {
		// If we can't parse the host given by the options, we should exit fatal
		return nil, fmt.Errorf("failed parsing host: %s", opts.URL)
	}

	hostRegex, err := regexp.Compile(fmt.Sprintf(regTemplate, parsedURL.Host))
	if err != nil {
		return nil, fmt.Errorf("failed parsing regex: %s for host: %s", regTemplate, parsedURL.Host)
	}

	// Only try to setup auth if an actually URL is present.
	if opts.URL != "" {
		// Setup Auth if username and password used.
		if len(opts.Username) > 0 || len(opts.Password) > 0 {
			if len(opts.Bearer) > 0 {
				return nil, errors.New("cannot specify Bearer as well as username/password")
			}

			token, err := basicAuthSetup(client, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to setup auth: %s", err)
			}
			opts.Bearer = token
		}
	}

	fmt.Println(hostRegex)

	return &Client{
		Options:   opts,
		Client:    client,
		hostRegex: hostRegex,
	}, nil
}

func (c *Client) Tags(ctx context.Context, _, repo, image string) ([]api.ImageTag, error) {
	url := fmt.Sprintf(lookupURL, c.Options.URL, repo, image)
	urlManifest := fmt.Sprintf(manifestURL, c.Options.URL, repo, image)
	var tags []api.ImageTag
	var time time.Time

	response, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	for _, tag := range response.Tags {

		manifestResponse, err := c.doManifestRequest(ctx, urlManifest+tag)
		if err != nil {
			return nil, err
		}

		for _, v1History := range manifestResponse.History {
			data := V1Compatibility{}
			if err := json.Unmarshal([]byte(v1History.V1Compatibility), &data); err != nil {
				return nil, err
			}

			if !data.Created.IsZero() {
				time = data.Created
				// Each layer has its own created timestamp. We just want a general reference.
				// Take the first and step out the loop
				break
			}
		}

		tags = append(tags, api.ImageTag{
			Tag:          tag,
			SHA:          manifestResponse.Digest,
			Timestamp:    time,
			Architecture: manifestResponse.Architecture,
		})
	}

	return tags, nil
}

func (c *Client) doManifestRequest(ctx context.Context, url string) (*ManifestResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	if len(c.Bearer) > 0 {
		req.Header.Add("Authorization", "Bearer "+c.Bearer)
	}

	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v1+json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get docker image: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := new(ManifestResponse)
	if err := json.Unmarshal(body, response); err != nil {
		return nil, fmt.Errorf("unexpected image tags response: %s", body)
	}

	response.Digest = resp.Header.Get("Docker-Content-Digest")

	if response.Digest == "" {
		return nil, fmt.Errorf("Missing Docker-Content-Digest in response header: %s", resp.Header)
	}

	return response, nil
}

func (c *Client) doRequest(ctx context.Context, url string) (*TagResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	if len(c.Bearer) > 0 {
		req.Header.Add("Authorization", "Bearer "+c.Bearer)
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

func basicAuthSetup(client *http.Client, opts Options) (string, error) {
	upReader := strings.NewReader(
		fmt.Sprintf(`{"username": "%s", "password": "%s"}`,
			opts.Username, opts.Password,
		),
	)

	req, err := http.NewRequest(http.MethodPost, opts.LoginURL, upReader)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

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
