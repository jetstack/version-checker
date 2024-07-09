package acr

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest"
)

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

	token, err := cred.GetToken(ctx, azidentity.TokenRequestOptions{
		Scopes: []string{"https://" + host + "/.default"},
	})
	if err != nil {
		return nil, err
	}

	client := autorest.NewClientWithUserAgent(userAgent)
	client.Authorizer = autorest.NewBearerAuthorizer(&adal.Token{
		AccessToken: token.Token,
	})

	return &acrClient{
		Client:      &client,
		tokenExpiry: token.ExpiresOn,
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
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	token, err := credential.GetToken(ctx, azidentity.TokenRequestOptions{
		Scopes: []string{"https://" + host + "/.default"},
	})
	if err != nil {
		return nil, err
	}

	client := autorest.NewClientWithUserAgent(userAgent)
	client.Authorizer = autorest.NewBearerAuthorizer(&adal.Token{
		AccessToken: token.Token,
	})

	return &acrClient{
		Client:      &client,
		tokenExpiry: token.ExpiresOn,
	}, nil
}

func getAccessTokenClient(ctx context.Context, opts AuthOptions, host string) (*acrClient, error) {
	client := autorest.NewClientWithUserAgent(userAgent)
	urlParameters := map[string]interface{}{
		"url": "https://" + host,
	}

	formDataParameters := map[string]interface{}{
		"grant_type":    "refresh_token",
		"refresh_token": opts.RefreshToken,
		"scope":         "repository:*:*",
		"service":       host,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPost(),
		autorest.WithCustomBaseURL("{url}", urlParameters),
		autorest.WithPath("/oauth2/token"),
		autorest.WithFormData(autorest.MapToValues(formDataParameters)))
	req, err := preparer.Prepare((&http.Request{}).WithContext(ctx))
	if err != nil {
		return nil, err
	}

	resp, err := autorest.SendWithSender(client, req,
		autorest.DoRetryForStatusCodes(client.RetryAttempts, client.RetryDuration, autorest.StatusCodesForRetry...))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to request access token: %s", host, err)
	}

	var respToken ACRAccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&respToken); err != nil {
		return nil, fmt.Errorf("%s: failed to decode access token response: %s", host, err)
	}

	exp, err := getTokenExpiration(respToken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", host, err)
	}

	client.Authorizer = autorest.NewBearerAuthorizer(&adal.Token{
		AccessToken: respToken.AccessToken,
	})

	return &acrClient{
		tokenExpiry: exp,
		Client:      &client,
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
