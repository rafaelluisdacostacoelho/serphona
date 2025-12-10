package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/serphona/serphona/backend/go/services/tools-gateway/internal/domain/repository"
)

// TenantToolService defines the interface for tenant tool configuration operations
type TenantToolService interface {
	// ConfigureTool configures a tool for a tenant
	ConfigureTool(ctx context.Context, tenantTool *entity.TenantTool) error

	// GetTenantTool gets a tenant tool configuration
	GetTenantTool(ctx context.Context, tenantID, toolID uuid.UUID) (*entity.TenantTool, error)

	// ListTenantTools lists all tools for a tenant
	ListTenantTools(ctx context.Context, tenantID uuid.UUID, onlyEnabled bool) ([]*entity.TenantTool, error)

	// UpdateTenantTool updates a tenant tool configuration
	UpdateTenantTool(ctx context.Context, tenantTool *entity.TenantTool) error

	// EnableTool enables a tool for a tenant
	EnableTool(ctx context.Context, tenantID, toolID uuid.UUID) error

	// DisableTool disables a tool for a tenant
	DisableTool(ctx context.Context, tenantID, toolID uuid.UUID) error

	// RemoveTenantTool removes a tool configuration for a tenant
	RemoveTenantTool(ctx context.Context, tenantID, toolID uuid.UUID) error
}

// tenantToolServiceImpl implements TenantToolService
type tenantToolServiceImpl struct {
	toolRepo       repository.ToolRepository
	tenantToolRepo repository.TenantToolRepository
}

// NewTenantToolService creates a new TenantToolService
func NewTenantToolService(
	toolRepo repository.ToolRepository,
	tenantToolRepo repository.TenantToolRepository,
) TenantToolService {
	return &tenantToolServiceImpl{
		toolRepo:       toolRepo,
		tenantToolRepo: tenantToolRepo,
	}
}

// ConfigureTool configures a tool for a tenant
func (s *tenantToolServiceImpl) ConfigureTool(ctx context.Context, tenantTool *entity.TenantTool) error {
	// Verify tool exists
	tool, err := s.toolRepo.FindByID(ctx, tenantTool.ToolID)
	if err != nil {
		return fmt.Errorf("tool not found: %w", err)
	}

	// Check if tool is active
	if !tool.IsActive {
		return fmt.Errorf("cannot configure inactive tool '%s'", tool.Name)
	}

	// Check if configuration already exists
	existing, err := s.tenantToolRepo.FindByTenantAndTool(ctx, tenantTool.TenantID, tenantTool.ToolID)
	if err == nil && existing != nil {
		return fmt.Errorf("tool '%s' is already configured for this tenant", tool.Name)
	}

	// Create tenant tool configuration
	return s.tenantToolRepo.Create(ctx, tenantTool)
}

// GetTenantTool gets a tenant tool configuration
func (s *tenantToolServiceImpl) GetTenantTool(ctx context.Context, tenantID, toolID uuid.UUID) (*entity.TenantTool, error) {
	return s.tenantToolRepo.FindByTenantAndTool(ctx, tenantID, toolID)
}

// ListTenantTools lists all tools for a tenant
func (s *tenantToolServiceImpl) ListTenantTools(ctx context.Context, tenantID uuid.UUID, onlyEnabled bool) ([]*entity.TenantTool, error) {
	return s.tenantToolRepo.FindByTenant(ctx, tenantID, onlyEnabled)
}

// UpdateTenantTool updates a tenant tool configuration
func (s *tenantToolServiceImpl) UpdateTenantTool(ctx context.Context, tenantTool *entity.TenantTool) error {
	// Verify configuration exists
	existing, err := s.tenantToolRepo.FindByTenantAndTool(ctx, tenantTool.TenantID, tenantTool.ToolID)
	if err != nil {
		return fmt.Errorf("tenant tool configuration not found: %w", err)
	}

	// Update with existing ID
	tenantTool.ID = existing.ID

	return s.tenantToolRepo.Update(ctx, tenantTool)
}

// EnableTool enables a tool for a tenant
func (s *tenantToolServiceImpl) EnableTool(ctx context.Context, tenantID, toolID uuid.UUID) error {
	// Verify tool exists
	tool, err := s.toolRepo.FindByID(ctx, toolID)
	if err != nil {
		return fmt.Errorf("tool not found: %w", err)
	}

	// Check if tool is active
	if !tool.IsActive {
		return fmt.Errorf("cannot enable inactive tool '%s'", tool.Name)
	}

	// Check if configuration exists
	_, err = s.tenantToolRepo.FindByTenantAndTool(ctx, tenantID, toolID)
	if err != nil {
		// Create default configuration if it doesn't exist
		tenantTool := &entity.TenantTool{
			ID:        uuid.New(),
			TenantID:  tenantID,
			ToolID:    toolID,
			IsEnabled: true,
		}
		return s.tenantToolRepo.Create(ctx, tenantTool)
	}

	// Enable existing configuration
	return s.tenantToolRepo.Enable(ctx, tenantID, toolID)
}

// DisableTool disables a tool for a tenant
func (s *tenantToolServiceImpl) DisableTool(ctx context.Context, tenantID, toolID uuid.UUID) error {
	// Check if configuration exists
	_, err := s.tenantToolRepo.FindByTenantAndTool(ctx, tenantID, toolID)
	if err != nil {
		return fmt.Errorf("tenant tool configuration not found: %w", err)
	}

	return s.tenantToolRepo.Disable(ctx, tenantID, toolID)
}

// RemoveTenantTool removes a tool configuration for a tenant
func (s *tenantToolServiceImpl) RemoveTenantTool(ctx context.Context, tenantID, toolID uuid.UUID) error {
	// Get configuration
	tenantTool, err := s.tenantToolRepo.FindByTenantAndTool(ctx, tenantID, toolID)
	if err != nil {
		return fmt.Errorf("tenant tool configuration not found: %w", err)
	}

	return s.tenantToolRepo.Delete(ctx, tenantTool.ID)
}
