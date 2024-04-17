package ghcr

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v58/github"
	"github.com/jetstack/version-checker/pkg/api"
)

type Options struct {
	Token string
}

type Client struct {
	client *github.Client
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
	client := github.NewClient(ghRateLimiter).WithAuthToken(opts.Token)

	return &Client{
		client: client,
	}
}

func (c *Client) Name() string {
	return "ghcr"
}

func (c *Client) Tags(ctx context.Context, host, owner, repo string) ([]api.ImageTag, error) {
	// Choose the correct list packages function based on whether the owner
	// is a user or an organization
	getAllVersions := c.client.Organizations.PackageGetAllVersions
	user, _, err := c.client.Users.Get(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}
	if strings.ToLower(user.GetType()) == "user" {
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
				if strings.HasSuffix(tag, ".att") {
					continue
				}
				if strings.HasSuffix(tag, ".sig") {
					continue
				}
				if strings.HasSuffix(tag, ".sbom") {
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
