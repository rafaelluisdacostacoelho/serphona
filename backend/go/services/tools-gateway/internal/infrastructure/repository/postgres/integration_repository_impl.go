// Package postgres contains PostgreSQL repository implementations.
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"gorm.io/gorm"
)

// integrationRepositoryImpl implements repository.IntegrationRepository.
type integrationRepositoryImpl struct {
	db *gorm.DB
}

// NewIntegrationRepository creates a new IntegrationRepository.
func NewIntegrationRepository(db *gorm.DB) repository.IntegrationRepository {
	return &integrationRepositoryImpl{db: db}
}

// Create creates a new integration.
func (r *integrationRepositoryImpl) Create(ctx context.Context, integration *entity.Integration) error {
	if err := r.db.WithContext(ctx).Create(integration).Error; err != nil {
		return fmt.Errorf("failed to create integration: %w", err)
	}
	return nil
}

// FindByID finds an integration by ID.
func (r *integrationRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.Integration, error) {
	var integration entity.Integration
	if err := r.db.WithContext(ctx).First(&integration, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("integration not found")
		}
		return nil, fmt.Errorf("failed to find integration: %w", err)
	}
	return &integration, nil
}

// FindByTenantID finds all integrations for a tenant.
func (r *integrationRepositoryImpl) FindByTenantID(ctx context.Context, tenantID uuid.UUID) ([]*entity.Integration, error) {
	var integrations []*entity.Integration
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&integrations).Error; err != nil {
		return nil, fmt.Errorf("failed to find integrations: %w", err)
	}
	return integrations, nil
}

// FindByTenantAndProvider finds an integration by tenant and provider.
func (r *integrationRepositoryImpl) FindByTenantAndProvider(ctx context.Context, tenantID uuid.UUID, provider string) (*entity.Integration, error) {
	var integration entity.Integration
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND provider = ?", tenantID, provider).
		First(&integration).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("integration not found")
		}
		return nil, fmt.Errorf("failed to find integration: %w", err)
	}
	return &integration, nil
}

// FindActive finds all active integrations for a tenant.
func (r *integrationRepositoryImpl) FindActive(ctx context.Context, tenantID uuid.UUID) ([]*entity.Integration, error) {
	var integrations []*entity.Integration
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("created_at DESC").
		Find(&integrations).Error; err != nil {
		return nil, fmt.Errorf("failed to find active integrations: %w", err)
	}
	return integrations, nil
}

// Update updates an integration.
func (r *integrationRepositoryImpl) Update(ctx context.Context, integration *entity.Integration) error {
	if err := r.db.WithContext(ctx).Save(integration).Error; err != nil {
		return fmt.Errorf("failed to update integration: %w", err)
	}
	return nil
}

// Delete deletes an integration.
func (r *integrationRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&entity.Integration{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}
	return nil
}

// List lists integrations with filters.
func (r *integrationRepositoryImpl) List(ctx context.Context, filters repository.IntegrationFilters) ([]*entity.Integration, int64, error) {
	var integrations []*entity.Integration
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.Integration{})

	// Apply filters
	if filters.TenantID != nil {
		query = query.Where("tenant_id = ?", *filters.TenantID)
	}

	if filters.Provider != "" {
		query = query.Where("provider = ?", filters.Provider)
	}

	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
	}

	if filters.IsActive != nil {
		query = query.Where("is_active = ?", *filters.IsActive)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count integrations: %w", err)
	}

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	// Execute query
	if err := query.Order("created_at DESC").Find(&integrations).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list integrations: %w", err)
	}

	return integrations, total, nil
}
