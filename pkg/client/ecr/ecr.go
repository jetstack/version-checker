package ecr

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/jetstack/version-checker/pkg/api"
)

type Client struct {
	cacheMu             sync.Mutex
	cachedRegionClients map[string]*ecr.ECR

	Options
}

type Options struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

func New(opts Options) *Client {
	return &Client{
		Options:             opts,
		cachedRegionClients: make(map[string]*ecr.ECR),
	}
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	matches := ecrPattern.FindStringSubmatch(host)
	if len(matches) < 3 {
		return nil, fmt.Errorf("aws client not suitable for image host: %s", host)
	}

	id := matches[1]
	region := matches[3]

	client, err := c.getClient(region)
	if err != nil {
		return nil, fmt.Errorf("failed to construct ecr client for image host %s: %s",
			host, err)
	}

	images, err := client.DescribeImagesWithContext(ctx, &ecr.DescribeImagesInput{
		RepositoryName: joinRepoImage(repo, image),
		RegistryId:     aws.String(id),
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
				Tag:       *tag,
			})
		}
	}

	return tags, nil
}

func (c *Client) getClient(region string) (*ecr.ECR, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	var err error

	client, ok := c.cachedRegionClients[region]
	if !ok || client == nil || client.Config.Credentials.IsExpired() {

		// If the client is not yet created, or the token has expired, create new.
		client, err = c.createRegionClient(region)
		if err != nil {
			return nil, err
		}
	}

	c.cachedRegionClients[region] = client
	return client, nil
}

func (c *Client) createRegionClient(region string) (*ecr.ECR, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(c.AccessKeyID, c.SecretAccessKey, c.SessionToken),
		Region:      &region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to construct aws credentials: %s", err)
	}

	return ecr.New(sess, sess.Config), nil
}

func joinRepoImage(repo, image string) *string {
	if len(repo) == 0 {
		return &image
	}
	if len(image) == 0 {
		return &repo
	}

	return aws.String(repo + "/" + image)
}
