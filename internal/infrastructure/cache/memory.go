package cache

import (
	"sync"
	"time"
)

// CacheItem holds the actual data and its expiration time.
type CacheItem struct {
	Data      []byte // Storing as []byte as data is already JSON marshaled
	ExpiresAt time.Time
}

// MemoryCache is a simple in-memory cache with TTL support.
type MemoryCache struct {
	store sync.Map
	// No global TTL, each item will have its own expiration
	// A cleanup goroutine could be added for proactive eviction,
	// but for now, eviction is passive (on Get).
}

// NewMemoryCache creates a new MemoryCache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{}
}

// Set adds an item to the cache with a specific TTL.
// Key is string, value is []byte, ttl is time.Duration.
func (mc *MemoryCache) Set(key string, value []byte, ttl time.Duration) {
	expiresAt := time.Now().Add(ttl)
	mc.store.Store(key, CacheItem{
		Data:      value,
		ExpiresAt: expiresAt,
	})
}

// Get retrieves an item from the cache.
// Returns the data and true if found and not expired, otherwise nil and false.
func (mc *MemoryCache) Get(key string) ([]byte, bool) {
	item, ok := mc.store.Load(key)
	if !ok {
		return nil, false // Not found
	}

	cacheItem, ok := item.(CacheItem)
	if !ok {
		// Should not happen if only CacheItem is stored
		mc.store.Delete(key) // Clean up malformed entry
		return nil, false
	}

	if time.Now().After(cacheItem.ExpiresAt) {
		// Item has expired
		mc.store.Delete(key) // Passive eviction
		return nil, false
	}

	return cacheItem.Data, true
}

// Delete removes an item from the cache.
func (mc *MemoryCache) Delete(key string) {
	mc.store.Delete(key)
}

// TODO: Consider adding a periodic cleanup goroutine in NewMemoryCache
// if proactive eviction of many expired items is desired to manage memory,
// rather than only passive eviction on Get or Delete.
// For example:
// go func() {
// 	for range time.Tick(cleanupInterval) {
// 		mc.store.Range(func(key, value interface{}) bool {
// 			cacheItem, ok := value.(CacheItem)
// 			if ok && time.Now().After(cacheItem.ExpiresAt) {
// 				mc.store.Delete(key)
// 			}
// 			return true // continue iteration
// 		})
// 	}
// }()
// This would require `cleanupInterval` to be defined and the goroutine to be managed (e.g., stopped on shutdown).
// For simplicity in this step, proactive cleanup is omitted.
