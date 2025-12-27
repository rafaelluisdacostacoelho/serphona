package embedding

import (
	"context"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	value     [][]float32
	expiresAt time.Time
}

// Cache defines minimal operations needed by CachingClient.
type Cache interface {
	Get(key string) ([][]float32, bool)
	Set(key string, value [][]float32, ttl time.Duration)
}

// InMemoryCache is a simple mutex-protected cache with TTL per entry.
type InMemoryCache struct {
	mu    sync.Mutex
	items map[string]cacheEntry
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{items: make(map[string]cacheEntry)}
}

func (c *InMemoryCache) Get(key string) ([][]float32, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		delete(c.items, key)
		return nil, false
	}
	return entry.value, true
}

func (c *InMemoryCache) Set(key string, value [][]float32, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp := time.Time{}
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	// store a copy to avoid caller mutating cached slice
	copyVal := make([][]float32, len(value))
	for i := range value {
		copyVal[i] = append([]float32(nil), value[i]...)
	}
	c.items[key] = cacheEntry{value: copyVal, expiresAt: exp}
}

// CachingClient wraps a Client and caches results per joined input key.
type CachingClient struct {
	Inner Client
	Cache Cache
	TTL   time.Duration
}

func (c CachingClient) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if c.Inner == nil {
		return nil, ErrNoInnerClient
	}
	if c.Cache == nil {
		return c.Inner.Embed(ctx, inputs)
	}
	key := cacheKey(inputs)
	if val, ok := c.Cache.Get(key); ok {
		return val, nil
	}
	res, err := c.Inner.Embed(ctx, inputs)
	if err != nil {
		return nil, err
	}
	c.Cache.Set(key, res, c.TTL)
	return res, nil
}

func cacheKey(inputs []string) string {
	// simple deterministic join; inputs ordering matters
	return strings.Join(inputs, "\u0001")
}
