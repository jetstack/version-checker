package quay

import (
	"context"
	"fmt"
	"sync"

	"github.com/jetstack/version-checker/pkg/api"
)

// pager is used for implementing the paging mechanism for fetching image tags.
type pager struct {
	*Client

	repo, image string

	mu sync.Mutex
	wg sync.WaitGroup

	tags []api.ImageTag
	errs []error
}

func (c *Client) newPager(repo, image string) *pager {
	return &pager{
		Client: c,
		repo:   repo,
		image:  image,
	}
}

func (p *pager) fetchTags(ctx context.Context) error {
	var (
		page          = 1
		hasAdditional = true
		err           error
	)

	// Need to set a fair page limit to handle some registries
	for hasAdditional && page < 60 {
		// Fetch all image tags in this page
		hasAdditional, err = p.fetchTagsPaged(ctx, page)
		if err != nil {
			return err
		}

		page++
	}

	p.wg.Wait()

	return nil
}

// fetchTagsPaged will return the image tags from a given page number, as well
// as if there are more pages.
func (p *pager) fetchTagsPaged(ctx context.Context, page int) (bool, error) {
	url := fmt.Sprintf(tagURL, p.repo, p.image, page)
	var resp responseTag
	if err := p.makeRequest(ctx, url, &resp); err != nil {
		return false, err
	}

	p.wg.Add(len(resp.Tags))

	// Concurrently fetch all images from a given tag
	for i := range resp.Tags {
		go func(i int) {
			defer p.wg.Done()

			imageTags, err := p.fetchImageManifest(ctx, p.repo, p.image, &resp.Tags[i])

			p.mu.Lock()
			defer p.mu.Unlock()

			if err != nil {
				p.errs = append(p.errs, err)
				return
			}

			p.tags = append(p.tags, imageTags...)
		}(i)
	}

	return resp.HasAdditional, nil
}
