package search

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/cache"
	"github.com/jetstack/version-checker/pkg/version"
)

// Searcher is the interface for Search to facilitate testing.
type Searcher interface {
	LatestImage(context.Context, string, *api.Options) (*api.ImageTag, error)
	ResolveSHAToTag(ctx context.Context, imageURL string, imageSHA string) (string, error)
}

// Search is the implementation for the searching and caching of image URLs.
type Search struct {
	log *logrus.Entry

	versionGetter *version.Version
	searchCache   *cache.Cache
}

// New creates a new Search for querying searches over image tags.
func New(log *logrus.Entry, cacheTimeout time.Duration, versionGetter *version.Version) *Search {
	s := &Search{
		log:           log.WithField("module", "search"),
		versionGetter: versionGetter,
	}

	s.searchCache = cache.New(s.log, cacheTimeout, s)

	return s
}

func (s *Search) Fetch(ctx context.Context, imageURL string, opts *api.Options) (interface{}, error) {
	latestImage, err := s.versionGetter.LatestTagFromImage(ctx, imageURL, opts)
	if err != nil {
		return nil, err
	}

	return latestImage, nil
}

// LatestImage will get the latestImage image given an image URL and
// options. If not found in the cache, or is too old, then will do a fresh
// lookup and commit to the cache.
func (s *Search) LatestImage(ctx context.Context, imageURL string, opts *api.Options) (*api.ImageTag, error) {
	hashIndex, err := calculateHashIndex(imageURL, opts)
	if err != nil {
		return nil, err
	}

	lastestImage, err := s.searchCache.Get(ctx, hashIndex, imageURL, opts)
	if err != nil {
		return nil, err
	}

	return lastestImage.(*api.ImageTag), nil
}

func (s *Search) ResolveSHAToTag(ctx context.Context, imageURL string, imageSHA string) (string, error) {

	tag, err := s.versionGetter.ResolveSHAToTag(ctx, imageURL, imageSHA)
	if err != nil {
		return "", fmt.Errorf("failed to resolve sha to tag: %w", err)
	}

	return tag, err
}

// calculateHashIndex returns a hash index given an imageURL and options.
func calculateHashIndex(imageURL string, opts *api.Options) (string, error) {
	optsJSON, err := json.Marshal(opts)
	if err != nil {
		return "", fmt.Errorf("failed to marshal options: %s", err)
	}

	hash := fnv.New32()
	if _, err := hash.Write(append(optsJSON, []byte(imageURL)...)); err != nil {
		return "", fmt.Errorf("failed to calculate search hash: %s", err)
	}

	return fmt.Sprintf("%d", hash.Sum32()), nil
}
