package ghcr

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v53/github"
	"github.com/jetstack/version-checker/pkg/api"
	"golang.org/x/oauth2"
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
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	var tags []api.ImageTag
	opt := &github.ListOptions{PerPage: 50}
	for {
		releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, opt)
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
