package api

import (
	"context"
	"time"
)

// ImageTag describes a container image tag.
type ImageTag struct {
	Tag          string       `json:"tag"`
	SHA          string       `json:"sha"`
	Timestamp    time.Time    `json:"timestamp"`
	OS           OS           `json:"os,omitempty"`
	Architecture Architecture `json:"architecture,omitempty"`

	// If this is a Manifest list we need to keep them together
	Children []*ImageTag `json:"children,omitempty"`
}

func (i *ImageTag) HasChildren() bool {
	return len(i.Children) > 0
}

func (i *ImageTag) MatchesSHA(sha string) bool {
	if sha == i.SHA {
		return true
	}
	for _, known := range i.Children {
		if known.MatchesSHA(sha) {
			return true
		}
	}
	return false
}

type Platform struct {
	OS           OS
	Architecture Architecture
}

type OS string
type Architecture string

// ImageClient represents a image registry client that can list available tags
// for image URLs.
type ImageClient interface {
	// Returns the name of the client
	Name() string

	// IsHost will return true if this client is appropriate for the given
	// host.
	IsHost(host string) bool

	// RepoImage will return the registries repository and image, from a given
	// URL path.
	RepoImageFromPath(path string) (string, string)

	// Tags will return the available tags for the given host, repo, and image
	// using that client.
	Tags(ctx context.Context, host, repo, image string) ([]ImageTag, error)
}
