package oci

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

type Client struct{}

func New() (*Client, error) {
	return &Client{}, nil
}

func (c *Client) Name() string {
	return "oci"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	src := fmt.Sprintf("%s/%s/%s", host, repo, image)
	rpo, err := name.NewRepository(src)
	if err != nil {
		return []api.ImageTag{}, err
	}

	bareTags, err := remote.List(rpo, remote.WithContext(ctx))
	if err != nil {
		return []api.ImageTag{}, err
	}

	var tags []api.ImageTag
	for _, t := range bareTags {
		// Filter SBOMS, Attestations, Signiture Tags
		if util.FilterSbomAttestationSigs(t) {
			continue
		}
		tags = append(tags, api.ImageTag{Tag: t})
	}

	return tags, nil
}

func (c *Client) IsHost(host string) bool {
	return true
}

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
