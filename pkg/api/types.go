package api

import (
	"context"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/sirupsen/logrus"
)

// ImageTag describes a container image tag.
type ImageTag struct {
	Tag          string       `json:"tag"`
	SHA          string       `json:"sha"`
	Timestamp    time.Time    `json:"timestamp"`
	OS           OS           `json:"os,omitempty"`
	Architecture Architecture `json:"architecture,omitempty"`
}

type OS string
type Architecture string

// ImageClient represents a image registry client that can list available tags
// for image URLs.
type ImageClient interface {

	// Name returns the client name, can also have the URI in question
	Name() string

	// RepoImage will return the registries repository and image, from a given
	// URL path.
	RepoImageFromPath(path string) (string, string)

	// Tags will return the available tags for the given host, repo, and image
	// using that client.
	Tags(ctx context.Context, host, repo, image string) ([]ImageTag, error)
}

type ImageClientFactory interface {
	// Name returns the client name, can also have the URI in question
	Name() string

	// IsHost returns true if the client is configured for the given host.
	IsHost(host string) bool

	// New creates an instance of said client, with authentication and logging.
	NewClient(auth *authn.AuthConfig, log *logrus.Entry) (ImageClient, error)

	// Implementing the Resolve func to match that of a authn.Keychain
	Resolve(res authn.Resource) (authn.Authenticator, error)
}
