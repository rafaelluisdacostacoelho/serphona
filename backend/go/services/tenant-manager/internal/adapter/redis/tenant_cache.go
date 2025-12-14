// Package redis provides Redis cache functionality.
package redis

import (
	"context"
	"fmt"
	"time"

	"tenant-manager/internal/domain/tenant"

	"github.com/google/uuid"
)

// TenantCache implements tenant.Cache interface using Redis.
type TenantCache struct {
	cache *Cache
	ttl   time.Duration
}

// NewTenantCache creates a new TenantCache.
func NewTenantCache(cache *Cache, ttl time.Duration) *TenantCache {
	return &TenantCache{
		cache: cache,
		ttl:   ttl,
	}
}

// Get retrieves a tenant from cache.
func (tc *TenantCache) Get(ctx context.Context, key string) (*tenant.Tenant, error) {
	var t tenant.Tenant
	if err := tc.cache.Get(ctx, key, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// Set stores a tenant in cache.
func (tc *TenantCache) Set(ctx context.Context, key string, t *tenant.Tenant) error {
	return tc.cache.Set(ctx, key, t, tc.ttl)
}

// Delete removes a tenant from cache.
func (tc *TenantCache) Delete(ctx context.Context, key string) error {
	return tc.cache.Delete(ctx, key)
}

// GetSettings retrieves settings from cache.
func (tc *TenantCache) GetSettings(ctx context.Context, tenantID uuid.UUID) (*tenant.Settings, error) {
	key := fmt.Sprintf("tenant:settings:%s", tenantID)
	var settings tenant.Settings
	if err := tc.cache.Get(ctx, key, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// SetSettings stores settings in cache.
func (tc *TenantCache) SetSettings(ctx context.Context, tenantID uuid.UUID, settings *tenant.Settings) error {
	key := fmt.Sprintf("tenant:settings:%s", tenantID)
	return tc.cache.Set(ctx, key, settings, tc.ttl)
}

// Invalidate removes all cached data for a tenant.
func (tc *TenantCache) Invalidate(ctx context.Context, tenantID uuid.UUID) error {
	// Delete tenant cache
	if err := tc.Delete(ctx, fmt.Sprintf("tenant:%s", tenantID)); err != nil {
		return err
	}

	// Delete settings cache
	if err := tc.Delete(ctx, fmt.Sprintf("tenant:settings:%s", tenantID)); err != nil {
		return err
	}

	return nil
}
