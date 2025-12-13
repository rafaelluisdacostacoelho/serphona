// Package tenant contains the tenant domain model and business logic.
package tenant

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// Email regex for basic validation
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	// Slug regex (lowercase alphanumeric and hyphens)
	slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// Service encapsulates tenant domain business logic.
type Service struct {
	repo Repository
}

// NewService creates a new tenant domain service.
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// Create creates a new tenant with validation.
func (s *Service) Create(ctx context.Context, name, email string, plan Plan) (*Tenant, error) {
	// Validate inputs
	if err := s.validateName(name); err != nil {
		return nil, err
	}

	if err := s.validateEmail(email); err != nil {
		return nil, err
	}

	if err := s.validatePlan(plan); err != nil {
		return nil, err
	}

	// Check if email already exists
	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	// Generate slug from name
	slug := s.generateSlug(name)

	// Ensure slug is unique
	slug, err = s.ensureUniqueSlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique slug: %w", err)
	}

	// Create tenant
	tenant := NewTenant(name, email, plan)
	tenant.Slug = slug

	// Save to repository
	if err := s.repo.Save(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to save tenant: %w", err)
	}

	return tenant, nil
}

// Update updates an existing tenant with validation.
func (s *Service) Update(ctx context.Context, id uuid.UUID, name, email, phone string) (*Tenant, error) {
	// Get existing tenant
	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate inputs if changed
	if name != "" && name != tenant.Name {
		if err := s.validateName(name); err != nil {
			return nil, err
		}
		tenant.Name = name
	}

	if email != "" && email != tenant.Email {
		if err := s.validateEmail(email); err != nil {
			return nil, err
		}

		// Check if new email already exists
		exists, err := s.repo.ExistsByEmail(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("failed to check email existence: %w", err)
		}
		if exists {
			return nil, ErrEmailAlreadyExists
		}

		tenant.Email = email
	}

	if phone != "" {
		if err := s.validatePhone(phone); err != nil {
			return nil, err
		}
		tenant.Phone = phone
	}

	tenant.UpdatedAt = time.Now().UTC()

	// Save changes
	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	return tenant, nil
}

// UpdateSettings updates tenant settings with validation.
func (s *Service) UpdateSettings(ctx context.Context, id uuid.UUID, settings Settings) (*Tenant, error) {
	// Get existing tenant
	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate settings
	if err := s.validateSettings(settings); err != nil {
		return nil, err
	}

	// Update settings
	tenant.Settings = settings
	tenant.UpdatedAt = time.Now().UTC()

	// Save changes
	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to update tenant settings: %w", err)
	}

	return tenant, nil
}

// Activate activates a tenant.
func (s *Service) Activate(ctx context.Context, id uuid.UUID) error {
	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if tenant.Status == StatusActive {
		return ErrTenantAlreadyActive
	}

	if tenant.Status == StatusDeleted {
		return ErrTenantDeleted
	}

	tenant.Activate()

	if err := s.repo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("failed to activate tenant: %w", err)
	}

	return nil
}

// Suspend suspends a tenant.
func (s *Service) Suspend(ctx context.Context, id uuid.UUID, reason string) error {
	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if tenant.Status == StatusSuspended {
		return ErrTenantAlreadySuspended
	}

	if tenant.Status == StatusDeleted {
		return ErrTenantDeleted
	}

	tenant.Suspend()

	// Store suspension reason in metadata if provided
	if reason != "" {
		if tenant.Metadata.Custom == nil {
			tenant.Metadata.Custom = make(map[string]string)
		}
		tenant.Metadata.Custom["suspension_reason"] = reason
		tenant.Metadata.Custom["suspended_at"] = time.Now().UTC().Format(time.RFC3339)
	}

	if err := s.repo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("failed to suspend tenant: %w", err)
	}

	return nil
}

// Delete soft deletes a tenant.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if tenant.Status == StatusDeleted {
		return ErrTenantAlreadyDeleted
	}

	tenant.SoftDelete()

	if err := s.repo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	return nil
}

// ChangePlan changes the tenant's subscription plan.
func (s *Service) ChangePlan(ctx context.Context, id uuid.UUID, newPlan Plan) error {
	if err := s.validatePlan(newPlan); err != nil {
		return err
	}

	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if !tenant.IsActive() {
		return ErrTenantNotActive
	}

	if tenant.Plan == newPlan {
		return ErrSamePlan
	}

	oldPlan := tenant.Plan
	tenant.Plan = newPlan
	tenant.UpdatedAt = time.Now().UTC()

	// Adjust limits based on plan
	s.adjustPlanLimits(tenant, newPlan)

	// Store plan change in metadata
	if tenant.Metadata.Custom == nil {
		tenant.Metadata.Custom = make(map[string]string)
	}
	tenant.Metadata.Custom["previous_plan"] = string(oldPlan)
	tenant.Metadata.Custom["plan_changed_at"] = time.Now().UTC().Format(time.RFC3339)

	if err := s.repo.Update(ctx, tenant); err != nil {
		return fmt.Errorf("failed to change plan: %w", err)
	}

	return nil
}

