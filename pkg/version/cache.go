package version

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/jetstack/version-checker/pkg/api"
)

// imageCache is used to store a cache of all remote images for a given
// imageURL. This cache is periodically garbage collected.
type imageCacheItem struct {
	timestamp time.Time
	tags      []api.ImageTag
}

// CalculateHashIndex returns a hash index given an imageURL and options.
func CalculateHashIndex(imageURL string, opts *api.Options) (string, error) {
	opsJson, err := json.Marshal(opts)
	if err != nil {
		return "", fmt.Errorf("failed to marshal options: %s", err)
	}

	hash := fnv.New32()
	if _, err := hash.Write(append(opsJson, []byte(imageURL)...)); err != nil {
		return "", fmt.Errorf("failed to calculate search hash: %s", err)
	}

	return fmt.Sprintf("%d", hash.Sum32()), nil
}

// tryImageCache return an imageCacheItem item and true if their is a cache hit
// on the given image URL. Deletes the item, and returns nil and false if there
// is a miss.
func (v *VersionGetter) tryImageCache(imageURL string) ([]api.ImageTag, bool) {
	v.cacheMu.RLock()
	defer v.cacheMu.RUnlock()

	if imageCacheItem, ok := v.imageCache[imageURL]; ok &&
		!imageCacheItem.timestamp.Add(v.cacheTimeout).Before(time.Now()) {

		v.log.WithField("cache", "getter").Debugf(
			"found image tags: %q", imageURL)

		return imageCacheItem.tags, true
	}

	return nil, false
}

// garbageCollect is a blocking func that will run the garbage collector
// against the images tag cache.
func (v *VersionGetter) garbageCollect(refreshRate time.Duration) {
	log := v.log.WithField("cache", "garbage_collector")
	log.Infof("starting image tags cache garbage collector")
	ticker := time.NewTicker(refreshRate)

	for {
		<-ticker.C

		v.cacheMu.Lock()

		now := time.Now()
		for imageURL, cacheItem := range v.imageCache {
			if cacheItem.timestamp.Add(v.cacheTimeout).Before(now) {

				log.Debugf("removing stale image tags: %q", imageURL)
				delete(v.imageCache, imageURL)
			}
		}

		v.cacheMu.Unlock()
	}
}
