package oci

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
)

type Options struct {
	Transporter http.RoundTripper
}

var _ api.ImageClient = (*Client)(nil)

// Client is a client for a registry compatible with the OCI Distribution Spec
type Client struct {
	*Options
	puller *remote.Puller
	log    *logrus.Entry
}

// New returns a new client
func NewClient(opts *Options, auth *authn.AuthConfig, log *logrus.Entry) (*Client, error) {
	pullOpts := []remote.Option{
		remote.WithJobs(runtime.NumCPU()),
		remote.WithUserAgent("version-checker"),
		remote.WithAuth(
			// We need to convert it back to an Authenticator for the Puller
			authn.FromConfig(*auth),
		),
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
		Options: opts,
		log:     log.WithField("client", "oci"),
	}, nil
}

// Name is the name of this client
func (c *Client) Name() string {
	return "oci"
}

// Tags lists all the tags in the specified repository
func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	log := c.log.WithField("host", host)
	reg, err := name.NewRegistry(host)
	if err != nil {
		return nil, fmt.Errorf("parsing registry host: %w", err)
	}

	log.Debugf("Listing tags for %s/%s", repo, image)
	bareTags, err := c.puller.List(ctx, reg.Repo(repo, image))
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}
	log.Debugf("Found %v Tags for %s/%s", len(bareTags), repo, image)

	var tags []api.ImageTag
	for _, tag := range bareTags {
		tlog := log.WithField("tag", tag)
		tlog.Debug("Getting descriptor")

		imgref := reg.Repo(repo, image).Tag(tag)
		desc, err := c.puller.Get(ctx, imgref)
		if err != nil {
			return nil, fmt.Errorf("getting descriptor for tag %s: %w", tag, err)
		}

		// If we detect a OCI or Docker Manifest Index.. lets process as is
		if desc.MediaType.IsIndex() {
			tlog.Debug("Discovered image index")
			imgidx, err := desc.ImageIndex()
			if err != nil {
				return nil, fmt.Errorf("getting imageIndex for tag %s: %w", tag, err)
			}
			tlog.Debug("Discovering IndexManifest")
			idxMan, err := imgidx.IndexManifest()
			if err != nil {
				return nil, fmt.Errorf("getting index manifest for tag %s: %w", tag, err)
			}

			tlog.WithField("count", len(idxMan.Manifests)).Debug("Found Manifests")
			for _, man := range idxMan.Manifests {
				// We need to skip attestation-manifests..
				if man.Annotations["vnd.docker.reference.type"] == "attestation-manifest" ||
					man.Annotations["dev.cosignproject.sigstore/attestation-type"] != "" {
					tlog.WithField("digest", man.Digest.String()).Debug("Skipping attestation-manifest")
					continue
				}

				tags = append(tags, api.ImageTag{
					Tag:          tag,
					SHA:          man.Digest.String(),
					Architecture: api.Architecture(man.Platform.Architecture),
					OS:           api.OS(man.Platform.OS),
				})
			}
			// We continue, as to not create duplicates
			continue
		}

		tags = append(tags,
			api.ImageTag{
				Tag: tag,
				SHA: desc.Digest.String(),
			})
	}
	c.log.WithField("tags", tags).Debugf("Tags Discovered...")

	return tags, nil
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
