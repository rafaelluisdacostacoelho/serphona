package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// ToolRepository defines the interface for tool data operations
type ToolRepository interface {
	// Create creates a new tool
	Create(ctx context.Context, tool *entity.Tool) error

	// FindByID finds a tool by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Tool, error)

	// FindByName finds a tool by name
	FindByName(ctx context.Context, name string) (*entity.Tool, error)

	// FindAll finds all tools with optional filters
	FindAll(ctx context.Context, filters ToolFilters) ([]*entity.Tool, int64, error)

	// Update updates an existing tool
	Update(ctx context.Context, tool *entity.Tool) error

	// Delete deletes a tool by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// FindPublicTools finds all public tools
	FindPublicTools(ctx context.Context) ([]*entity.Tool, error)

	// FindByCategory finds tools by category
	FindByCategory(ctx context.Context, category string) ([]*entity.Tool, error)
}

// ToolFilters represents filters for tool queries
type ToolFilters struct {
	Category string
	IsActive *bool
	IsPublic *bool
	Search   string // search in name, display_name, description
	Limit    int
	Offset   int
}
