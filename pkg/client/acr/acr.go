package acr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

func New(opts Options) (*Client, error) {
	client := &http.Client{
		Timeout: time.Second * 5,
	}

	if len(opts.RefreshToken) > 0 &&
		(len(opts.Username) > 0 || len(opts.Password) > 0) {
		return nil, errors.New("cannot specify refresh token as well as username/password")
	}

	return &Client{
		Options:         opts,
		Client:          client,
		cachedACRClient: make(map[string]*acrClient),
	}, nil
}

func (c *Client) Name() string {
	return "acr"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	var tags []api.ImageTag

	client, err := c.getACRClient(ctx, host)
	if err != nil {
		return nil, err
	}

	resp, err := c.getManifestsWithClient(ctx, client, host, repo, image)
	if err != nil {
		return nil, err
	}

	var manifestResp ACRManifestResponse
	if err := json.NewDecoder(resp.Body).Decode(&manifestResp); err != nil {
		return nil, fmt.Errorf("%s: failed to decode manifest response: %s", host, err)
	}

	for _, manifest := range manifestResp.Manifests {
		if len(manifest.Tags) == 0 {
			tags = append(tags, api.ImageTag{
				SHA:          manifest.Digest,
				Timestamp:    manifest.CreatedTime,
				OS:           manifest.OS,
				Architecture: manifest.Architecture,
			})
			continue
		}

		for _, tag := range manifest.Tags {
			tags = append(tags, api.ImageTag{
				SHA:          manifest.Digest,
				Timestamp:    manifest.CreatedTime,
				Tag:          tag,
				OS:           manifest.OS,
				Architecture: manifest.Architecture,
			})
		}
	}

	return tags, nil
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

func (c *Client) getACRClient(ctx context.Context, host string) (*acrClient, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	if client, ok := c.cachedACRClient[host]; ok && time.Now().Before(client.tokenExpiry) {
		return client, nil
	}

	var (
		client *acrClient
		err    error
	)

	authOpts := AuthOptions{
		Username:     c.Options.Username,
		Password:     c.Options.Password,
		TenantID:     c.Options.TenantID,
		AppID:        c.Options.AppID,
		ClientSecret: c.Options.ClientSecret,
		RefreshToken: c.Options.RefreshToken,
	}
	if len(authOpts.RefreshToken) > 0 {
		client, err = getAccessTokenClient(ctx, authOpts, host)
	} else if authOpts.Username != "" && authOpts.Password != "" {
		client, err = getBasicAuthClient(authOpts, host)
	} else if authOpts.TenantID != "" && authOpts.AppID != "" && authOpts.ClientSecret != "" {
		client, err = getServicePrincipalClient(ctx, authOpts, host)
	} else {
		client, err = getManagedIdentityClient(ctx, host)
	}
	if err != nil {
		return nil, err
	}

	c.cachedACRClient[host] = client

	return client, nil
}
