package acr

import (
	"net/http"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/go-autorest/autorest"
	"github.com/jetstack/version-checker/pkg/api"
)

type Client struct {
	*http.Client
	Options

	cacheMu         sync.Mutex
	cachedACRClient map[string]*acrClient
}

type acrClient struct {
	token       azcore.AccessToken
	tokenExpiry time.Time
	Client      *autorest.Client
}

type Options struct {
	// Basic Auth
	Username string
	Password string
	// Refresh Auth
	RefreshToken string

	TenantID     string
	AppID        string
	ClientSecret string
}

type ACRAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type ACRManifestResponse struct {
	Manifests []struct {
		Digest       string           `json:"digest"`
		CreatedTime  time.Time        `json:"createdTime"`
		LastUpdated  time.Time        `json:"lastUpdateTime"`
		Tags         []string         `json:"tags"`
		Architecture api.Architecture `json:"architecture"`
		OS           api.OS           `json:"os"`

		MediaType       string `json:"mediaType"`
		ConfigMediaType string `json:"configMediaType"`
	} `json:"manifests"`
}
