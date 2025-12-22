package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// TenantToolRepository defines the interface for tenant tool data operations
type TenantToolRepository interface {
	// Create creates a new tenant tool configuration
	Create(ctx context.Context, tenantTool *entity.TenantTool) error

	// FindByID finds a tenant tool by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entity.TenantTool, error)

	// FindByTenantAndTool finds a tenant tool by tenant ID and tool ID
	FindByTenantAndTool(ctx context.Context, tenantID, toolID uuid.UUID) (*entity.TenantTool, error)

	// FindByTenant finds all tools for a tenant
	FindByTenant(ctx context.Context, tenantID uuid.UUID, onlyEnabled bool) ([]*entity.TenantTool, error)

	// Update updates an existing tenant tool configuration
	Update(ctx context.Context, tenantTool *entity.TenantTool) error

	// Delete deletes a tenant tool configuration
	Delete(ctx context.Context, id uuid.UUID) error

	// Enable enables a tool for a tenant
	Enable(ctx context.Context, tenantID, toolID uuid.UUID) error

	// Disable disables a tool for a tenant
	Disable(ctx context.Context, tenantID, toolID uuid.UUID) error
}
