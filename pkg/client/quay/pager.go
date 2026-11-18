package quay

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jetstack/version-checker/pkg/api"
)

// pager is used for implementing the paging mechanism for fetching image tags.
type pager struct {
	*Client

	repo, image string

	tags []api.ImageTag
	errs []error
	wg   sync.WaitGroup

	mu sync.Mutex
}

func (c *Client) newPager(repo, image string) *pager {
	return &pager{
		Client: c,
		repo:   repo,
		image:  image,
	}
}

func (p *pager) fetchTags(ctx context.Context) error {
	// Always wait for in-flight manifest fetches, including on error paths, so
	// no goroutine outlives the call or races with the caller reading p.tags.
	defer p.wg.Wait()

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

	// Individual manifest failures skip a tag; only fail when nothing could be fetched.
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.tags) == 0 && len(p.errs) > 0 {
		return errors.Join(p.errs...)
	}

	return nil
}

// fetchTagsPaged will return the image tags from a given page number, as well
// as if there are more pages.
func (p *pager) fetchTagsPaged(ctx context.Context, page int) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	url := fmt.Sprintf(tagURL, p.repo, p.image, page)
	var resp responseTag
	if err := p.makeRequest(ctx, url, &resp); err != nil {
		return false, err
	}

	sem := make(chan struct{}, 10) // limit concurrent fetches
	p.wg.Add(len(resp.Tags))

	for _, tag := range resp.Tags {
		go func(tag responseTagItem) {
			defer p.wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			imageTag, err := p.fetchImageManifest(ctx, p.repo, p.image, &tag)

			p.mu.Lock()
			defer p.mu.Unlock()

			if err != nil {
				p.errs = append(p.errs, err)
				return
			}

			p.tags = append(p.tags, *imageTag)
		}(tag)
	}

	return resp.HasAdditional, nil
}
