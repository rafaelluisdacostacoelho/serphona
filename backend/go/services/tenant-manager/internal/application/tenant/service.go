// Package tenant contains the application layer for tenant use cases.
package tenant

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"go.uber.org/zap"

	"tenant-manager/internal/domain/tenant"
	apperrors "tenant-manager/pkg/errors"
)

// APIKeyRepository defines the interface for API key operations.
type APIKeyRepository interface {
	GenerateAPIKey(ctx context.Context, tenantID uuid.UUID) (string, error)
	ValidateAPIKey(ctx context.Context, apiKey string) (*uuid.UUID, error)
}

// Service implements tenant use cases.
type Service struct {
	repo           tenant.Repository
	apiKeyRepo     APIKeyRepository
	cache          tenant.Cache
	eventPublisher tenant.EventPublisher
	logger         *zap.Logger
}

// NewService creates a new tenant service.
func NewService(
	repo tenant.Repository,
	apiKeyRepo APIKeyRepository,
	cache tenant.Cache,
	eventPublisher tenant.EventPublisher,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:           repo,
		apiKeyRepo:     apiKeyRepo,
		cache:          cache,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// CreateTenant creates a new tenant.
func (s *Service) CreateTenant(ctx context.Context, cmd CreateTenantCommand) (*TenantDTO, error) {
	// Validate command
	if err := cmd.Validate(); err != nil {
		return nil, apperrors.NewValidationError(err.Error())
	}

	// Check if email already exists
	exists, err := s.repo.ExistsByEmail(ctx, cmd.Email)
	if err != nil {
		s.logger.Error("failed to check email existence", zap.Error(err))
		return nil, apperrors.NewInternalError("failed to validate email")
	}
	if exists {
		return nil, apperrors.NewConflictError(fmt.Sprintf("tenant with email %s already exists", cmd.Email))
	}

	// Create tenant entity
	tenantEntity := tenant.NewTenant(cmd.Name, cmd.Email, tenant.Plan(cmd.Plan))

	// Generate slug
	baseSlug := slug.Make(cmd.Name)
	tenantSlug := baseSlug
	counter := 1
	for {
		exists, err := s.repo.ExistsBySlug(ctx, tenantSlug)
		if err != nil {
			s.logger.Error("failed to check slug existence", zap.Error(err))
			return nil, apperrors.NewInternalError("failed to generate slug")
		}
		if !exists {
			break
		}
		tenantSlug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}
	tenantEntity.Slug = tenantSlug

	// Set optional fields
	if cmd.Phone != "" {
		tenantEntity.Phone = cmd.Phone
	}
	if cmd.BillingEmail != "" {
		tenantEntity.BillingEmail = cmd.BillingEmail
	}

	// Set metadata
	if cmd.Industry != "" || cmd.CompanySize != "" || cmd.Website != "" {
		tenantEntity.Metadata = tenant.Metadata{
			Industry:    cmd.Industry,
			CompanySize: cmd.CompanySize,
			Website:     cmd.Website,
		}
	}

	// Persist tenant
	if err := s.repo.Create(ctx, tenantEntity); err != nil {
		s.logger.Error("failed to create tenant", zap.Error(err))
		return nil, apperrors.NewInternalError("failed to create tenant")
	}

	// Activate tenant immediately (or keep as pending based on business logic)
	tenantEntity.Activate()
	if err := s.repo.Update(ctx, tenantEntity); err != nil {
		s.logger.Error("failed to activate tenant", zap.Error(err))
		// Continue anyway, tenant is created
	}

	// Cache the tenant
	if err := s.cache.Set(ctx, fmt.Sprintf("tenant:%s", tenantEntity.ID), tenantEntity); err != nil {
		s.logger.Warn("failed to cache tenant", zap.Error(err))
		// Non-critical error, continue
	}

	// Publish created event
	if err := s.eventPublisher.PublishCreated(ctx, tenantEntity); err != nil {
		s.logger.Error("failed to publish tenant created event", zap.Error(err))
		// Non-critical error, continue
	}

	return toDTO(tenantEntity), nil
}

// GetTenant retrieves a tenant by ID.
func (s *Service) GetTenant(ctx context.Context, id uuid.UUID) (*TenantDTO, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("tenant:%s", id)
	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return toDTO(cached), nil
	}

	// Fetch from repository
	tenantEntity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get tenant", zap.String("id", id.String()), zap.Error(err))
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("tenant with id %s not found", id))
	}

	// Cache for future requests
	if err := s.cache.Set(ctx, cacheKey, tenantEntity); err != nil {
		s.logger.Warn("failed to cache tenant", zap.Error(err))
	}

	return toDTO(tenantEntity), nil
}

