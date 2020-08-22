package cache

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Cache is a generic cache store.
type Cache struct {
	log *logrus.Entry

	mu      sync.RWMutex
	timeout time.Duration
	handler Handler

	store map[string]*cacheItem
}

// cacheItem is a single item for the cache stored. This cache item is
// periodically garbage collected.
type cacheItem struct {
	mu        sync.Mutex
	timestamp time.Time
	i         interface{}
}

// Handler is an interface for implementations of the cache fetch
type Handler interface {
	// Fetch should fetch an item by the given index and options
	Fetch(ctx context.Context, index string, opts interface{}) (interface{}, error)
}

// New returns a new generic Cache
func New(log *logrus.Entry, timeout time.Duration, handler Handler) *Cache {
	return &Cache{
		log:     log.WithField("cache", "handler"),
		handler: handler,
		timeout: timeout,
		store:   make(map[string]*cacheItem),
	}
}

// Get returns the cache item from the store given the index. Will populate
// the cache if the index does not currently exist.
func (c *Cache) Get(ctx context.Context, index string, fetchIndex string, opts interface{}) (interface{}, error) {
	c.mu.RLock()
	item, ok := c.store[index]
	c.mu.RUnlock()

	// If the item doesn't yet exist, create a new zero item.
	if !ok {
		item = new(cacheItem)
		c.mu.Lock()
		c.store[index] = item
		c.mu.Unlock()
	}

	item.mu.Lock()
	defer item.mu.Unlock()

	// Test if exists in the cache or is too old
	if item.timestamp.Add(c.timeout).Before(time.Now()) {
		// Fetch a new item to commit
		i, err := c.handler.Fetch(ctx, fetchIndex, opts)
		if err != nil {
			return nil, err
		}

		// Commit to the cache
		c.log.Debugf("committing item: %q", index)
		item.timestamp = time.Now()
		item.i = i

		return i, nil
	}

	c.log.Debugf("found: %q", index)

	return item.i, nil
}

// StartGarbageCollector is a blocking func that will run the garbage collector
// against the cache.
func (c *Cache) StartGarbageCollector(refreshRate time.Duration) {
	log := c.log.WithField("cache", "garbage_collector")
	log.Infof("starting cache garbage collector")
	ticker := time.NewTicker(refreshRate)

	for {
		<-ticker.C

		c.mu.Lock()

		now := time.Now()
		for index, item := range c.store {
			if item.timestamp.Add(c.timeout).Before(now) {

				log.Debugf("removing stale cache item: %q", index)
				delete(c.store, index)
			}
		}

		c.mu.Unlock()
	}
}
