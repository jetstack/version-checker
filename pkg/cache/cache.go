package cache

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jetstack/version-checker/pkg/api"
	"github.com/patrickmn/go-cache"
)

// Cache is a generic cache store - that supports a handler
type Cache struct {
	log     *logrus.Entry
	handler Handler

	store *cache.Cache
}

// Handler is an interface for implementations of the cache fetch.
type Handler interface {
	// Fetch should fetch an item by the given index and options
	Fetch(ctx context.Context, index string, opts *api.Options) (interface{}, error)
}

// New returns a new generic Cache.
func New(log *logrus.Entry, timeout time.Duration, handler Handler) *Cache {
	c := &Cache{
		log:     log.WithField("cache", "handler"),
		handler: handler,
		store:   cache.New(timeout, timeout*2),
	}
	// Set our Cleanup hook
	c.store.OnEvicted(c.cleanup)
	return c
}

func (c *Cache) cleanup(key string, obj interface{}) {
	c.log.Debugf("removing item from cache: %q", key)
}

func (c *Cache) Shutdown() {
	c.store.Flush()
}

// Get returns the cache item from the store given the index. Will populate
// the cache if the index does not currently exist.
func (c *Cache) Get(ctx context.Context, index string, fetchIndex string, opts *api.Options) (item interface{}, err error) {
	item, found := c.store.Get(index)

	// If the item doesn't yet exist, Lets look it up
	if !found {
		// Fetch a new item to commit
		item, err = c.handler.Fetch(ctx, fetchIndex, opts)
		if err != nil {
			return nil, err
		}

		// Commit to the cache
		c.log.Debugf("committing item: %q", index)
		c.store.Set(index, item, cache.DefaultExpiration)
	}
	c.log.Debugf("found: %q", index)

	return item, err
}

func (c *Cache) Update(index string, item interface{}) {
	c.store.SetDefault(index, item)
}
func (c *Cache) Delete(index string) {
	c.store.Delete(index)
}
