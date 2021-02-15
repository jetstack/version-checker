package selfhosted

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/generic"
)

type Options struct {
	Host     string
	Username string
	Password string
	Bearer   string
}

type Client struct {
	genericClient *generic.Client
	log           *logrus.Entry
}

func New(ctx context.Context, log *logrus.Entry, opts *Options) (*Client, error) {
	log = log.WithField("client", opts.Host)
	genericOptions := &generic.Options{
		Host:     opts.Host,
		Username: opts.Username,
		Password: opts.Password,
		Bearer:   opts.Bearer,
	}

	genericClient, err := generic.New(ctx, nil, log, genericOptions)
	if err != nil {
		return nil, err
	}

	client := &Client{
		genericClient: genericClient,
		log:           log,
	}

	return client, nil
}

// Name returns the name of the host URL for the selfhosted client
func (c *Client) Name() string {
	return c.Name()
}

// Tags will fetch the image tags from a given image URL. It must first query
// the tags that are available, then query the 2.1 and 2.2 API endpoints to
// gather the image digest and created time.
func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	return c.Tags(ctx, host, repo, image)
}
