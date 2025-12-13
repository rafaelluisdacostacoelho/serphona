// Package apikey contains the API key domain model and business logic.
package apikey

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for API key persistence.
type Repository interface {
	// Save saves a new API key
	Save(ctx context.Context, key *APIKey) error

	// Update updates an existing API key
	Update(ctx context.Context, key *APIKey) error

	// FindByID finds an API key by ID
	FindByID(ctx context.Context, id uuid.UUID) (*APIKey, error)

	// FindByTenantID finds all API keys for a tenant
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*APIKey, error)

	// FindByKeyHash finds an API key by its hash
	FindByKeyHash(ctx context.Context, keyHash string) (*APIKey, error)

	// FindByKeyPrefix finds an API key by its prefix
	FindByKeyPrefix(ctx context.Context, prefix string) (*APIKey, error)

	// FindActiveByTenantID finds all active API keys for a tenant
	FindActiveByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*APIKey, error)

	// Delete deletes an API key (hard delete)
	Delete(ctx context.Context, id uuid.UUID) error

	// ExistsByName checks if an API key with the given name exists for a tenant
	ExistsByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error)

	// CountByTenantID counts API keys for a tenant
	CountByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// CountActiveByTenantID counts active API keys for a tenant
	CountActiveByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error)

	// ListExpired lists all expired API keys
	ListExpired(ctx context.Context, limit int) ([]*APIKey, error)

	// RevokeAll revokes all API keys for a tenant
	RevokeAll(ctx context.Context, tenantID uuid.UUID, revokedBy uuid.UUID) error
}
