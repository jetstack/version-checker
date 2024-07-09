package quay

import (
	"github.com/hashicorp/go-retryablehttp"
	"github.com/jetstack/version-checker/pkg/api"
)

type Options struct {
	Token string
}

type Client struct {
	*retryablehttp.Client
	Options
}

type responseTag struct {
	Tags          []responseTagItem `json:"tags"`
	HasAdditional bool              `json:"has_additional"`
	Page          int               `json:"page"`
}

type responseTagItem struct {
	Name           string `json:"name"`
	ManifestDigest string `json:"manifest_digest"`
	LastModified   string `json:"last_modified"`
	IsManifestList bool   `json:"is_manifest_list"`
}

type responseManifest struct {
	ManifestData string `json:"manifest_data"`
	Status       *int   `json:"status,omitempty"`
}

type responseManifestData struct {
	Manifests []responseManifestDataItem `json:"manifests"`
}

type responseManifestDataItem struct {
	Digest   string `json:"digest"`
	Platform struct {
		Architecture api.Architecture `json:"architecture"`
		OS           api.OS           `json:"os"`
	} `json:"platform"`
}
