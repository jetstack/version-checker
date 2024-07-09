package quay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

const (
	tagURL      = "https://quay.io/api/v1/repository/%s/%s/tag/?page=%d"
	manifestURL = "https://quay.io/api/v1/repository/%s/%s/manifest/%s"
)

func New(opts Options) *Client {
	client := retryablehttp.NewClient()
	client.RetryMax = 10
	client.Logger = nil

	return &Client{
		Options: opts,
		Client:  client,
	}
}

func (c *Client) Name() string {
	return "quay"
}

// Fetch the image tags from an upstream repository and image
func (c *Client) Tags(ctx context.Context, _, repo, image string) ([]api.ImageTag, error) {
	p := c.newPager(repo, image)

	if err := p.fetchTags(ctx); err != nil {
		return nil, err
	}

	return p.tags, nil
}

// fetchImageManifest will lookup all manifests for a tag, if it is a list
func (c *Client) fetchImageManifest(ctx context.Context, repo, image string, tag *responseTagItem) ([]api.ImageTag, error) {
	timestamp, err := time.Parse(time.RFC1123Z, tag.LastModified)
	if err != nil {
		return nil, err
	}
	// Filter Sbom, Attestations, Sigs
	if util.FilterSbomAttestationSigs(tag.Name) {
		return []api.ImageTag{}, nil
	}

	// If a multi-arch image, call manifest endpoint
	if tag.IsManifestList {
		url := fmt.Sprintf(manifestURL, repo, image, tag.ManifestDigest)
		tags, err := c.callManifests(ctx, timestamp, tag.Name, url)
		if err != nil {
			return nil, err
		}

		return tags, nil
	}

	// Fallback to not using multi-arch image
	os, arch := util.OSArchFromTag(tag.Name)

	return []api.ImageTag{
		{
			Tag:          tag.Name,
			SHA:          tag.ManifestDigest,
			Timestamp:    timestamp,
			OS:           os,
			Architecture: arch,
		},
	}, nil
}

// callManifests endpoint on the tags image manifest
func (c *Client) callManifests(ctx context.Context, timestamp time.Time, tag, url string) ([]api.ImageTag, error) {
	var manifestResp responseManifest
	if err := c.makeRequest(ctx, url, &manifestResp); err != nil {
		return nil, err
	}

	// Got error on this manifest, ignore
	if manifestResp.Status != nil {
		return nil, nil
	}

	var manifestData responseManifestData
	if err := json.Unmarshal([]byte(manifestResp.ManifestData), &manifestData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest data %s: %#+v: %s",
			tag, manifestResp, err)
	}

	var tags []api.ImageTag
	for _, manifest := range manifestData.Manifests {
		if util.FilterSbomAttestationSigs(tag) {
			continue
		}
		tags = append(tags, api.ImageTag{
			Tag:          tag,
			SHA:          manifest.Digest,
			Timestamp:    timestamp,
			Architecture: manifest.Platform.Architecture,
			OS:           manifest.Platform.OS,
		})
	}

	return tags, nil
}

// makeRequest will make a call and write the response to the object.
// Implements backoff.
func (c *Client) makeRequest(ctx context.Context, url string, obj interface{}) error {
	req, err := retryablehttp.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if len(c.Token) > 0 {
		req.Header.Add("Authorization", "Bearer "+c.Token)
	}

	req.URL.Scheme = "https"
	req = req.WithContext(ctx)

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make quay call %q: %s", url, err)
	}

	if err := json.NewDecoder(resp.Body).Decode(obj); err != nil {
		return err
	}

	return nil
}
