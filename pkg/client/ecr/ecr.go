package ecr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/sirupsen/logrus"

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

	var tags []api.ImageTag
	for _, img := range images.ImageDetails {

		tags = append(tags, api.ImageTag{
			SHA:       *img.ImageDigest,
			Timestamp: *img.ImagePushedAt,
		})

		// Continue early if no tags available
		if len(img.ImageTags) == 0 {
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
	tags = getOSArchDetails(client, repoName, tags)

	return tags, nil
}

func (c *Client) createClient(ctx context.Context, region string) (*ecr.Client, error) {
	var cfg aws.Config
	var err error

	if c.IamRoleArn != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, c.SessionToken)),
			config.WithRegion(region),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to construct aws credentials: %s", err)
	}
	return ecr.NewFromConfig(cfg), nil
}

type ImageManifest struct {
	Config struct {
		Digest string `json:"digest"`
	} `json:"config"`
}

type ImageConfig struct {
	OS   string `json:"os"`
	Arch string `json:"architecture"`
}

func getOSArchDetails(client *ecr.Client, repositoryName string, tags []api.ImageTag) []api.ImageTag {

	// AWS only accept 100 tags at a time
	tagGroups := splitTags(tags, 100)

	for _, tags := range tagGroups {
		var imageIds []ecrtypes.ImageIdentifier
		for _, tag := range tags {
			imageIds = append(imageIds, ecrtypes.ImageIdentifier{
				ImageDigest: aws.String(tag.SHA),
			})
		}

		manifestInput := &ecr.BatchGetImageInput{
			RepositoryName:     aws.String(repositoryName),
			ImageIds:           imageIds,
			AcceptedMediaTypes: []string{"application/vnd.docker.distribution.manifest.v2+json"},
		}

		manifestOutput, err := client.BatchGetImage(context.Background(), manifestInput)
		if err != nil {
			log.Fatalf("failed to get image manifest: %v", err)
		}

		if len(manifestOutput.Images) > 0 {
			manifest := manifestOutput.Images[0].ImageManifest

			// Parse the image manifest
			var imageManifest ImageManifest
			err = json.Unmarshal([]byte(*manifest), &imageManifest)
			if err != nil {
				log.Fatalf("failed to unmarshal image manifest: %v", err)
			}

			// Get image configuration
			configRequest := &ecr.BatchGetImageInput{
				RepositoryName:     aws.String(repositoryName),
				ImageIds:           imageIds,
				AcceptedMediaTypes: []string{"application/vnd.docker.container.image.v1+json"},
			}

			configResponse, err := client.BatchGetImage(context.Background(), configRequest)
			if err != nil {
				log.Fatalf("failed to get image configuration: %v", err)
			}

			if len(configResponse.Images) > 0 {
				for _, cfg := range configResponse.Images {

					config := cfg.ImageManifest
					sha := cfg.ImageId.ImageDigest

					var imageConfig ImageConfig
					err = json.Unmarshal([]byte(*config), &imageConfig)
					if err != nil {
						log.Fatalf("failed to unmarshal image configuration: %v", err)
					}

					// We need to go back through, ALL of our tags
					// and enrich the OS and Architecture fields
					found := false
					for _, tag := range tags {
						if tag.SHA == *sha || tag.Tag == *cfg.ImageId.ImageTag {
							tag.OS = api.OS(imageConfig.OS) // Convert string to api.OS
							tag.Architecture = api.Architecture(imageConfig.Arch)
							found = true
						}
					}
					if !found {
						logrus.Warnf("failed to find ImageConfig for image digest: %s", *sha)
						tags = append(tags, api.ImageTag{
							SHA:          *sha,
							Tag:          *cfg.ImageId.ImageTag,
							OS:           api.OS(imageConfig.OS),
							Architecture: api.Architecture(imageConfig.Arch),
						})
					}
				}
			}
		}
	}

	return tags
}

// splitTags splits a slice of ImageTags into groups of a specified size
func splitTags(slice []api.ImageTag, size int) [][]api.ImageTag {
	var result [][]api.ImageTag
	for size < len(slice) {
		slice, result = slice[size:], append(result, slice[0:size:size])
	}
	result = append(result, slice)
	return result
}
