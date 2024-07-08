package oci

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/jetstack/version-checker/pkg/api"
)

// Client is a client for a registry compatible with the OCI Distribution Spec
type Client struct {
	puller *remote.Puller
}

// New returns a new client
func New() (*Client, error) {
	puller, err := remote.NewPuller()
	if err != nil {
		return nil, fmt.Errorf("creating puller: %w", err)
	}

	return &Client{
		puller: puller,
	}, nil
}

// Name is the name of this client
func (c *Client) Name() string {
	return "oci"
}

// Tags lists all the tags in the specified repository
func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	reg, err := name.NewRegistry(host)
	if err != nil {
		return nil, fmt.Errorf("parsing registry host: %w", err)
	}

	bareTags, err := c.puller.List(ctx, reg.Repo(repo, image))
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	var tags []api.ImageTag
	for _, t := range bareTags {
		tags = append(tags, api.ImageTag{Tag: t})
	}

	return tags, nil
}

// IsHost always returns true because it supports any host
func (c *Client) IsHost(_ string) bool {
	return true
}

// RepoImageFromPath splits a repository path into 'repo' and 'image' segments
func (c *Client) RepoImageFromPath(path string) (string, string) {
	split := strings.Split(path, "/")

	lenSplit := len(split)
	if lenSplit == 1 {
		return "", split[0]
	}

	if lenSplit > 1 {
		return split[lenSplit-2], split[lenSplit-1]
	}

	return path, ""
}
