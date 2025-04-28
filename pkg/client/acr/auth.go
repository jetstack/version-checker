package acr

// The intention here is to provide a client for Azure Container Registry (ACR)
// that can authenticate using either basic authentication (username/password)
// or a refresh token. The client will cache the access token and its expiration
// time to avoid unnecessary requests to the ACR server.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

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
