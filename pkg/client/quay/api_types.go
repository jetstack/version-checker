package quay

import (
	"github.com/jetstack/version-checker/pkg/api"
)

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
	Status       *int   `json:"status,omitempty"`
	ManifestData string `json:"manifest_data"`
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
