package cache

import (
	"context"
	"sync"
	"time"
)

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

func NewMemoryCache() Cache {
	mc := &MemoryCache{
		items: make(map[string]cacheItem),
	}
	return mc
}

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}
	return item.value, true
}

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	return nil
}