// ValidateSlug validates and returns a slug.
func (s *Service) ValidateSlug(ctx context.Context, slug string) error {
	if !slugRegex.MatchString(slug) {
		return ErrInvalidSlug
	}

	exists, err := s.repo.ExistsBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("failed to check slug existence: %w", err)
	}

	if exists {
		return ErrSlugAlreadyExists
	}

	return nil
}

// validateName validates the tenant name.
func (s *Service) validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyName
	}

	if len(name) < 2 {
		return ErrNameTooShort
	}

	if len(name) > 100 {
		return ErrNameTooLong
	}

	return nil
}

// validateEmail validates the email format.
func (s *Service) validateEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return ErrEmptyEmail
	}

	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}

	return nil
}

// validatePhone validates the phone number format (basic validation).
func (s *Service) validatePhone(phone string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil // Phone is optional
	}

	// Basic validation: should contain only digits, spaces, +, -, (, )
	phoneRegex := regexp.MustCompile(`^[\d\s+\-()]+$`)
	if !phoneRegex.MatchString(phone) {
		return ErrInvalidPhone
	}

	if len(phone) < 10 || len(phone) > 20 {
		return ErrInvalidPhone
	}

	return nil
}

// validatePlan validates the subscription plan.
func (s *Service) validatePlan(plan Plan) error {
	switch plan {
	case PlanStarter, PlanProfessional, PlanEnterprise:
		return nil
	default:
		return ErrInvalidPlan
	}
}

// validateSettings validates tenant settings.
func (s *Service) validateSettings(settings Settings) error {
	// Validate telephony settings
	if settings.Telephony.MaxConcurrentCalls < 0 {
		return ErrInvalidSettings
	}

	if settings.Telephony.MaxConcurrentCalls > 1000 {
		return ErrInvalidSettings
	}

	// Validate AI agent settings
	if settings.AIAgent.MaxConversationMin < 1 || settings.AIAgent.MaxConversationMin > 240 {
		return ErrInvalidSettings
	}

	// Validate notification settings
	if settings.Notifications.WebhookEnabled && settings.Notifications.WebhookURL == "" {
		return ErrInvalidSettings
	}

	// Validate security settings
	if settings.Security.SessionTimeoutMin < 5 || settings.Security.SessionTimeoutMin > 1440 {
		return ErrInvalidSettings
	}

	if settings.Security.PasswordPolicy.MinLength < 6 || settings.Security.PasswordPolicy.MinLength > 128 {
		return ErrInvalidSettings
	}

	return nil
}

// generateSlug generates a URL-friendly slug from a name.
func (s *Service) generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}

	return slug
}

// ensureUniqueSlug ensures the slug is unique by appending a number if needed.
func (s *Service) ensureUniqueSlug(ctx context.Context, slug string) (string, error) {
	originalSlug := slug
	counter := 1

	for {
		exists, err := s.repo.ExistsBySlug(ctx, slug)
		if err != nil {
			return "", err
		}

		if !exists {
			return slug, nil
		}

		// Append counter to make it unique
		slug = fmt.Sprintf("%s-%d", originalSlug, counter)
		counter++

		// Prevent infinite loop
		if counter > 1000 {
			return "", fmt.Errorf("failed to generate unique slug after 1000 attempts")
		}
	}
}

// adjustPlanLimits adjusts tenant limits based on the plan.
func (s *Service) adjustPlanLimits(tenant *Tenant, plan Plan) {
	switch plan {
	case PlanStarter:
		tenant.Settings.Telephony.MaxConcurrentCalls = 5
	case PlanProfessional:
		tenant.Settings.Telephony.MaxConcurrentCalls = 20
	case PlanEnterprise:
		tenant.Settings.Telephony.MaxConcurrentCalls = 100
	}
}

// CanPerformAction checks if a tenant can perform a specific action.
func (s *Service) CanPerformAction(tenant *Tenant, action string) error {
	if !tenant.IsActive() {
		return ErrTenantNotActive
	}

	switch action {
	case "make_call":
		if !tenant.CanMakeCalls() {
			return ErrCallsNotAllowed
		}
	case "create_api_key":
		// Could check quota here
		return nil
	case "update_settings":
		return nil
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	return nil
}
