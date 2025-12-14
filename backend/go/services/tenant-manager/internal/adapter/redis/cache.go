// Package redis provides Redis cache functionality.
package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps redis.Client and provides cache operations.
type Cache struct {
	client *redis.Client
}

// NewCache creates a new Cache instance.
func NewCache(client *redis.Client) *Cache {
	return &Cache{
		client: client,
	}
}

// Get retrieves a value from cache and unmarshals into dest.
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// Set stores a value in cache with TTL.
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

// Delete removes a key from cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Close closes the Redis connection.
func (c *Cache) Close() error {
	return c.client.Close()
}
