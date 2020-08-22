package cache

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
)

// Cache is the cache store for image manifests
type Cache struct {
	log *logrus.Entry

	// cacheTimeout is the amount of time a imageCache item is considered fresh
	// for.
	cacheTimeout time.Duration
	mu           sync.RWMutex
	imageCache   map[string]imageCacheItem
}

// imageCache is used to store a cache of all remote images for a given
// imageURL. This cache is periodically garbage collected.
type imageCacheItem struct {
	timestamp time.Time
	tags      []api.ImageTag
}

// New returns a new image cache.
func New(log *logrus.Entry, cacheTimeout time.Duration) *Cache {
	return &Cache{
		log:          log.WithField("cache", "getter"),
		cacheTimeout: cacheTimeout,
		imageCache:   make(map[string]imageCacheItem),
	}
}

// ImageTags returns the given image tags for a given image URL.  Deletes the
// item, and returns nil and false if there is a miss.
func (c *Cache) ImageTags(imageURL string) ([]api.ImageTag, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if imageCacheItem, ok := c.imageCache[imageURL]; ok &&
		!imageCacheItem.timestamp.Add(c.cacheTimeout).Before(time.Now()) {

		c.log.Debugf("found image tags: %q", imageURL)

		return imageCacheItem.tags, true
	}

	return nil, false
}

// CommitTags will commit a set of image tags for the given image URL to the cache.
func (c *Cache) CommitTags(imageURL string, tags []api.ImageTag) {
	// Add tags to cache
	c.mu.Lock()
	defer c.mu.Unlock()
	c.imageCache[imageURL] = imageCacheItem{
		timestamp: time.Now(),
		tags:      tags,
	}

}

// StartGarbageCollector is a blocking func that will run the garbage collector
// against the images tag cache.
func (c *Cache) StartGarbageCollector(refreshRate time.Duration) {
	log := c.log.WithField("cache", "garbage_collector")
	log.Infof("starting image tags cache garbage collector")
	ticker := time.NewTicker(refreshRate)

	for {
		<-ticker.C

		c.mu.Lock()

		now := time.Now()
		for imageURL, cacheItem := range c.imageCache {
			if cacheItem.timestamp.Add(c.cacheTimeout).Before(now) {

				log.Debugf("removing stale image tags: %q", imageURL)
				delete(c.imageCache, imageURL)
			}
		}

		c.mu.Unlock()
	}
}
