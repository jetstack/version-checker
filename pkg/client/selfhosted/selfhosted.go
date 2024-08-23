package selfhosted

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
	selfhostederrors "github.com/jetstack/version-checker/pkg/client/selfhosted/errors"
	"github.com/jetstack/version-checker/pkg/client/util"
)

const (
	// {host}/v2/{repo/image}/tags/list?n=500.
	tagsPath = "%s/v2/%s/tags/list?n=500"
	// /v2/{repo/image}/manifests/{tag}.
	manifestPath = "%s/v2/%s/manifests/%s"
	// Token endpoint.
	defaultTokenPath = "/v2/token"

	// HTTP headers to request API version.
	dockerAPIv1Header = "application/vnd.docker.distribution.manifest.v1+json"
	dockerAPIv2Header = "application/vnd.docker.distribution.manifest.v2+json"
)

type Options struct {
	Host      string
	Username  string
	Password  string
	Bearer    string
	TokenPath string
	Insecure  bool
	CAPath    string
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
	Architecture api.Architecture `json:"architecture"`
	History      []History        `json:"history"`
}

type History struct {
	V1Compatibility string `json:"v1Compatibility"`
}

type V1Compatibility struct {
	Created time.Time `json:"created,omitempty"`
}

func New(ctx context.Context, log *logrus.Entry, opts *Options) (*Client, error) {
	client := &Client{
		Client: &http.Client{
			Timeout: time.Second * 10,
		},
		Options: opts,
		log:     log.WithField("client", opts.Host),
	}

	if err := client.setupHostAndAuth(ctx); err != nil {
		return nil, err
	}

	client.setDefaultScheme()

	if err := client.setupTLSConfig(); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) setupHostAndAuth(ctx context.Context) error {
	if c.Options.Host == "" {
		return nil
	}

	hostRegex, scheme, err := parseURL(c.Options.Host)
	if err != nil {
		return fmt.Errorf("failed parsing url: %w", err)
	}
	c.hostRegex = hostRegex
	c.httpScheme = scheme

	if len(c.Options.Username) > 0 || len(c.Options.Password) > 0 {
		if len(c.Options.Bearer) > 0 {
			return errors.New("cannot specify Bearer token as well as username/password")
		}

		tokenPath := c.Options.TokenPath
		if tokenPath == "" {
			tokenPath = defaultTokenPath
		}

		token, err := c.setupBasicAuth(ctx, c.Options.Host, tokenPath)
		if httpErr, ok := selfhostederrors.IsHTTPError(err); ok {
			return fmt.Errorf("failed to setup token auth (%d): %s", httpErr.StatusCode, httpErr.Body)
		}
		if err != nil {
			return fmt.Errorf("failed to setup token auth: %w", err)
		}
		c.Bearer = token
	}

	return nil
}

func (c *Client) setDefaultScheme() {
	if c.httpScheme == "" {
		c.httpScheme = "https"
	}
}

func (c *Client) setupTLSConfig() error {
	if c.httpScheme != "https" {
		return nil
	}

	tlsConfig, err := newTLSConfig(c.Options.Insecure, c.Options.CAPath)
	if err != nil {
		return err
	}

	c.Client.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
	}

	return nil
}

// Name returns the name of the host URL for the selfhosted client.
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
	if _, err := c.doRequest(ctx, tagURL, "", &tagResponse); err != nil {
		return nil, err
	}

	var tags []api.ImageTag
	for _, tag := range tagResponse.Tags {
		manifestURL := fmt.Sprintf(manifestPath, host, path, tag)

		var manifestResponse ManifestResponse
		_, err := c.doRequest(ctx, manifestURL, dockerAPIv1Header, &manifestResponse)

		if httpErr, ok := selfhostederrors.IsHTTPError(err); ok {
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

		header, err := c.doRequest(ctx, manifestURL, dockerAPIv2Header, new(ManifestResponse))
		if httpErr, ok := selfhostederrors.IsHTTPError(err); ok {
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

func (c *Client) doRequest(ctx context.Context, url, header string, obj interface{}) (http.Header, error) {
	url = fmt.Sprintf("%s://%s", c.httpScheme, url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	if len(c.Bearer) > 0 {
		req.Header.Add("Authorization", "Bearer "+c.Bearer)
	}
	if len(header) > 0 {
		req.Header.Set("Accept", header)
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get docker image: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, selfhostederrors.NewHTTPError(resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, obj); err != nil {
		return nil, fmt.Errorf("unexpected %s response: %s", url, body)
	}

	return resp.Header, nil
}

func (c *Client) setupBasicAuth(ctx context.Context, url, tokenPath string) (string, error) {
	upReader := strings.NewReader(
		fmt.Sprintf(`{"username": "%s", "password": "%s"}`,
			c.Username, c.Password,
		),
	)

	tokenURL := url + tokenPath

	req, err := http.NewRequest(http.MethodPost, tokenURL, upReader)
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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", selfhostederrors.NewHTTPError(resp.StatusCode, body)
	}

	response := new(AuthResponse)
	if err := json.Unmarshal(body, response); err != nil {
		return "", err
	}

	return response.Token, nil
}

func newTLSConfig(insecure bool, CAPath string) (*tls.Config, error) {
	// Load system CA Certs and/or create a new CertPool
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	if CAPath != "" {
		certs, err := os.ReadFile(CAPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to append %q to RootCAs: %v", CAPath, err)
		}
		rootCAs.AppendCertsFromPEM(certs)
	}

	return &tls.Config{
		InsecureSkipVerify: insecure,
		RootCAs:            rootCAs,
	}, nil
}
