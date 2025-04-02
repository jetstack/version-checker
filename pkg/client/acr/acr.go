package acr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v3"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/golang-jwt/jwt/v5"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

const (
	userAgent     = "jetstack/version-checker"
	requiredScope = "repository:*:metadata_read"
)

type Client struct {
	Options

	cacheMu         sync.Mutex
	cachedACRClient map[string]*acrClient
}

type acrClient struct {
	tokenExpiry time.Time
	*autorest.Client
}

type Options struct {
	Username     string
	Password     string
	RefreshToken string
	JWKSURI      string
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type ManifestResponse struct {
	Manifests []struct {
		Digest      string    `json:"digest"`
		CreatedTime time.Time `json:"createdTime"`
		Tags        []string  `json:"tags"`
	} `json:"manifests"`
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

	var tags []api.ImageTag
	for _, manifest := range manifestResp.Manifests {
		if len(manifest.Tags) == 0 {
			tags = append(tags, api.ImageTag{
				SHA:       manifest.Digest,
				Timestamp: manifest.CreatedTime,
			})

			continue
		}

		for _, tag := range manifest.Tags {
			tags = append(tags, api.ImageTag{
				SHA:       manifest.Digest,
				Timestamp: manifest.CreatedTime,
				Tag:       tag,
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

	if client, ok := c.cachedACRClient[host]; ok && time.Now().After(client.tokenExpiry) {
		return client, nil
	}

	var (
		client            *acrClient
		accessTokenClient *autorest.Client
		accessTokenReq    *http.Request
		err               error
	)
	if len(c.RefreshToken) > 0 {
		accessTokenClient, accessTokenReq, err = c.getAccessTokenRequesterForRefreshToken(ctx, host)
	} else {
		accessTokenClient, accessTokenReq, err = c.getAccessTokenRequesterForBasicAuth(ctx, host)
	}
	if err != nil {
		return nil, err
	}
	if client, err = c.getAuthorizedClient(accessTokenClient, accessTokenReq, host); err != nil {
		return nil, err
	}

	c.cachedACRClient[host] = client

	return client, nil
}

func (c *Client) getAccessTokenRequesterForBasicAuth(ctx context.Context, host string) (*autorest.Client, *http.Request, error) {
	client := autorest.NewClientWithUserAgent(userAgent)
	client.Authorizer = autorest.NewBasicAuthorizer(c.Username, c.Password)
	urlParameters := map[string]interface{}{
		"url": "https://" + host,
	}

	preparer := autorest.CreatePreparer(
		autorest.WithCustomBaseURL("{url}", urlParameters),
		autorest.WithPath("/oauth2/token"),
		autorest.WithQueryParameters(map[string]interface{}{
			"scope":   requiredScope,
			"service": host,
		}),
	)
	req, err := preparer.Prepare((&http.Request{}).WithContext(ctx))
	if err != nil {
		return nil, nil, err
	}

	return &client, req, nil
}

func (c *Client) getAccessTokenRequesterForRefreshToken(ctx context.Context, host string) (*autorest.Client, *http.Request, error) {
	client := autorest.NewClientWithUserAgent(userAgent)
	urlParameters := map[string]interface{}{
		"url": "https://" + host,
	}

	formDataParameters := map[string]interface{}{
		"grant_type":    "refresh_token",
		"refresh_token": c.RefreshToken,
		"scope":         requiredScope,
		"service":       host,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPost(),
		autorest.WithCustomBaseURL("{url}", urlParameters),
		autorest.WithPath("/oauth2/token"),
		autorest.WithFormData(autorest.MapToValues(formDataParameters)))
	req, err := preparer.Prepare((&http.Request{}).WithContext(ctx))
	if err != nil {
		return nil, nil, err
	}
	return &client, req, nil
}

func (c *Client) getAuthorizedClient(client *autorest.Client, req *http.Request, host string) (*acrClient, error) {
	resp, err := autorest.SendWithSender(client, req,
		autorest.DoRetryForStatusCodes(client.RetryAttempts, client.RetryDuration, autorest.StatusCodesForRetry...),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to request access token: %s",
			host, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var respToken AccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&respToken); err != nil {
		return nil, fmt.Errorf("%s: failed to decode access token response: %s",
			host, err)
	}

	exp, err := c.getTokenExpiration(respToken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", host, err)
	}

	token := &adal.Token{
		RefreshToken: "", // empty if access_token was retrieved with basic auth. but client is not reused after expiry anyway (see cachedACRClient)
		AccessToken:  respToken.AccessToken,
	}

	client.Authorizer = autorest.NewBearerAuthorizer(token)

	return &acrClient{
		tokenExpiry: exp,
		Client:      client,
	}, nil
}

func (c *Client) getTokenExpiration(tokenString string) (time.Time, error) {
	jwtParser := jwt.NewParser(jwt.WithoutClaimsValidation())
	var token *jwt.Token
	var err error
	if c.JWKSURI != "" {
		var k keyfunc.Keyfunc
		k, err = keyfunc.NewDefaultCtx(context.TODO(), []string{c.JWKSURI})
		if err != nil {
			return time.Time{}, err
		}
		token, err = jwtParser.Parse(tokenString, k.Keyfunc)
	} else {
		token, _, err = jwtParser.ParseUnverified(tokenString, jwt.MapClaims{})
	}
	if err != nil {
		return time.Time{}, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return time.Time{}, fmt.Errorf("failed to process claims in access token")
	}

	if exp, ok := claims["exp"].(float64); ok {
		timestamp := time.Unix(int64(exp), 0)
		return timestamp, nil
	}

	return time.Time{}, fmt.Errorf("failed to find 'exp' claim in access token")
}