// GetTenantBySlug retrieves a tenant by slug.
func (s *Service) GetTenantBySlug(ctx context.Context, slug string) (*TenantDTO, error) {
	tenantEntity, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("tenant with slug %s not found", slug))
	}

	return toDTO(tenantEntity), nil
}

// UpdateTenant updates an existing tenant.
func (s *Service) UpdateTenant(ctx context.Context, cmd UpdateTenantCommand) (*TenantDTO, error) {
	// Validate command
	if err := cmd.Validate(); err != nil {
		return nil, apperrors.NewValidationError(err.Error())
	}

	// Fetch existing tenant
	tenantEntity, err := s.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("tenant with id %s not found", cmd.ID))
	}

	// Check if tenant is deleted
	if tenantEntity.Status == tenant.StatusDeleted {
		return nil, apperrors.NewValidationError("cannot update deleted tenant")
	}

	// Update fields if provided
	if cmd.Name != nil {
		tenantEntity.Name = *cmd.Name
		// Optionally update slug when name changes
		// tenantEntity.Slug = slug.Make(*cmd.Name)
	}
	if cmd.Email != nil {
		// Check if new email already exists
		if *cmd.Email != tenantEntity.Email {
			exists, err := s.repo.ExistsByEmail(ctx, *cmd.Email)
			if err != nil {
				return nil, apperrors.NewInternalError("failed to validate email")
			}
			if exists {
				return nil, apperrors.NewConflictError(fmt.Sprintf("email %s already in use", *cmd.Email))
			}
			tenantEntity.Email = *cmd.Email
		}
	}
	if cmd.Phone != nil {
		tenantEntity.Phone = *cmd.Phone
	}
	if cmd.BillingEmail != nil {
		tenantEntity.BillingEmail = *cmd.BillingEmail
	}

	tenantEntity.UpdatedAt = time.Now().UTC()

	// Persist changes
	if err := s.repo.Update(ctx, tenantEntity); err != nil {
		s.logger.Error("failed to update tenant", zap.Error(err))
		return nil, apperrors.NewInternalError("failed to update tenant")
	}

	// Invalidate cache
	if err := s.cache.Invalidate(ctx, tenantEntity.ID); err != nil {
		s.logger.Warn("failed to invalidate cache", zap.Error(err))
	}

	// Publish updated event
	if err := s.eventPublisher.PublishUpdated(ctx, tenantEntity); err != nil {
		s.logger.Error("failed to publish tenant updated event", zap.Error(err))
	}

	return toDTO(tenantEntity), nil
}

// DeleteTenant soft-deletes a tenant.
func (s *Service) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	// Fetch tenant
	tenantEntity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.NewNotFoundError(fmt.Sprintf("tenant with id %s not found", id))
	}

	// Check if already deleted
	if tenantEntity.Status == tenant.StatusDeleted {
		return apperrors.NewValidationError("tenant already deleted")
	}

	// Soft delete
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete tenant", zap.Error(err))
		return apperrors.NewInternalError("failed to delete tenant")
	}

	// Invalidate cache
	if err := s.cache.Invalidate(ctx, id); err != nil {
		s.logger.Warn("failed to invalidate cache", zap.Error(err))
	}

	// Publish deleted event
	if err := s.eventPublisher.PublishDeleted(ctx, id); err != nil {
		s.logger.Error("failed to publish tenant deleted event", zap.Error(err))
	}

	return nil
}

