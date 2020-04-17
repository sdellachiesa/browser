// Copyright 2020 Eurac Research. All rights reserved.
package browser

import (
	"context"
	"log"
	"sync"
	"time"
)

var (
	// Guarantee we implement browser.Metadata.
	_ Metadata = &InMemCache{}

	// CacheRefreshInterval is the interval in which the cache
	// will be refeshed.
	CacheRefreshInterval = 2 * time.Minute
)

// InMemCache represents an in memory cache currently for
// metadata only.
type InMemCache struct {
	metadata Metadata

	mu    sync.RWMutex
	cache map[Role]Stations
}

func NewInMemCache(m Metadata) *InMemCache {
	c := &InMemCache{
		metadata: m,
		cache:    make(map[Role]Stations),
	}

	c.loadCache()
	go c.refreshCache()

	return c
}

// loadCache initializes the cache for each Role due to the
// slow "SHOW TAG VALUES" queries on large datasets inside
// InfluxDB. Until measurements aren't present in SnipeIT
// they must be retrieved from InfluxDB.
func (c *InMemCache) loadCache() {
	cache := make(map[Role]Stations)

	for _, r := range Roles {
		log.Printf("loading cache for %s\n", r)

		ctx := context.WithValue(context.Background(), BrowserContextKey, &User{Role: r})
		s, err := c.metadata.Stations(ctx, &Message{})
		if err != nil {
			log.Printf("error: cache loading failed for %q: %v", r, err)
			continue
		}
		cache[r] = s
	}

	c.mu.Lock()
	c.cache = cache
	c.mu.Unlock()
}

func (c *InMemCache) refreshCache() {
	ticker := time.NewTicker(CacheRefreshInterval)

	go func() {
		for range ticker.C {
			c.loadCache()
		}
	}()
}

// Stations returns a cached instance of stations if available.
func (c *InMemCache) Stations(ctx context.Context, m *Message) (Stations, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	u := UserFromContext(ctx)
	s, ok := c.cache[u.Role]
	if !ok {
		log.Println("cache missed")
		return c.metadata.Stations(ctx, m)
	}

	return s, nil
}
