package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"gorm.io/gorm"
)

// tenantToolRepositoryImpl implements the TenantToolRepository interface
type tenantToolRepositoryImpl struct {
	db *gorm.DB
}

// NewTenantToolRepository creates a new instance of TenantToolRepository
func NewTenantToolRepository(db *gorm.DB) repository.TenantToolRepository {
	return &tenantToolRepositoryImpl{db: db}
}

// Create creates a new tenant tool configuration
func (r *tenantToolRepositoryImpl) Create(ctx context.Context, tenantTool *entity.TenantTool) error {
	return r.db.WithContext(ctx).Create(tenantTool).Error
}

// FindByID finds a tenant tool by ID
func (r *tenantToolRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.TenantTool, error) {
	var tenantTool entity.TenantTool
	err := r.db.WithContext(ctx).
		Preload("Tool").
		Where("id = ?", id).
		First(&tenantTool).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tenant tool not found: %w", err)
		}
		return nil, err
	}
	return &tenantTool, nil
}

// FindByTenantAndTool finds a tenant tool by tenant ID and tool ID
func (r *tenantToolRepositoryImpl) FindByTenantAndTool(ctx context.Context, tenantID, toolID uuid.UUID) (*entity.TenantTool, error) {
	var tenantTool entity.TenantTool
	err := r.db.WithContext(ctx).
		Preload("Tool").
		Where("tenant_id = ? AND tool_id = ?", tenantID, toolID).
		First(&tenantTool).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tenant tool not found: %w", err)
		}
		return nil, err
	}
	return &tenantTool, nil
}

// FindByTenant finds all tools for a tenant
func (r *tenantToolRepositoryImpl) FindByTenant(ctx context.Context, tenantID uuid.UUID, onlyEnabled bool) ([]*entity.TenantTool, error) {
	var tenantTools []*entity.TenantTool
	query := r.db.WithContext(ctx).
		Preload("Tool").
		Where("tenant_id = ?", tenantID)

	if onlyEnabled {
		query = query.Where("is_enabled = ?", true)
	}

	err := query.Order("created_at DESC").Find(&tenantTools).Error
	if err != nil {
		return nil, err
	}

	return tenantTools, nil
}

// Update updates an existing tenant tool configuration
func (r *tenantToolRepositoryImpl) Update(ctx context.Context, tenantTool *entity.TenantTool) error {
	return r.db.WithContext(ctx).Save(tenantTool).Error
}

// Delete deletes a tenant tool configuration
func (r *tenantToolRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.TenantTool{}, "id = ?", id).Error
}

// Enable enables a tool for a tenant
func (r *tenantToolRepositoryImpl) Enable(ctx context.Context, tenantID, toolID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&entity.TenantTool{}).
		Where("tenant_id = ? AND tool_id = ?", tenantID, toolID).
		Update("is_enabled", true).Error
}

// Disable disables a tool for a tenant
func (r *tenantToolRepositoryImpl) Disable(ctx context.Context, tenantID, toolID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&entity.TenantTool{}).
		Where("tenant_id = ? AND tool_id = ?", tenantID, toolID).
		Update("is_enabled", false).Error
}
