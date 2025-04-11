package oci

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"

	"github.com/jetstack/version-checker/pkg/api"
)

var numWorkers = runtime.NumCPU() * 5

type Options struct {
	Transporter http.RoundTripper
	Auth        *authn.AuthConfig
}

func (o *Options) Authorization() (*authn.AuthConfig, error) {
	if o.Auth != nil {
		return o.Auth, nil
	}
	return authn.Anonymous.Authorization()
}

// Client is a client for a registry compatible with the OCI Distribution Spec
type Client struct {
	*Options
	log    *logrus.Entry
	puller *remote.Puller
}

// Ensure that we are an ImageClient
var _ api.ImageClient = (*Client)(nil)

// New returns a new client
func New(opts *Options, log *logrus.Entry) (*Client, error) {
	pullOpts := []remote.Option{
		remote.WithJobs(numWorkers),
		remote.WithUserAgent("version-checker/oci"),
		remote.WithAuth(opts),
	}
	if opts.Transporter != nil {
		pullOpts = append(pullOpts, remote.WithTransport(opts.Transporter))
	}

	puller, err := remote.NewPuller(pullOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating puller: %w", err)
	}

	return &Client{
		puller:  puller,
		log:     log.WithField("client", "OCI"),
		Options: opts,
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
	c.log.Infof("Collected %v tags..", len(bareTags))
	return c.Manifests(ctx, reg.Repo(repo, image), bareTags)
}

func (c *Client) Manifests(ctx context.Context, repo name.Repository, tags []string) (fulltags []api.ImageTag, err error) {
	wg := sync.WaitGroup{}
	sem := make(chan struct{}, numWorkers) // limit concurrent fetches
	wg.Add(len(tags))
	mu := sync.Mutex{}

	// Lets lookup all the child Manifests (where applicable)
	for _, tag := range tags {
		go func(repo name.Repository, tag string) {
			log := c.log.WithFields(logrus.Fields{"tag": tag, "repo": repo.Name()})
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Parse the Tag
			t, err := name.NewTag(repo.Name() + ":" + tag)
			if err != nil {
				log.Errorf("parsing Tag: %s", err)
				return
			}

			// Fetch the manifest
			manifest, err := c.puller.Get(ctx, t)
			if err != nil {
				log.Errorf("getting manifest: %s", err)
				return
			}

			// Lock when we have the data!
			mu.Lock()
			defer mu.Unlock()

			ts, err := discoverTimestamp(manifest.Annotations)
			if err != nil {
				log.Errorf("Unable to discover Timestamp: %s", err)
				return
			}

			baseTag := api.ImageTag{
				Tag:       tag,
				Timestamp: ts,
			}

			// We have a suitable Image Index!
			if manifest.MediaType == types.OCIImageIndex || manifest.MediaType == types.DockerManifestList {
				children := []*api.ImageTag{}
				imgidx, err := manifest.ImageIndex()
				if err != nil {
					log.Errorf("getting imageindex: %s", err)
					return
				}
				idxman, err := imgidx.IndexManifest()
				for _, img := range idxman.Manifests {

					children = append(children, &api.ImageTag{
						Tag: tag,
						SHA: img.Digest.String(),
					})
				}
				baseTag.Children = children
			} else if manifest.MediaType == types.OCIManifestSchema1 || manifest.MediaType == types.DockerManifestSchema2 {
				img, err := manifest.Image()
				if err != nil {
					log.Errorf("unable to collect image from manifest: %s", err)
					return
				}
				sha, err := img.Digest()
				if err != nil {
					log.Errorf("unable to collect digest from manifest: %s", err)
					return
				}
				baseTag.SHA = sha.String()
			}

			// Add it to the full tags
			fulltags = append(fulltags, baseTag)
		}(repo, tag)
	}
	// Wait for everything to complete!
	wg.Wait()

	return fulltags, err
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
		return strings.Join(split[:len(split)-1], "/"), split[lenSplit-1]
	}

	return path, ""
}
