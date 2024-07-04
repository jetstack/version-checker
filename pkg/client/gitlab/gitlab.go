package gitlab

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/jetstack/version-checker/pkg/api"

	gitlab "github.com/xanzy/go-gitlab"
)

const (
	gitlabBaseURL = "https://gitlab.com"
)

type Options struct {
	Host      string
	Username  string
	Password  string
	Bearer    string
	TokenPath string
	Insecure  bool
	CAPath    string
}

type Client struct {
	Client *gitlab.Client
	*Options
}

func New(opts Options) (*Client, error) {
	var (
		host   string
		glopts []gitlab.ClientOptionFunc
	)

	if opts.Host != "" {
		host = opts.Host
	} else {
		host = gitlabBaseURL
	}

	glopts = append(glopts, gitlab.WithBaseURL(host))
	if opts.Insecure || opts.CAPath != "" {
		// Create a custom http.Client with a custom tls.Config
		httpClient, err := newHTTPClient(opts.Insecure, opts.CAPath)
		if err != nil {
			return nil, err
		}
		glopts = append(glopts, gitlab.WithHTTPClient(httpClient))
	}

	glclient, err := gitlab.NewClient(opts.Bearer, glopts...)
	if err != nil {
		return nil, err
	}

	client := &Client{
		Client:  glclient,
		Options: &opts,
	}

	return client, nil
}

// Name returns the name of the host URL for the selfhosted client
func (c *Client) Name() string {
	if len(c.Host) == 0 {
		return "gitlabapi"
	}

	return c.Host
}

// Tags will fetch the image tags from a given image URL. It must first query
// the tags that are available, then query the 2.1 and 2.2 API endpoints to
// gather the image digest and created time.
func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	var tags []api.ImageTag
	repoId, err := getRepoIDByPath(c.Client, repo, image)
	if err != nil {
		return nil, err
	}

	page := 1
	for {
		t, resp, err := c.Client.ContainerRegistry.ListRegistryRepositoryTags(repo, repoId,
			&gitlab.ListRegistryRepositoryTagsOptions{Page: page, PerPage: 20})
		if err != nil {
			return tags, err
		}

		for _, tag := range t {
			tags = append(tags, api.ImageTag{
				Tag:       tag.Name,
				SHA:       tag.Digest,
				Timestamp: *tag.CreatedAt,
			})
		}

		// Break the loop if we've reached the last page
		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		page++
	}
	return tags, nil
}

func newHTTPClient(insecure bool, CAPath string) (*http.Client, error) {
	// Load system CA Certs and/or create a new CertPool
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	if CAPath != "" {
		certs, err := os.ReadFile(CAPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to append %q to RootCAs: %v", CAPath, err)
		}
		if !rootCAs.AppendCertsFromPEM(certs) {
			return nil, fmt.Errorf("Failed to append CA certs from %q", CAPath)
		}
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure,
		RootCAs:            rootCAs,
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
		},
	}, nil
}

func getRepoIDByPath(client *gitlab.Client, projectID, repoPath string) (int, error) {
	page := 1
	for {
		repos, resp, err := client.ContainerRegistry.ListProjectRegistryRepositories(projectID,
			&gitlab.ListRegistryRepositoriesOptions{
				ListOptions: gitlab.ListOptions{Page: page, PerPage: 20},
			})
		if err != nil {
			return 0, fmt.Errorf("failed to list container repositories: %w", err)
		}

		for _, repo := range repos {
			if repo.Path == repoPath {
				return repo.ID, nil
			}
		}

		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		page++
	}

	return 0, fmt.Errorf("repository '%s' not found", repoPath)
}
