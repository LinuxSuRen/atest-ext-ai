package ai

import (
	"sync"
	"time"

	"atest-ext-ai-core/internal/logger"
)

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

// IsExpired checks if the cache item has expired
func (c *CacheItem) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// Cache represents an in-memory cache with TTL support
type Cache struct {
	mu       sync.RWMutex
	items    map[string]*CacheItem
	maxSize  int
	ttl      time.Duration
	stopChan chan struct{}
}

// NewCache creates a new cache instance
func NewCache(maxSize int, ttl time.Duration) *Cache {
	cache := &Cache{
		items:    make(map[string]*CacheItem),
		maxSize:  maxSize,
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists || item.IsExpired() {
		return nil
	}

	return item.Value
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict items to make space
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = &CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheItem)
}

// Size returns the current number of items in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	expiredCount := 0
	for _, item := range c.items {
		if item.IsExpired() {
			expiredCount++
		}
	}

	return map[string]interface{}{
		"total_items":   len(c.items),
		"expired_items": expiredCount,
		"max_size":      c.maxSize,
		"ttl_seconds":   c.ttl.Seconds(),
	}
}

// Close stops the cache cleanup goroutine
func (c *Cache) Close() {
	close(c.stopChan)
}

// evictOldest removes the oldest item from the cache
func (c *Cache) evictOldest() {
	if len(c.items) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, item := range c.items {
		if first || item.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.ExpiresAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// cleanup periodically removes expired items
func (c *Cache) cleanup() {
	logger.Debug("Starting cache cleanup routine")
	ticker := time.NewTicker(c.ttl / 2) // Cleanup every half TTL
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stopChan:
			return
		}
	}
}

// removeExpired removes all expired items from the cache
func (c *Cache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiredCount := 0
	for key, item := range c.items {
		if item.IsExpired() {
			delete(c.items, key)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		logger.Debugf("Cache cleanup: removed %d expired items", expiredCount)
	}
}
