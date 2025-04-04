package ghcr

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jetstack/version-checker/pkg/api"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v70/github"
)

// Ensure that we are an ImageClient
var _ api.ImageClient = (*Client)(nil)

type Options struct {
	Token       string
	Hostname    string
	Transporter http.RoundTripper
}

type Client struct {
	client     *github.Client
	opts       Options
	ownerTypes map[string]string
}

func New(opts Options) *Client {
	rateLimitDetection := func(ctx *github_ratelimit.CallbackContext) {
		fmt.Printf("Hit Github Rate Limit, sleeping for %v", ctx.TotalSleepTime)
	}

	ghRatelimitOpts := github_ratelimit.WithLimitDetectedCallback(rateLimitDetection)
	ghRateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(opts.Transporter, ghRatelimitOpts)
	if err != nil {
		panic(err)
	}
	client := github.NewClient(ghRateLimiter).WithAuthToken(opts.Token)
	if opts.Hostname != "" {
		client, err = client.WithEnterpriseURLs(fmt.Sprintf("https://%s/", opts.Hostname), fmt.Sprintf("https://%s/api/uploads/", opts.Hostname))
		if err != nil {
			panic(fmt.Errorf("failed setting enterprise URLs: %w", err))
		}
	}

	return &Client{
		client:     client,
		opts:       opts,
		ownerTypes: map[string]string{},
	}
}

func (c *Client) Name() string {
	return "ghcr"
}

func (c *Client) Tags(ctx context.Context, _, owner, repo string) ([]api.ImageTag, error) {
	// Determine the correct function to get all versions based on the owner type
	getAllVersions, repo, err := c.determineGetAllVersionsFunc(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	opts := c.buildPackageListOptions()

	var tags []api.ImageTag
	for {
		versions, resp, err := getAllVersions(ctx, owner, "container", repo, opts)
		if err != nil {
			return nil, fmt.Errorf("getting versions: %w", err)
		}

		tags = append(tags, c.extractImageTags(versions)...)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return tags, nil
}

func (c *Client) determineGetAllVersionsFunc(ctx context.Context, owner, repo string) (func(ctx context.Context, owner, pkgType, repo string, opts *github.PackageListOptions) ([]*github.PackageVersion, *github.Response, error), string, error) {
	getAllVersions := c.client.Organizations.PackageGetAllVersions
	ownerType, err := c.ownerType(ctx, owner)
	if err != nil {
		return nil, "", fmt.Errorf("fetching owner type: %w", err)
	}
	if ownerType == "user" {
		getAllVersions = c.client.Users.PackageGetAllVersions
		repo = url.PathEscape(repo)
	}
	return getAllVersions, repo, nil
}

func (c *Client) buildPackageListOptions() *github.PackageListOptions {
	return &github.PackageListOptions{
		PackageType: github.Ptr("container"),
		State:       github.Ptr("active"),
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
}

func (c *Client) extractImageTags(versions []*github.PackageVersion) []api.ImageTag {
	tags := map[string]api.ImageTag{}
	for _, ver := range versions {
		if meta, ok := ver.GetMetadata(); ok {

			var sha string
			if strings.HasPrefix(*ver.Name, "sha") {
				sha = *ver.Name
			}

			base := api.ImageTag{
				Tag:       *ver.Name,
				SHA:       sha,
				Timestamp: ver.CreatedAt.Time,
			}

			for _, tag := range meta.Container.Tags {
				current := base   // copy the base
				current.Tag = tag // set tag value

				// Tag Already exists — add as child
				if parent, exists := tags[tag]; exists {
					parent.Children = append(parent.Children, &current)
					tags[tag] = parent
				} else {
					// First occurrence of Tag — assign as root
					tags[tag] = current
				}
			}
		}
	}

	// Convert Map to Slice
	taglist := make([]api.ImageTag, 0, len(tags))
	for _, tag := range tags {
		taglist = append(taglist, tag)
	}
	return taglist
}

func (c *Client) ownerType(ctx context.Context, owner string) (string, error) {
	if ownerType, ok := c.ownerTypes[owner]; ok {
		return ownerType, nil
	}
	user, _, err := c.client.Users.Get(ctx, owner)
	if err != nil {
		return "", fmt.Errorf("fetching user: %w", err)
	}
	ownerType := strings.ToLower(user.GetType())

	c.ownerTypes[owner] = ownerType

	return ownerType, nil
}
