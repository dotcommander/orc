package core

import (
	"context"
	"sync"
	"time"
)

// CacheEntry represents a cached item with TTL
type CacheEntry[T any] struct {
	Value     T
	ExpiresAt time.Time
}

// IsExpired checks if the cache entry has expired
func (e *CacheEntry[T]) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// MemoryCache provides high-performance in-memory caching
type MemoryCache[K comparable, V any] struct {
	data      map[K]*CacheEntry[V]
	mutex     sync.RWMutex
	defaultTTL time.Duration
	maxSize   int
	hits      uint64
	misses    uint64
}

// NewMemoryCache creates an optimized memory cache
func NewMemoryCache[K comparable, V any](defaultTTL time.Duration, maxSize int) *MemoryCache[K, V] {
	if maxSize <= 0 {
		maxSize = 1000 // Reasonable default
	}
	
	cache := &MemoryCache[K, V]{
		data:       make(map[K]*CacheEntry[V]),
		defaultTTL: defaultTTL,
		maxSize:    maxSize,
	}
	
	// Start cleanup goroutine
	go cache.cleanupLoop()
	
	return cache
}

// Get retrieves a value from cache
func (c *MemoryCache[K, V]) Get(key K) (V, bool) {
	c.mutex.RLock()
	entry, exists := c.data[key]
	c.mutex.RUnlock()
	
	if !exists || entry.IsExpired() {
		c.misses++
		var zero V
		return zero, false
	}
	
	c.hits++
	return entry.Value, true
}

// Set stores a value in cache with default TTL
func (c *MemoryCache[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a value with custom TTL
func (c *MemoryCache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Evict if at capacity
	if len(c.data) >= c.maxSize {
		c.evictOldest()
	}
	
	c.data[key] = &CacheEntry[V]{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a key from cache
func (c *MemoryCache[K, V]) Delete(key K) {
	c.mutex.Lock()
	delete(c.data, key)
	c.mutex.Unlock()
}

// Clear removes all entries
func (c *MemoryCache[K, V]) Clear() {
	c.mutex.Lock()
	c.data = make(map[K]*CacheEntry[V])
	c.mutex.Unlock()
}

// Stats returns cache statistics
func (c *MemoryCache[K, V]) Stats() (hits, misses uint64, size int) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.hits, c.misses, len(c.data)
}

// evictOldest removes the oldest entry (simple LRU approximation)
func (c *MemoryCache[K, V]) evictOldest() {
	var oldestKey K
	var oldestTime time.Time = time.Now()
	
	for key, entry := range c.data {
		if entry.ExpiresAt.Before(oldestTime) {
			oldestTime = entry.ExpiresAt
			oldestKey = key
		}
	}
	
	delete(c.data, oldestKey)
}

// cleanupLoop periodically removes expired entries
func (c *MemoryCache[K, V]) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries
func (c *MemoryCache[K, V]) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	for key, entry := range c.data {
		if entry.IsExpired() {
			delete(c.data, key)
		}
	}
}

// PhaseResultCache provides optimized caching for phase execution results
type PhaseResultCache struct {
	cache *MemoryCache[string, PhaseOutput]
}

// NewPhaseResultCache creates a cache optimized for phase results
func NewPhaseResultCache(ttl time.Duration, maxEntries int) *PhaseResultCache {
	return &PhaseResultCache{
		cache: NewMemoryCache[string, PhaseOutput](ttl, maxEntries),
	}
}

// Get retrieves a cached phase result
func (p *PhaseResultCache) Get(ctx context.Context, phase string, input PhaseInput) (PhaseOutput, bool) {
	key := p.generateKey(phase, input)
	return p.cache.Get(key)
}

// Set caches a phase result
func (p *PhaseResultCache) Set(ctx context.Context, phase string, input PhaseInput, output PhaseOutput) {
	key := p.generateKey(phase, input)
	p.cache.Set(key, output)
}

// generateKey creates a cache key from phase and input
func (p *PhaseResultCache) generateKey(phase string, input PhaseInput) string {
	// Simple key generation - could be enhanced with proper hashing
	return phase + ":" + input.Request
}

// Stats returns cache performance statistics
func (p *PhaseResultCache) Stats() (hits, misses uint64, size int) {
	return p.cache.Stats()
}