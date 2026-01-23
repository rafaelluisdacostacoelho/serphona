package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"gorm.io/gorm"
)

// toolRepositoryImpl implements the ToolRepository interface
type toolRepositoryImpl struct {
	db *gorm.DB
}

// NewToolRepository creates a new instance of ToolRepository
func NewToolRepository(db *gorm.DB) repository.ToolRepository {
	return &toolRepositoryImpl{db: db}
}

// accessibleScope restricts tools to public or enabled for the given tenant.
func (r *toolRepositoryImpl) accessibleScope(query *gorm.DB, tenantID uuid.UUID) *gorm.DB {
	return query.Where(
		"(is_public = ? OR EXISTS (SELECT 1 FROM tenant_tools tt WHERE tt.tool_id = tools.id AND tt.tenant_id = ? AND tt.is_enabled = ?))",
		true, tenantID, true,
	)
}

// Create creates a new tool
func (r *toolRepositoryImpl) Create(ctx context.Context, tool *entity.Tool) error {
	return r.db.WithContext(ctx).Create(tool).Error
}

// FindByID finds a tool by ID
func (r *toolRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*entity.Tool, error) {
	var tool entity.Tool
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&tool).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tool not found: %w", err)
		}
		return nil, err
	}
	return &tool, nil
}

// FindAccessibleByID finds a tool visible to the given tenant (public or enabled).
func (r *toolRepositoryImpl) FindAccessibleByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*entity.Tool, error) {
	var tool entity.Tool
	query := r.accessibleScope(r.db.WithContext(ctx).Model(&entity.Tool{}), tenantID).Where("id = ?", id)
	if err := query.First(&tool).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tool not found or not enabled for tenant: %w", err)
		}
		return nil, err
	}
	return &tool, nil
}

// FindByName finds a tool by name
func (r *toolRepositoryImpl) FindByName(ctx context.Context, name string) (*entity.Tool, error) {
	var tool entity.Tool
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tool).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tool not found: %w", err)
		}
		return nil, err
	}
	return &tool, nil
}

// FindAll finds all tools with optional filters
func (r *toolRepositoryImpl) FindAll(ctx context.Context, filters repository.ToolFilters) ([]*entity.Tool, int64, error) {
	var tools []*entity.Tool
	var total int64

	query := applyToolFilters(r.db.WithContext(ctx).Model(&entity.Tool{}), filters)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Find(&tools).Error; err != nil {
		return nil, 0, err
	}

	return tools, total, nil
}

// FindAllAccessible finds tools visible to the given tenant (public or enabled) with filters.
func (r *toolRepositoryImpl) FindAllAccessible(ctx context.Context, tenantID uuid.UUID, filters repository.ToolFilters) ([]*entity.Tool, int64, error) {
	var tools []*entity.Tool
	var total int64

	query := applyToolFilters(r.accessibleScope(r.db.WithContext(ctx).Model(&entity.Tool{}), tenantID), filters)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Find(&tools).Error; err != nil {
		return nil, 0, err
	}

	return tools, total, nil
}

// Update updates an existing tool
func (r *toolRepositoryImpl) Update(ctx context.Context, tool *entity.Tool) error {
	return r.db.WithContext(ctx).Save(tool).Error
}

// Delete deletes a tool by ID
func (r *toolRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.Tool{}, "id = ?", id).Error
}

// FindPublicTools finds all public tools
func (r *toolRepositoryImpl) FindPublicTools(ctx context.Context) ([]*entity.Tool, error) {
	var tools []*entity.Tool
	err := r.db.WithContext(ctx).
		Where("is_public = ? AND is_active = ?", true, true).
		Order("category ASC, display_name ASC").
		Find(&tools).Error
	if err != nil {
		return nil, err
	}
	return tools, nil
}

// FindByCategory finds tools by category
func (r *toolRepositoryImpl) FindByCategory(ctx context.Context, category string) ([]*entity.Tool, error) {
	var tools []*entity.Tool
	err := r.db.WithContext(ctx).
		Where("category = ? AND is_active = ?", category, true).
		Order("display_name ASC").
		Find(&tools).Error
	if err != nil {
		return nil, err
	}
	return tools, nil
}

// applyToolFilters applies common filters and pagination ordering.
func applyToolFilters(query *gorm.DB, filters repository.ToolFilters) *gorm.DB {
	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}

	if filters.IsActive != nil {
		query = query.Where("is_active = ?", *filters.IsActive)
	}

	if filters.IsPublic != nil {
		query = query.Where("is_public = ?", *filters.IsPublic)
	}

	if filters.Search != "" {
		searchPattern := "%" + filters.Search + "%"
		query = query.Where(
			"name ILIKE ? OR display_name ILIKE ? OR description ILIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	return query.Order("created_at DESC")
}
