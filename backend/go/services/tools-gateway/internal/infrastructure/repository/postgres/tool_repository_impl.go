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

	query := r.db.WithContext(ctx).Model(&entity.Tool{})

	// Apply filters
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

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}

	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	// Order by created_at desc
	query = query.Order("created_at DESC")

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