// ListTenants lists tenants with pagination and filtering.
func (s *Service) ListTenants(ctx context.Context, query ListTenantsQuery) (*ListTenantsResult, error) {
	// Validate query
	if err := query.Validate(); err != nil {
		return nil, apperrors.NewValidationError(err.Error())
	}

	// Build filter
	filter := tenant.ListFilter{
		PageSize:   query.PageSize,
		PageNumber: query.Page,
		Search:     query.Search,
		SortBy:     "created_at",
		SortOrder:  "desc",
	}

	if query.Status != "" {
		status := tenant.Status(query.Status)
		filter.Status = &status
	}

	// Fetch from repository
	result, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error("failed to list tenants", zap.Error(err))
		return nil, apperrors.NewInternalError("failed to list tenants")
	}

	// Convert to DTOs
	tenants := make([]*TenantDTO, len(result.Tenants))
	for i, t := range result.Tenants {
		tenants[i] = toDTO(t)
	}

	return &ListTenantsResult{
		Tenants:    tenants,
		Total:      result.Total,
		Page:       result.PageNumber,
		PageSize:   result.PageSize,
		TotalPages: result.TotalPages,
	}, nil
}

// ActivateTenant activates a tenant.
func (s *Service) ActivateTenant(ctx context.Context, id uuid.UUID) error {
	tenantEntity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.NewNotFoundError(fmt.Sprintf("tenant with id %s not found", id))
	}

	if tenantEntity.Status == tenant.StatusActive {
		return nil // Already active
	}

	tenantEntity.Activate()
	if err := s.repo.Update(ctx, tenantEntity); err != nil {
		return apperrors.NewInternalError("failed to activate tenant")
	}

	// Invalidate cache
	s.cache.Invalidate(ctx, id)

	// Publish event
	s.eventPublisher.PublishActivated(ctx, tenantEntity)

	return nil
}

// SuspendTenant suspends a tenant.
func (s *Service) SuspendTenant(ctx context.Context, id uuid.UUID) error {
	tenantEntity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperrors.NewNotFoundError(fmt.Sprintf("tenant with id %s not found", id))
	}

	if tenantEntity.Status == tenant.StatusSuspended {
		return nil // Already suspended
	}

	tenantEntity.Suspend()
	if err := s.repo.Update(ctx, tenantEntity); err != nil {
		return apperrors.NewInternalError("failed to suspend tenant")
	}

	// Invalidate cache
	s.cache.Invalidate(ctx, id)

	// Publish event
	s.eventPublisher.PublishSuspended(ctx, tenantEntity)

	return nil
}

// ValidateAPIKey validates an API key and returns the tenant ID.
func (s *Service) ValidateAPIKey(ctx context.Context, apiKey string) (*uuid.UUID, error) {
	if apiKey == "" {
		return nil, apperrors.NewUnauthorizedError("API key is required")
	}

	tenantID, err := s.apiKeyRepo.ValidateAPIKey(ctx, apiKey)
	if err != nil {
		return nil, apperrors.NewUnauthorizedError("invalid API key")
	}

	// Check if tenant is active
	tenantEntity, err := s.repo.GetByID(ctx, *tenantID)
	if err != nil {
		return nil, apperrors.NewUnauthorizedError("tenant not found")
	}

	if !tenantEntity.IsActive() {
		return nil, apperrors.NewForbiddenError("tenant is not active")
	}

	return tenantID, nil
}

// toDTO converts a domain tenant to a DTO.
func toDTO(t *tenant.Tenant) *TenantDTO {
	return &TenantDTO{
		ID:           t.ID,
		Name:         t.Name,
		Slug:         t.Slug,
		Email:        t.Email,
		Phone:        t.Phone,
		Status:       string(t.Status),
		Plan:         string(t.Plan),
		Settings:     t.Settings,
		Metadata:     t.Metadata,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
		BillingEmail: t.BillingEmail,
	}
}

// Helper to normalize strings
func normalizeString(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
