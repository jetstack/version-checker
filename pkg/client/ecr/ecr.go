package ecr

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

// Ensure that we are an ImageClient
var _ api.ImageClient = (*Client)(nil)

type Client struct {
	Options
	Config aws.Config

	// callerAccounts caches the STS caller account per region.
	callerAccounts sync.Map
}

type Options struct {
	Transporter     http.RoundTripper
	IamRoleArn      string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string

	// AllowCrossAccount permits lookups in registries owned by other AWS accounts.
	AllowCrossAccount bool
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

	client, cfg, err := c.createClient(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to construct ecr client for image host %s: %s",
			host, err)
	}

	if !c.AllowCrossAccount {
		account, ok := c.callerAccounts.Load(region)
		if !ok {
			out, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
			if err != nil {
				return nil, fmt.Errorf("failed to determine own aws account: %w", err)
			}
			account = aws.ToString(out.Account)
			c.callerAccounts.Store(region, account)
		}
		if account != id {
			return nil, fmt.Errorf("refusing cross-account ecr lookup for account %s (set --ecr-allow-cross-account to permit)", id)
		}
	}

	repoName := util.JoinRepoImage(repo, image)

	tags := map[string]api.ImageTag{}
	input := &ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		RegistryId:     aws.String(id),
	}

	for {
		images, err := client.DescribeImages(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe images: %w", err)
		}

		for _, img := range images.ImageDetails {
			// Base data shared across tags
			base := api.ImageTag{
				SHA:       aws.ToString(img.ImageDigest),
				Timestamp: aws.ToTime(img.ImagePushedAt),
			}

			// Continue early if no tags available
			if len(img.ImageTags) == 0 {
				tags[base.SHA] = base
				continue
			}

			for _, tag := range img.ImageTags {
				current := base   // copy the base
				current.Tag = tag // set tag value

				util.BuildTags(tags, tag, &current)
			}
		}

		if aws.ToString(images.NextToken) == "" {
			break
		}
		input.NextToken = images.NextToken
	}

	return util.TagMaptoList(tags), nil
}

func (c *Client) createClient(ctx context.Context, region string) (*ecr.Client, aws.Config, error) {
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
		return nil, cfg, fmt.Errorf("failed to construct aws credentials: %s", err)
	}
	return ecr.NewFromConfig(cfg), cfg, nil
}
