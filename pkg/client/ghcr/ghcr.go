package ghcr

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofri/go-github-ratelimit/github_ratelimit"
	"github.com/google/go-github/v58/github"
	"github.com/jetstack/version-checker/pkg/api"
)

type Options struct {
	Token string
}

type Client struct {
	*http.Client
	Options
}

func New(opts Options) *Client {
	return &Client{
		Options: opts,
		Client: &http.Client{
			Timeout: time.Second * 5,
		},
	}
}

func (c *Client) Name() string {
	return "ghcr"
}

func (c *Client) Tags(ctx context.Context, host, owner, repo string) ([]api.ImageTag, error) {
	var err error
	var tags []api.ImageTag

	rateLimitDetection := func(ctx *github_ratelimit.CallbackContext) {
		fmt.Printf("Hit Github Rate Limit, sleeping for %v", ctx.TotalSleepTime)
	}

	ghRatelimitOpts := github_ratelimit.WithLimitDetectedCallback(rateLimitDetection)
	ghRateLimiter, err := github_ratelimit.NewRateLimitWaiterClient(nil, ghRatelimitOpts)
	if err != nil {
		panic(err)
	}
	client := github.NewClient(ghRateLimiter).WithAuthToken(c.Token)

	if repoExist(client, owner, repo) {
		tags, err = getTagsFromRelease(client, owner, repo)
		if err != nil {
			return nil, err
		}
	} else {

		tags, err = getTagsFromOrgPackage(client, owner, repo)
		if err != nil {
			return nil, err
		}
	}
	return tags, nil
}

func repoExist(client *github.Client, owner string, repo string) bool {
	_, _, err := client.Repositories.Get(context.TODO(), owner, repo)
	return err == nil
}

func getTagsFromOrgPackage(client *github.Client, owner string, repo string) ([]api.ImageTag, error) {
	var tags []api.ImageTag
	packageType := "container"
	packageState := "active"
	opts := &github.PackageListOptions{
		PackageType: &packageType,
		State:       &packageState,
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		versions, resp, err := client.Organizations.PackageGetAllVersions(context.TODO(), owner, packageType, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get Org Package Versions: %s", err)
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
				if strings.HasSuffix(tag, ".att") { // Skip tags that are attestations
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

func getTagsFromRelease(client *github.Client, owner string, repo string) ([]api.ImageTag, error) {
	var tags []api.ImageTag
	opt := &github.ListOptions{PerPage: 50}
	for {
		releases, resp, err := client.Repositories.ListReleases(context.TODO(), owner, repo, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to get github Releases: %s", err)
		}

		for _, rel := range releases {
			if rel.TagName == nil {
				continue
			}
			tags = append(tags, api.ImageTag{
				Tag:       *rel.TagName,
				SHA:       "",
				Timestamp: rel.PublishedAt.Time,
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return tags, nil
}
