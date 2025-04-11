package selfhosted

import (
	"encoding/json"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
)

type AuthResponse struct {
	Token string `json:"token"`
}

type TagResponse struct {
	Tags []string `json:"tags"`
}

type ManifestResponse struct {
	Digest       string           `json:"digest,omitempty"`
	Architecture api.Architecture `json:"architecture"`
	History      []History        `json:"history"`
}

type ManafestListResponse struct {
	Manifests []ManifestResponse `json:"manifests"`
}

type History struct {
	V1Compatibility V1CompatibilityWrapper `json:"v1Compatibility"`
}

type V1Compatibility struct {
	Created time.Time `json:"created,omitempty"`
}

type V1CompatibilityWrapper struct {
	V1Compatibility
}

func (v *V1CompatibilityWrapper) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), &v.V1Compatibility)
}

type ErrorResponse struct {
	Errors []ErrorType `json:"errors"`
}

type ErrorType struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type V2ManifestListResponse struct {
	SchemaVersion int                   `json:"schemaVersion"`
	MediaType     string                `json:"mediaType"`
	Manifests     []V2ManifestListEntry `json:"manifests"`
}

type V2ManifestListEntry struct {
	Digest    string       `json:"digest"`
	MediaType string       `json:"mediaType"`
	Platform  api.Platform `json:"platform"`
}
