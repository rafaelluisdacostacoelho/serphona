package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/service"
)

// ToolService defines the interface for tool management operations
type ToolService interface {
	// CreateTool creates a new tool
	CreateTool(ctx context.Context, tool *entity.Tool) error

	// GetTool gets a tool by ID
	GetTool(ctx context.Context, tenantID uuid.UUID, toolID uuid.UUID) (*entity.Tool, error)

	// GetToolByName gets a tool by name
	GetToolByName(ctx context.Context, tenantID uuid.UUID, name string) (*entity.Tool, error)

	// ListTools lists all tools with filters
	ListTools(ctx context.Context, tenantID uuid.UUID, filters repository.ToolFilters) ([]*entity.Tool, int64, error)

	// UpdateTool updates a tool
	UpdateTool(ctx context.Context, tool *entity.Tool) error

	// DeleteTool deletes a tool
	DeleteTool(ctx context.Context, toolID uuid.UUID) error

	// ListPublicTools lists all public tools available to all tenants
	ListPublicTools(ctx context.Context) ([]*entity.Tool, error)

	// ListToolsByCategory lists tools by category
	ListToolsByCategory(ctx context.Context, category string) ([]*entity.Tool, error)
}

// toolServiceImpl implements ToolService
type toolServiceImpl struct {
	toolRepo  repository.ToolRepository
	validator service.SchemaValidator
	policy    ExecutionPolicy
}

// NewToolService creates a new ToolService
func NewToolService(
	toolRepo repository.ToolRepository,
	validator service.SchemaValidator,
	policy ExecutionPolicy,
) ToolService {
	return &toolServiceImpl{
		toolRepo:  toolRepo,
		validator: validator,
		policy:    policy,
	}
}

// CreateTool creates a new tool
func (s *toolServiceImpl) CreateTool(ctx context.Context, tool *entity.Tool) error {
	// Validate input schema
	if err := s.validator.IsValidSchema(tool.InputSchema); err != nil {
		return fmt.Errorf("invalid input schema: %w", err)
	}

	// Validate output schema
	if err := s.validator.IsValidSchema(tool.OutputSchema); err != nil {
		return fmt.Errorf("invalid output schema: %w", err)
	}

	// Validate tool name uniqueness
	existing, err := s.toolRepo.FindByName(ctx, tool.Name)
	if err == nil && existing != nil {
		return fmt.Errorf("tool with name '%s' already exists", tool.Name)
	}

	// Validate auth type
	if !isValidAuthType(tool.AuthType) {
		return fmt.Errorf("invalid auth type: %s", tool.AuthType)
	}

	// Validate HTTP method
	if !isValidHTTPMethod(tool.Method) {
		return fmt.Errorf("invalid HTTP method: %s", tool.Method)
	}

	if err := s.validateBaseURL(tool.BaseURL); err != nil {
		return err
	}

	// Create tool
	return s.toolRepo.Create(ctx, tool)
}

// GetTool gets a tool by ID
func (s *toolServiceImpl) GetTool(ctx context.Context, tenantID uuid.UUID, toolID uuid.UUID) (*entity.Tool, error) {
	return s.toolRepo.FindAccessibleByID(ctx, tenantID, toolID)
}

// GetToolByName gets a tool by name
func (s *toolServiceImpl) GetToolByName(ctx context.Context, tenantID uuid.UUID, name string) (*entity.Tool, error) {
	tool, err := s.toolRepo.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.toolRepo.FindAccessibleByID(ctx, tenantID, tool.ID)
}

// ListTools lists all tools with filters
func (s *toolServiceImpl) ListTools(ctx context.Context, tenantID uuid.UUID, filters repository.ToolFilters) ([]*entity.Tool, int64, error) {
	return s.toolRepo.FindAllAccessible(ctx, tenantID, filters)
}

// UpdateTool updates a tool
func (s *toolServiceImpl) UpdateTool(ctx context.Context, tool *entity.Tool) error {
	// Check if tool exists
	existing, err := s.toolRepo.FindByID(ctx, tool.ID)
	if err != nil {
		return fmt.Errorf("tool not found: %w", err)
	}

	// If name is being changed, check uniqueness
	if tool.Name != existing.Name {
		nameExists, err := s.toolRepo.FindByName(ctx, tool.Name)
		if err == nil && nameExists != nil && nameExists.ID != tool.ID {
			return fmt.Errorf("tool with name '%s' already exists", tool.Name)
		}
	}

	// Validate schemas if changed
	if string(tool.InputSchema) != string(existing.InputSchema) {
		if err := s.validator.IsValidSchema(tool.InputSchema); err != nil {
			return fmt.Errorf("invalid input schema: %w", err)
		}
	}

	if string(tool.OutputSchema) != string(existing.OutputSchema) {
		if err := s.validator.IsValidSchema(tool.OutputSchema); err != nil {
			return fmt.Errorf("invalid output schema: %w", err)
		}
	}

	// Validate auth type
	if !isValidAuthType(tool.AuthType) {
		return fmt.Errorf("invalid auth type: %s", tool.AuthType)
	}

	// Validate HTTP method
	if !isValidHTTPMethod(tool.Method) {
		return fmt.Errorf("invalid HTTP method: %s", tool.Method)
	}

	if err := s.validateBaseURL(tool.BaseURL); err != nil {
		return err
	}

	return s.toolRepo.Update(ctx, tool)
}

// DeleteTool deletes a tool
func (s *toolServiceImpl) DeleteTool(ctx context.Context, toolID uuid.UUID) error {
	// Check if tool exists
	_, err := s.toolRepo.FindByID(ctx, toolID)
	if err != nil {
		return fmt.Errorf("tool not found: %w", err)
	}

	return s.toolRepo.Delete(ctx, toolID)
}

// ListPublicTools lists all public tools
func (s *toolServiceImpl) ListPublicTools(ctx context.Context) ([]*entity.Tool, error) {
	return s.toolRepo.FindPublicTools(ctx)
}

// ListToolsByCategory lists tools by category
func (s *toolServiceImpl) ListToolsByCategory(ctx context.Context, category string) ([]*entity.Tool, error) {
	return s.toolRepo.FindByCategory(ctx, category)
}

// Helper functions

func isValidAuthType(authType string) bool {
	validTypes := []string{
		entity.AuthTypeNone.String(),
		entity.AuthTypeAPIKey.String(),
		entity.AuthTypeBearer.String(),
		entity.AuthTypeBasic.String(),
		entity.AuthTypeOAuth2.String(),
	}

	for _, valid := range validTypes {
		if authType == valid {
			return true
		}
	}

	return false
}

func isValidHTTPMethod(method string) bool {
	validMethods := []string{
		entity.HTTPMethodGET,
		entity.HTTPMethodPOST,
		entity.HTTPMethodPUT,
		entity.HTTPMethodDELETE,
		entity.HTTPMethodPATCH,
	}

	for _, valid := range validMethods {
		if method == valid {
			return true
		}
	}

	return false
}

func (s *toolServiceImpl) validateBaseURL(rawURL string) error {
	if len(s.policy.AllowedHosts) == 0 {
		return errors.New("no allowed hosts configured; set EXEC_ALLOWED_HOSTS")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("base_url must be a valid URL: %w", err)
	}

	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("base_url must use https scheme")
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("base_url must include a host")
	}

	for _, allowed := range s.policy.AllowedHosts {
		if strings.EqualFold(allowed, host) {
			return nil
		}
	}

	return fmt.Errorf("base_url host '%s' is not in allowed list", host)
}
