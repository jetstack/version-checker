package gcr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jetstack/version-checker/pkg/api"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"

	artifactregistry "cloud.google.com/go/artifactregistry/apiv1"
)

type Options struct {
	Token string
}

type Client struct {
	GAR *artifactregistry.Client
	GCR *retryablehttp.Client
	Options
}

func New(opts Options) *Client {
	retryableClient := retryablehttp.NewClient()
	retryableClient.RetryMax = 5                         // Set the number of retry attempts
	retryableClient.HTTPClient.Timeout = 5 * time.Second // Set the HTTP client timeout
	ctx := context.Background()
	var garClient *artifactregistry.Client

	if opts.Token == "" {
		// Create an HTTP client with ID token authentication
		idtokenClient, _ := idtoken.NewClient(ctx, "https://gcr.io")
		retryableClient.HTTPClient = idtokenClient

		// GAR Client by default does automatic token refresh
		garClient, _ = artifactregistry.NewClient(ctx)
	} else {
		garClient, _ = artifactregistry.NewClient(ctx,
			option.WithTokenSource(
				oauth2.StaticTokenSource(&oauth2.Token{AccessToken: opts.Token})))
	}

	return &Client{
		Options: opts,
		GAR:     garClient,
		GCR:     retryableClient,
	}
}

func (c *Client) Name() string {
	return "gcp"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {

	if strings.Contains(host, "gcr.io") {
		// Handle GCR
		return c.listGCRTags(ctx, host, repo, image)
	} else if strings.Contains(host, "pkg.dev") {
		// Handle GAR
		return c.listGARTags(ctx, host, repo, image)
	}

	return nil, fmt.Errorf("unknown registry type for image path: %s/%s/%s", host, repo, image)
}
