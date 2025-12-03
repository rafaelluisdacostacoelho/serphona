// Package redis provides Redis client implementations.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"tenant-manager/internal/domain/tenant"
)

// Cache implements tenant.Cache using Redis.
type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewCache creates a new Redis cache.
func NewCache(client *redis.Client, ttl time.Duration) *Cache {
	return &Cache{
		client: client,
		ttl:    ttl,
	}
}

// Get retrieves a tenant from cache.
func (c *Cache) Get(ctx context.Context, key string) (*tenant.Tenant, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found in cache")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	var t tenant.Tenant
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tenant: %w", err)
	}

	return &t, nil
}

// Set stores a tenant in cache.
func (c *Cache) Set(ctx context.Context, key string, t *tenant.Tenant) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("failed to marshal tenant: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set in cache: %w", err)
	}

	return nil
}

// Delete removes a tenant from cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}
	return nil
}

// GetSettings retrieves settings from cache.
func (c *Cache) GetSettings(ctx context.Context, tenantID uuid.UUID) (*tenant.Settings, error) {
	key := fmt.Sprintf("tenant:%s:settings", tenantID)
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("settings not found in cache")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settings from cache: %w", err)
	}

	var settings tenant.Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return &settings, nil
}

// SetSettings stores settings in cache.
func (c *Cache) SetSettings(ctx context.Context, tenantID uuid.UUID, settings *tenant.Settings) error {
	key := fmt.Sprintf("tenant:%s:settings", tenantID)
	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set settings in cache: %w", err)
	}

	return nil
}

// Invalidate removes all cached data for a tenant.
func (c *Cache) Invalidate(ctx context.Context, tenantID uuid.UUID) error {
	keys := []string{
		fmt.Sprintf("tenant:%s", tenantID),
		fmt.Sprintf("tenant:%s:settings", tenantID),
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	return nil
}
