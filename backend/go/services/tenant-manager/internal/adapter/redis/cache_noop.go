// Package redis provides Redis cache functionality.
package redis

import (
	"context"

	"github.com/google/uuid"

	"tenant-manager/internal/domain/tenant"
)

// NoopCache implements tenant.Cache without backing storage.
type NoopCache struct{}

func (NoopCache) Get(ctx context.Context, key string) (*tenant.Tenant, error) { return nil, nil }
func (NoopCache) Set(ctx context.Context, key string, t *tenant.Tenant) error { return nil }
func (NoopCache) Delete(ctx context.Context, key string) error                { return nil }
func (NoopCache) GetSettings(ctx context.Context, tenantID uuid.UUID) (*tenant.Settings, error) {
	return nil, nil
}
func (NoopCache) SetSettings(ctx context.Context, tenantID uuid.UUID, settings *tenant.Settings) error {
	return nil
}
func (NoopCache) Invalidate(ctx context.Context, tenantID uuid.UUID) error { return nil }
