// Package tenant contains the application layer for tenant use cases.
package tenant

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
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
	s.cacheSet(ctx, fmt.Sprintf("tenant:%s", tenantEntity.ID), tenantEntity)

	// Publish created event
	s.publishCreated(ctx, tenantEntity)

	return toDTO(tenantEntity), nil
}

// GetTenant retrieves a tenant by ID.
func (s *Service) GetTenant(ctx context.Context, id uuid.UUID) (*TenantDTO, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("tenant:%s", id)
	if s.cache != nil {
		cached, err := s.cache.Get(ctx, cacheKey)
		if err == nil && cached != nil {
			return toDTO(cached), nil
		}
	}

	// Fetch from repository
	tenantEntity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get tenant", zap.String("id", id.String()), zap.Error(err))
		return nil, apperrors.NewNotFoundError(fmt.Sprintf("tenant with id %s not found", id))
	}

	// Cache for future requests
	s.cacheSet(ctx, cacheKey, tenantEntity)

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
	s.cacheInvalidate(ctx, tenantEntity.ID)

	// Publish updated event
	s.publishUpdated(ctx, tenantEntity)

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
	s.cacheInvalidate(ctx, id)

	// Publish deleted event
	s.publishDeleted(ctx, id)

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
	s.cacheInvalidate(ctx, id)

	// Publish event
	s.publishActivated(ctx, tenantEntity)

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
	s.cacheInvalidate(ctx, id)

	// Publish event
	s.publishSuspended(ctx, tenantEntity)

	return nil
}

