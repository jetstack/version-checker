package ecrpublic

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/aws/aws-sdk-go-v2/service/ecrpublic"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

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
	return "ecrpublic"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	client, err := c.createClient(ctx, "us-east-1")
	if err != nil {
		return nil, err
	}
	repoName := util.JoinRepoImage(repo, image)
	images, err := client.DescribeImages(ctx, &ecrpublic.DescribeImagesInput{
		RepositoryName: &repoName,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to describe images: %s", err)
	}

	var tags []api.ImageTag
	for _, img := range images.ImageDetails {
		// Continue early if no tags available
		if len(img.ImageTags) == 0 {
			tags = append(tags, api.ImageTag{
				SHA:       *img.ImageDigest,
				Timestamp: *img.ImagePushedAt,
			})

			continue
		}

		for _, tag := range img.ImageTags {
			tags = append(tags, api.ImageTag{
				SHA:       *img.ImageDigest,
				Timestamp: *img.ImagePushedAt,
				Tag:       tag,
			})
		}
	}
	// For public ECR, RegistryId is not required, so id can be left empty

	return tags, nil
}

func (c *Client) createClient(ctx context.Context, region string) (*ecrpublic.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, err
	}

	client := ecrpublic.NewFromConfig(cfg)
	return client, nil
}
