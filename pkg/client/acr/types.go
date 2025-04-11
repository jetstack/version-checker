package acr

import (
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/jetstack/version-checker/pkg/api"
)

type acrClient struct {
	tokenExpiry time.Time
	*autorest.Client
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// API Taken from documentation @
// https://learn.microsoft.com/en-us/rest/api/containerregistry/manifests/get-list?view=rest-containerregistry-2019-08-15&tabs=HTTP

type ManifestResponse struct {
	Manifests []struct {
		CreatedTime  time.Time        `json:"createdTime"`
		Digest       string           `json:"digest"`
		Architecture api.Architecture `json:"architecture,omitempty"`
		OS           api.OS           `json:"os,omitempty"`
		Tags         []string         `json:"tags"`
	} `json:"manifests"`
}
