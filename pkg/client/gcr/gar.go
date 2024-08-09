package gcr

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
	"github.com/jetstack/version-checker/pkg/api"
	"google.golang.org/api/iterator"
)

func (c *Client) listGARTags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	var tags []api.ImageTag

	// Construct the parent path
	parent := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/dockerImages/%s", extractProjectID(repo), extractLocation(host), extractRepo(repo), image)

	// Create the request
	req := &artifactregistrypb.ListTagsRequest{
		Parent: parent,
	}

	// Call the API to list docker images (manifests)
	it := c.GAR.ListTags(ctx, req)
	fmt.Println("GAR Tags:")
	for {
		tag, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		fmt.Println(tag.GetVersion())

		// Get manifest details for each tag
		// c.GAR.printGARManifestDetails(tag.GetVersion())
	}

	return tags, nil
}

func extractRepo(repo string) string {
	parts := strings.Split(repo, "/")
	return parts[1]
}

// Helper function to extract the project ID from the host
func extractProjectID(repo string) string {
	parts := strings.Split(repo, "/")
	return parts[0]
}

// Helper function to extract the location from the host
func extractLocation(host string) string {
	parts := strings.Split(host, "-")
	return strings.Join(parts[:len(parts)-1], "-")
}
