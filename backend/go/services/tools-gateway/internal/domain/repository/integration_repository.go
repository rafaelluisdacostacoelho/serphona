// Package repository contains repository interfaces.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// IntegrationRepository defines the interface for integration data access.
type IntegrationRepository interface {
	// Create creates a new integration.
	Create(ctx context.Context, integration *entity.Integration) error

	// FindByID finds an integration by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Integration, error)

	// FindByTenantID finds all integrations for a tenant.
	FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*entity.Integration, error)

	// FindByTenantAndProvider finds an integration by tenant and provider.
	FindByTenantAndProvider(ctx context.Context, tenantID uuid.UUID, provider string) (*entity.Integration, error)

	// FindActive finds all active integrations for a tenant.
	FindActive(ctx context.Context, tenantID uuid.UUID) ([]*entity.Integration, error)

	// Update updates an integration.
	Update(ctx context.Context, integration *entity.Integration) error

	// Delete deletes an integration.
	Delete(ctx context.Context, id uuid.UUID) error

	// List lists integrations with filters.
	List(ctx context.Context, filters IntegrationFilters) ([]*entity.Integration, int64, error)
}

// IntegrationFilters holds filtering options for listing integrations.
type IntegrationFilters struct {
	TenantID *uuid.UUID
	Provider string
	Type     entity.IntegrationType
	IsActive *bool
	Limit    int
	Offset   int
}
