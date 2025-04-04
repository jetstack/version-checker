package ecr

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

// Ensure that we are an ImageClient
var _ api.ImageClient = (*Client)(nil)

type Client struct {
	Config aws.Config

	Options
}

type Options struct {
	IamRoleArn      string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Transporter     http.RoundTripper
}

func New(opts Options) *Client {
	return &Client{
		Options: opts,
	}
}

func (c *Client) Name() string {
	return "ecr"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	matches := ecrPattern.FindStringSubmatch(host)
	if len(matches) < 3 {
		return nil, fmt.Errorf("aws client not suitable for image host: %s", host)
	}

	id := matches[1]
	region := matches[3]

	client, err := c.createClient(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to construct ecr client for image host %s: %s",
			host, err)
	}

	repoName := util.JoinRepoImage(repo, image)

	images, err := client.DescribeImages(ctx, &ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		RegistryId:     aws.String(id),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to describe images: %s", err)
	}

	tags := map[string]api.ImageTag{}
	for _, img := range images.ImageDetails {
		// Base data shared across tags
		base := api.ImageTag{
			SHA:       *img.ImageDigest,
			Timestamp: *img.ImagePushedAt,
		}

		// Continue early if no tags available
		if len(img.ImageTags) == 0 {
			tags[base.SHA] = base
			continue
		}

		for _, tag := range img.ImageTags {
			current := base   // copy the base
			current.Tag = tag // set tag value

			// Already exists — add as child
			if parent, exists := tags[tag]; exists {
				parent.Children = append(parent.Children, &current)
				tags[tag] = parent
			} else {
				// First occurrence — assign as root
				tags[tag] = current
			}
		}
	}

	// Convert Map to Slice
	taglist := make([]api.ImageTag, 0, len(tags))
	for _, tag := range tags {
		taglist = append(taglist, tag)
	}

	return taglist, nil
}

func (c *Client) createClient(ctx context.Context, region string) (*ecr.Client, error) {
	var cfg aws.Config
	var err error

	if c.IamRoleArn != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithHTTPClient(&http.Client{Transport: c.Transporter}),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, c.SessionToken),
			),
			config.WithRegion(region),
			config.WithHTTPClient(&http.Client{Transport: c.Transporter}),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to construct aws credentials: %s", err)
	}
	return ecr.NewFromConfig(cfg), nil
}
