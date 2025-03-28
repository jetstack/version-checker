package selfhosted

import (
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
	Digest       string
	Architecture api.Architecture `json:"architecture"`
	History      []History        `json:"history"`
}

type History struct {
	V1Compatibility string `json:"v1Compatibility"`
}

type V1Compatibility struct {
	Created time.Time `json:"created,omitempty"`
}
