package acr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/Azure/go-autorest/autorest"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

// Ensure that we are an ImageClient
var _ api.ImageClient = (*Client)(nil)

const (
	userAgent     = "jetstack/version-checker"
	requiredScope = "repository:*:metadata_read"
)

type Client struct {
	cachedACRClient map[string]*acrClient
	Options

	cacheMu sync.Mutex
}

type Options struct {
	Username     string
	Password     string
	RefreshToken string
	JWKSURI      string
}

func New(opts Options) (*Client, error) {
	if len(opts.RefreshToken) > 0 &&
		(len(opts.Username) > 0 || len(opts.Password) > 0) {
		return nil, errors.New("cannot specify refresh token as well as username/password")
	}

	return &Client{
		Options:         opts,
		cachedACRClient: make(map[string]*acrClient),
	}, nil
}

func (c *Client) Name() string {
	return "acr"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	client, err := c.getACRClient(ctx, host)
	if err != nil {
		return nil, err
	}

	resp, err := c.getManifestsWithClient(ctx, client, host, repo, image)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var manifestResp ManifestResponse
	if err := json.NewDecoder(resp.Body).Decode(&manifestResp); err != nil {
		return nil, fmt.Errorf("%s: failed to decode manifest response: %s",
			host, err)
	}

	// Create a map of tags, so that when we come up with additional Tags
	// we can add them as Children
	tags := map[string]api.ImageTag{}

	for _, manifest := range manifestResp.Manifests {
		// Base data shared across tags
		base := api.ImageTag{
			SHA:          manifest.Digest,
			Timestamp:    manifest.CreatedTime,
			OS:           manifest.OS,
			Architecture: manifest.Architecture,
		}

		// No tags, use digest as the key
		if len(manifest.Tags) == 0 {
			tags[base.SHA] = base
			continue
		}

		for _, tag := range manifest.Tags {
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
	return util.TagMaptoList(tags), nil
}

func (c *Client) getManifestsWithClient(ctx context.Context, client *acrClient, host, repo, image string) (*http.Response, error) {
	urlParameters := map[string]interface{}{
		"url": "https://" + host,
	}

	pathParameters := map[string]interface{}{
		"name": autorest.Encode("path", util.JoinRepoImage(repo, image)),
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithCustomBaseURL("{url}", urlParameters),
		autorest.WithPathParameters("/acr/v1/{name}/_manifests", pathParameters))

	req, err := preparer.Prepare(new(http.Request).WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare client: %s", err)
	}

	resp, err := autorest.SendWithSender(client, req,
		autorest.DoRetryForStatusCodes(client.RetryAttempts, client.RetryDuration, autorest.StatusCodesForRetry...))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("bad request for image host %s", host)
		}
		return nil, fmt.Errorf("bad request for image host %s: %s", host, body)
	}

	return resp, nil
}
