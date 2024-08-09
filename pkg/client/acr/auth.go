package acr

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
	"github.com/golang-jwt/jwt/v5"
)

const userAgent = "jetstack/version-checker"

type AuthOptions struct {
	Username     string
	Password     string
	TenantID     string
	AppID        string
	ClientSecret string
	RefreshToken string
}

func getAnonymousClient() *acrClient {
	client := autorest.NewClientWithUserAgent(userAgent)
	return &acrClient{
		Client:      &client,
		tokenExpiry: time.Unix(1<<63-1, 0),
	}
}

func getServicePrincipalClient(ctx context.Context, opts AuthOptions, host string) (*acrClient, error) {
	cred, err := azidentity.NewClientSecretCredential(opts.TenantID, opts.AppID, opts.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	tokenFunc := func(something, host string) (string, error) {
		token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{"https://" + host + "/.default"},
		})
		if err != nil {
			return "", err
		}
		return token.Token, nil
	}

	client := autorest.NewClientWithUserAgent(userAgent)
	auth := autorest.NewBearerAuthorizerCallback(client, autorest.BearerAuthorizerCallbackFunc(tokenFunc))
	client.Authorizer = auth

	return &acrClient{
		Client:      &client,
		tokenExpiry: time.Now().Add(time.Hour), // Assuming 1 hour validity for the token
	}, nil
}

func getBasicAuthClient(opts AuthOptions, host string) (*acrClient, error) {
	client := autorest.NewClientWithUserAgent(userAgent)
	client.Authorizer = autorest.NewBasicAuthorizer(opts.Username, opts.Password)

	return &acrClient{
		Client:      &client,
		tokenExpiry: time.Unix(1<<63-1, 0),
	}, nil
}

func getManagedIdentityClient(ctx context.Context, host string) (*acrClient, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	tokenFunc := func(somthing, host string) (string, error) {
		token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{"https://" + host + "/.default"},
		})
		if err != nil {
			return "", err
		}
		return token.Token, nil
	}

	client := autorest.NewClientWithUserAgent(userAgent)
	auth := autorest.NewBearerAuthorizerCallback(client, autorest.BearerAuthorizerCallbackFunc(tokenFunc))
	client.Authorizer = auth

	return &acrClient{
		Client:      &client,
		tokenExpiry: time.Now().Add(time.Hour), // Assuming 1 hour validity for the token
	}, nil
}

func getAccessTokenClient(ctx context.Context, opts AuthOptions, host string) (*acrClient, error) {
	cred, err := azidentity.NewClientSecretCredential(opts.TenantID, opts.AppID, opts.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	tokenFunc := func() (string, error) {
		token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{"https://" + host + "/.default"},
		})
		if err != nil {
			return "", fmt.Errorf("%s: failed to request access token: %s", host, err)
		}
		return token.Token, nil
	}

	client := autorest.NewClientWithUserAgent(userAgent)
	auth := autorest.NewBearerAuthorizerCallback(client, autorest.BearerAuthorizerCallbackFunc(tokenFunc))
	client.Authorizer = auth

	return &acrClient{
		Client:      &client,
		tokenExpiry: time.Now().Add(time.Hour), // Assuming 1 hour validity for the token
	}, nil
}

func getTokenExpiration(tokenString string) (time.Time, error) {
	token, err := jwt.Parse(tokenString, nil, jwt.WithoutClaimsValidation())
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

type acrClient struct {
	token       azcore.AccessToken
	tokenExpiry time.Time
	Client      *autorest.Client
}

func (c *acrClient) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if time.Now().After(c.tokenExpiry) {
		return azcore.AccessToken{}, fmt.Errorf("token expired")
	}
	return c.token, nil
}
