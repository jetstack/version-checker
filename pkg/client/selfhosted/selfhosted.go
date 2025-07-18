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

	"github.com/go-chi/transport"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/jetstack/version-checker/pkg/api"
	selfhostederrors "github.com/jetstack/version-checker/pkg/client/selfhosted/errors"
	"github.com/jetstack/version-checker/pkg/client/util"
)

// Ensure that we are an ImageClient
var _ api.ImageClient = (*Client)(nil)

const (
	// {host}/v2/{repo/image}/tags/list?n=500
	tagsPath = "%s/v2/%s/tags/list?n=500"
	// /v2/{repo/image}/manifests/{tag}
	manifestPath = "%s/v2/%s/manifests/%s"
	// Token endpoint
	defaultTokenPath = "/v2/token"

	// HTTP headers to request API version
	dockerAPIv1Header       = "application/vnd.docker.distribution.manifest.v1+json"
	dockerAPIv2Header       = "application/vnd.docker.distribution.manifest.v2+json"
	dockerAPIv2ManifestList = "application/vnd.docker.distribution.manifest.list.v2+json"
)

type Options struct {
	Transporter http.RoundTripper

	Host      string
	Username  string
	Password  string
	Bearer    string
	TokenPath string
	CAPath    string
	Insecure  bool
}
type Client struct {
	*http.Client
	*Options

	log *logrus.Entry

	hostRegex  *regexp.Regexp
	httpScheme string
}

func New(ctx context.Context, log *logrus.Entry, opts *Options) (*Client, error) {
	client := &Client{
		Client: &http.Client{
			Timeout:   time.Second * 10,
			Transport: cleanhttp.DefaultTransport(),
		},
		Options: opts,
		log:     log.WithField("client", "selfhosted-"+opts.Host),
	}

	if err := configureHost(ctx, client, opts); err != nil {
		return nil, err
	}

	if err := configureTransport(client, opts); err != nil {
		return nil, err
	}

	return client, nil
}

func configureHost(ctx context.Context, client *Client, opts *Options) error {
	if opts.Host == "" {
		return nil
	}

	hostRegex, scheme, err := parseURL(opts.Host)
	if err != nil {
		return fmt.Errorf("failed parsing url: %s", err)
	}
	client.hostRegex = hostRegex
	client.httpScheme = scheme

	if err := configureAuth(ctx, client, opts); err != nil {
		return err
	}

	return nil
}

func configureAuth(ctx context.Context, client *Client, opts *Options) error {
	if len(opts.Username) == 0 && len(opts.Password) == 0 {
		return nil
	}

	if len(opts.Bearer) > 0 {
		return errors.New("cannot specify Bearer token as well as username/password")
	}

	tokenPath := opts.TokenPath
	if tokenPath == "" {
		tokenPath = defaultTokenPath
	}

	token, err := client.setupBasicAuth(ctx, opts.Host, tokenPath)
	if httpErr, ok := selfhostederrors.IsHTTPError(err); ok {
		if httpErr.StatusCode == http.StatusNotFound {
			client.log.Warnf("Token endpoint not found, using basic auth: %s%s %s", opts.Host, tokenPath, httpErr.Body)
		} else {
			return fmt.Errorf("failed to setup token auth (%d): %s",
				httpErr.StatusCode, httpErr.Body)
		}
	} else if err != nil {
		return fmt.Errorf("failed to setup token auth: %s", err)
	}
	client.Bearer = token
	return nil
}

func configureTransport(client *Client, opts *Options) error {
	if client.httpScheme == "" {
		client.httpScheme = "https"
	}
	baseTransport := cleanhttp.DefaultTransport()
	baseTransport.Proxy = http.ProxyFromEnvironment

	if client.httpScheme == "https" {
		tlsConfig, err := newTLSConfig(opts.Insecure, opts.CAPath)
		if err != nil {
			return err
		}
		baseTransport.TLSClientConfig = tlsConfig
	}

	client.Transport = transport.Chain(baseTransport,
		transport.If(logrus.IsLevelEnabled(logrus.DebugLevel), transport.LogRequests(transport.LogOptions{Concise: true})),
		transport.If(opts.Transporter != nil, func(rt http.RoundTripper) http.RoundTripper { return opts.Transporter }))
	return nil
}

// Name returns the name of the host URL for the selfhosted client
func (c *Client) Name() string {
	if len(c.Host) == 0 {
		return "selfhosted"
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

	tags := map[string]api.ImageTag{}
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
			if !v1History.V1Compatibility.Created.IsZero() {
				timestamp = v1History.V1Compatibility.Created
				// Each layer has its own created timestamp. We just want a general reference.
				// Take the first and step out the loop
				break
			}
		}

		var manifestListResponse V2ManifestListResponse
		header, err := c.doRequest(ctx, manifestURL, strings.Join([]string{dockerAPIv2Header, dockerAPIv2ManifestList}, ","), &manifestListResponse)
		if httpErr, ok := selfhostederrors.IsHTTPError(err); ok {
			c.log.Errorf("%s: failed to get manifest sha response for tag, skipping (%d): %s",
				manifestURL, httpErr.StatusCode, httpErr.Body)
			continue
		}
		if err != nil {
			return nil, err
		}

		// Lets set as much of the current as we know
		current := api.ImageTag{
			Tag:          tag,
			SHA:          header.Get("Docker-Content-Digest"),
			Timestamp:    timestamp,
			Architecture: api.Architecture(manifestResponse.Architecture),
		}

		util.BuildTags(tags, tag, &current)

		for _, manifest := range manifestListResponse.Manifests {

			// If we didn't get a SHA from the inital call,
			// lets set it from the manifestList
			if current.SHA == "" && manifest.Digest != "" {
				current.SHA = manifest.Digest
			}

			util.BuildTags(tags, tag, &current)
		}
	}
	return util.TagMaptoList(tags), nil
}

func (c *Client) doRequest(ctx context.Context, url, header string, obj interface{}) (http.Header, error) {
	url = fmt.Sprintf("%s://%s", c.httpScheme, url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "version-checker/selfhosted")

	req = req.WithContext(ctx)
	if len(c.Bearer) > 0 {
		req.Header.Add("Authorization", "Bearer "+c.Bearer)
	} else if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	if len(header) > 0 {
		req.Header.Set("Accept", header)
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, selfhostederrors.NewHTTPError(resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, obj); err != nil {
		return nil, fmt.Errorf("unexpected %s response: %s - %w", url, body, err)
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
	defer func() { _ = resp.Body.Close() }()

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
			return nil, fmt.Errorf("failed to append %q to RootCAs: %v", CAPath, err)
		}
		rootCAs.AppendCertsFromPEM(certs)
	}

	return &tls.Config{
		Renegotiation:      tls.RenegotiateOnceAsClient,
		InsecureSkipVerify: insecure, // #nosec G402
		RootCAs:            rootCAs,
	}, nil
}
