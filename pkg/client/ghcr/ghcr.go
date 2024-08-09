package ghcr

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v62/github"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

type Options struct {
	Token string
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
	ghRateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil, ghRatelimitOpts)
	if err != nil {
		panic(err)
	}
	client := github.NewClient(ghRateLimiter)
	// Only add Auth Token if it is provided.
	if len(opts.Token) > 0 {
		client = client.WithAuthToken(opts.Token)
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

func (c *Client) Tags(ctx context.Context, host, owner, repo string) ([]api.ImageTag, error) {
	// Choose the correct list packages function based on whether the owner
	// is a user or an organization
	// getReleases := c.Client.Repositories.ListReleases(ctx, owner, repo)
	getAllVersions := c.client.Organizations.PackageGetAllVersions
	ownerType, err := c.ownerType(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("fetching owner type: %w", err)
	}
	if ownerType == "user" {
		getAllVersions = c.client.Users.PackageGetAllVersions
		// The User implementation doesn't path escape this for you:
		// - https://github.com/google/go-github/blob/v58.0.0/github/users_packages.go#L136
		// - https://github.com/google/go-github/blob/v58.0.0/github/orgs_packages.go#L105
		repo = url.PathEscape(repo)
	}

	opts := &github.PackageListOptions{
		PackageType: github.String("container"),
		State:       github.String("active"),
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var tags []api.ImageTag

	for {
		versions, resp, err := getAllVersions(ctx, owner, "container", repo, opts)
		if err != nil {
			return nil, fmt.Errorf("getting versions: %w", err)
		}

		for _, ver := range versions {
			if len(ver.Metadata.Container.Tags) == 0 {
				continue
			}

			sha := ""
			if strings.HasPrefix(*ver.Name, "sha") {
				sha = *ver.Name
			}

			for _, tag := range ver.Metadata.Container.Tags {
				// Exclude attestations, signatures and sboms
				if util.FilterSbomAttestationSigs(tag) {
					continue
				}

				tags = append(tags, api.ImageTag{
					Tag:       tag,
					SHA:       sha,
					Timestamp: ver.CreatedAt.Time,
				})
			}
		}
		if resp.NextPage == 0 {
			break
		}

		opts.ListOptions.Page = resp.NextPage
	}

	return tags, nil
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
