package handler

import (
	"sync"
	"time"
)

// DedupCache is a simple in-memory deduplication cache
type DedupCache struct {
	cache map[string]time.Time
	mu    sync.Mutex
}

var globalCache = &DedupCache{cache: make(map[string]time.Time)} // Global singleton

func NewDedupCache() *DedupCache {
	return globalCache // Always returns global instance
}

func (c *DedupCache) IsDuplicate(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.cache[key]; exists {
		return true
	}

	c.cache[key] = time.Now()
	return false
}

// CleanExpired removes old entries but NEVER called anywhere
// Memory leak: cache grows unbounded
func (c *DedupCache) CleanExpired(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, ts := range c.cache {
		if now.Sub(ts) > ttl {
			delete(c.cache, key)
		}
	}
}