// ValidateAPIKey validates an API key and returns the tenant ID.
func (s *Service) ValidateAPIKey(ctx context.Context, apiKey string) (*uuid.UUID, error) {
	if s.apiKeyRepo == nil {
		return nil, apperrors.NewInternalError("api key repository not configured")
	}
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

// GetQuota returns the quota for a tenant.
func (s *Service) GetQuota(ctx context.Context, tenantID uuid.UUID) (*QuotaDTO, error) {
	quota, err := s.repo.GetQuota(ctx, tenantID)
	if err != nil {
		s.logger.Error("failed to get tenant quota", zap.Error(err))
		return nil, apperrors.NewNotFoundError("quota not found for tenant")
	}

	return toQuotaDTO(quota), nil
}

// UpdateQuota updates tenant quota limits.
func (s *Service) UpdateQuota(ctx context.Context, cmd UpdateQuotaCommand) (*QuotaDTO, error) {
	if err := cmd.Validate(); err != nil {
		return nil, apperrors.NewValidationError(err.Error())
	}

	quota, err := s.repo.GetQuota(ctx, cmd.TenantID)
	if err != nil {
		s.logger.Error("failed to load quota before update", zap.Error(err))
		return nil, apperrors.NewNotFoundError("quota not found for tenant")
	}

	if cmd.MaxAPIKeys != nil {
		quota.MaxAPIKeys = *cmd.MaxAPIKeys
	}
	if cmd.MaxUsers != nil {
		quota.MaxUsers = *cmd.MaxUsers
	}
	if cmd.MaxCallsPerMonth != nil {
		quota.MaxCallsPerMonth = *cmd.MaxCallsPerMonth
	}
	if cmd.MaxMinutesPerMonth != nil {
		quota.MaxMinutesPerMonth = *cmd.MaxMinutesPerMonth
	}
	if cmd.MaxStorageGB != nil {
		quota.MaxStorageGB = *cmd.MaxStorageGB
	}
	if cmd.ResetAt != nil {
		quota.ResetAt = *cmd.ResetAt
	}

	if err := s.repo.UpdateQuota(ctx, quota); err != nil {
		s.logger.Error("failed to update quota", zap.Error(err))
		return nil, apperrors.NewInternalError("failed to update quota")
	}

	return toQuotaDTO(quota), nil
}

// IncrementUsage increments usage counters and returns the updated quota snapshot.
func (s *Service) IncrementUsage(ctx context.Context, cmd IncrementUsageCommand) (*QuotaDTO, error) {
	if err := cmd.Validate(); err != nil {
		return nil, apperrors.NewValidationError(err.Error())
	}

	if err := s.repo.IncrementUsage(ctx, cmd.TenantID, cmd.Calls, cmd.Minutes); err != nil {
		s.logger.Error("failed to increment tenant usage", zap.Error(err))
		return nil, apperrors.NewInternalError("failed to increment usage")
	}

	now := time.Now().UTC()
	period := now.Format("2006-01")

	reqID, err := authmw.RequestIDFromContext(ctx)
	if err != nil || reqID == "" {
		reqID = uuid.New().String()
	}

	usageEvent := tenant.UsageReportedEvent{
		TenantID:    cmd.TenantID,
		Period:      period,
		OccurredAt:  now,
		Source:      "tenant-manager",
		Calls:       cmd.Calls,
		Minutes:     cmd.Minutes,
		Messages:    0,
		StorageGB:   0,
		APIRequests: 0,
		RequestID:   reqID,
	}

	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishUsageReported(ctx, usageEvent); err != nil {
			s.logger.Error("failed to publish usage.reported event", zap.Error(err))
			return nil, apperrors.NewInternalError("failed to publish usage event")
		}
	}

	updated, err := s.repo.GetQuota(ctx, cmd.TenantID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to fetch updated quota")
	}

	return toQuotaDTO(updated), nil
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

// cache helpers tolerate nil cache to keep the service operational without Redis.
func (s *Service) cacheSet(ctx context.Context, key string, value *tenant.Tenant) {
	if s.cache == nil {
		return
	}
	if err := s.cache.Set(ctx, key, value); err != nil {
		s.logger.Warn("failed to cache tenant", zap.String("key", key), zap.Error(err))
	}
}

func (s *Service) cacheInvalidate(ctx context.Context, tenantID uuid.UUID) {
	if s.cache == nil {
		return
	}
	if err := s.cache.Invalidate(ctx, tenantID); err != nil {
		s.logger.Warn("failed to invalidate cache", zap.String("tenant_id", tenantID.String()), zap.Error(err))
	}
}

// publish helpers tolerate nil publisher to avoid panics when Kafka is disabled/unavailable.
func (s *Service) publishCreated(ctx context.Context, t *tenant.Tenant) {
	if s.eventPublisher == nil {
		return
	}
	if err := s.eventPublisher.PublishCreated(ctx, t); err != nil {
		s.logger.Error("failed to publish tenant created event", zap.Error(err))
	}
}

func (s *Service) publishUpdated(ctx context.Context, t *tenant.Tenant) {
	if s.eventPublisher == nil {
		return
	}
	if err := s.eventPublisher.PublishUpdated(ctx, t); err != nil {
		s.logger.Error("failed to publish tenant updated event", zap.Error(err))
	}
}

func (s *Service) publishDeleted(ctx context.Context, id uuid.UUID) {
	if s.eventPublisher == nil {
		return
	}
	if err := s.eventPublisher.PublishDeleted(ctx, id); err != nil {
		s.logger.Error("failed to publish tenant deleted event", zap.Error(err))
	}
}

func (s *Service) publishActivated(ctx context.Context, t *tenant.Tenant) {
	if s.eventPublisher == nil {
		return
	}
	if err := s.eventPublisher.PublishActivated(ctx, t); err != nil {
		s.logger.Error("failed to publish tenant activated event", zap.Error(err))
	}
}

func (s *Service) publishSuspended(ctx context.Context, t *tenant.Tenant) {
	if s.eventPublisher == nil {
		return
	}
	if err := s.eventPublisher.PublishSuspended(ctx, t); err != nil {
		s.logger.Error("failed to publish tenant suspended event", zap.Error(err))
	}
}

func toQuotaDTO(q *tenant.Quota) *QuotaDTO {
	return &QuotaDTO{
		TenantID:           q.TenantID,
		MaxAPIKeys:         q.MaxAPIKeys,
		MaxUsers:           q.MaxUsers,
		MaxCallsPerMonth:   q.MaxCallsPerMonth,
		MaxMinutesPerMonth: q.MaxMinutesPerMonth,
		MaxStorageGB:       q.MaxStorageGB,
		UsedCalls:          q.UsedCalls,
		UsedMinutes:        q.UsedMinutes,
		UsedStorageGB:      q.UsedStorageGB,
		ResetAt:            q.ResetAt,
	}
}
