package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/version"
)

// imageCacheItem is a single node item for the cache of a lastest image search.
type imageCacheItem struct {
	timestamp   time.Time
	latestImage *api.ImageTag
}

// StartGabageCollector will start the garbage collector for image search cache.
func (c *Controller) StartGabageCollector(refreshRate time.Duration) {
	log := c.log.WithField("search_cache", "garbage_collector")
	log.Infof("starting search cache garbage collector")

	ticker := time.NewTicker(refreshRate)
	for {
		<-ticker.C

		c.cacheMu.Lock()
		now := time.Now()
		for hashIndex, cacheItem := range c.imageCache {

			// Check is cache item is fresh
			if cacheItem.timestamp.Add(c.cacheTimeout).Before(now) {

				log.Debugf("removing stale search from cache: %q",
					hashIndex)

				delete(c.imageCache, hashIndex)
			}
		}
		c.cacheMu.Unlock()
	}
}

// getLatestImage will get the latestImage image given an image URL and
// options. If not found in the cache, or is too old, then will do a fresh
// lookup and commit to the cache.
func (c *Controller) getLatestImage(ctx context.Context, log *logrus.Entry,
	imageURL string, opts *api.Options) (*api.ImageTag, error) {

	log = c.log.WithField("search_cache", "getter")

	hashIndex, err := version.CalculateHashIndex(imageURL, opts)
	if err != nil {
		return nil, err
	}

	c.cacheMu.RLock()
	cacheItem, ok := c.imageCache[hashIndex]
	c.cacheMu.RUnlock()

	// Test if exists in the cache or is too old
	if !ok || cacheItem.timestamp.Add(c.cacheTimeout).Before(time.Now()) {
		latestImage, err := c.versionGetter.LatestTagFromImage(ctx, opts, imageURL)
		if err != nil {
			return nil, fmt.Errorf("%q: %s", imageURL, err)
		}

		// Commit to the cache
		log.Debugf("committing search: %q", hashIndex)
		c.cacheMu.Lock()
		c.imageCache[hashIndex] = imageCacheItem{time.Now(), latestImage}
		c.cacheMu.Unlock()

		return latestImage, nil
	}

	log.Debugf("found search: %q", hashIndex)

	return cacheItem.latestImage, nil
}
