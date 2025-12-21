// Package usecase contains application use cases.
package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
)

// IntegrationService handles integration management operations.
type IntegrationService interface {
	// CreateIntegration creates a new integration.
	CreateIntegration(ctx context.Context, integration *entity.Integration) error

	// GetIntegration retrieves an integration by ID.
	GetIntegration(ctx context.Context, id uuid.UUID) (*entity.Integration, error)

	// GetIntegrationByProvider retrieves an integration by tenant and provider.
	GetIntegrationByProvider(ctx context.Context, tenantID uuid.UUID, provider string) (*entity.Integration, error)

	// ListIntegrations lists integrations with filters.
	ListIntegrations(ctx context.Context, filters repository.IntegrationFilters) ([]*entity.Integration, int64, error)

	// ListTenantIntegrations lists all integrations for a tenant.
	ListTenantIntegrations(ctx context.Context, tenantID uuid.UUID) ([]*entity.Integration, error)

	// ListActiveIntegrations lists active integrations for a tenant.
	ListActiveIntegrations(ctx context.Context, tenantID uuid.UUID) ([]*entity.Integration, error)

	// UpdateIntegration updates an integration.
	UpdateIntegration(ctx context.Context, integration *entity.Integration) error

	// DeleteIntegration deletes an integration.
	DeleteIntegration(ctx context.Context, id uuid.UUID) error

	// ActivateIntegration activates an integration.
	ActivateIntegration(ctx context.Context, id uuid.UUID) error

	// DeactivateIntegration deactivates an integration.
	DeactivateIntegration(ctx context.Context, id uuid.UUID) error

	// ValidateIntegration validates integration configuration.
	ValidateIntegration(ctx context.Context, integration *entity.Integration) error
}

// integrationServiceImpl implements IntegrationService.
type integrationServiceImpl struct {
	integrationRepo repository.IntegrationRepository
	oauthTokenRepo  repository.OAuthTokenRepository
}

// NewIntegrationService creates a new IntegrationService.
func NewIntegrationService(
	integrationRepo repository.IntegrationRepository,
	oauthTokenRepo repository.OAuthTokenRepository,
) IntegrationService {
	return &integrationServiceImpl{
		integrationRepo: integrationRepo,
		oauthTokenRepo:  oauthTokenRepo,
	}
}

// CreateIntegration creates a new integration.
func (s *integrationServiceImpl) CreateIntegration(ctx context.Context, integration *entity.Integration) error {
	// Validate integration
	if err := s.ValidateIntegration(ctx, integration); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check for duplicate provider for tenant
	existing, err := s.integrationRepo.FindByTenantAndProvider(ctx, integration.TenantID, integration.Provider)
	if err == nil && existing != nil {
		return fmt.Errorf("integration with provider '%s' already exists for this tenant", integration.Provider)
	}

	// Create integration
	if err := s.integrationRepo.Create(ctx, integration); err != nil {
		return fmt.Errorf("failed to create integration: %w", err)
	}

	return nil
}

// GetIntegration retrieves an integration by ID.
func (s *integrationServiceImpl) GetIntegration(ctx context.Context, id uuid.UUID) (*entity.Integration, error) {
	integration, err := s.integrationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}
	return integration, nil
}

// GetIntegrationByProvider retrieves an integration by tenant and provider.
func (s *integrationServiceImpl) GetIntegrationByProvider(ctx context.Context, tenantID uuid.UUID, provider string) (*entity.Integration, error) {
	integration, err := s.integrationRepo.FindByTenantAndProvider(ctx, tenantID, provider)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}
	return integration, nil
}

// ListIntegrations lists integrations with filters.
func (s *integrationServiceImpl) ListIntegrations(ctx context.Context, filters repository.IntegrationFilters) ([]*entity.Integration, int64, error) {
	return s.integrationRepo.List(ctx, filters)
}

// ListTenantIntegrations lists all integrations for a tenant.
func (s *integrationServiceImpl) ListTenantIntegrations(ctx context.Context, tenantID uuid.UUID) ([]*entity.Integration, error) {
	integrations, err := s.integrationRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations: %w", err)
	}
	return integrations, nil
}

// ListActiveIntegrations lists active integrations for a tenant.
func (s *integrationServiceImpl) ListActiveIntegrations(ctx context.Context, tenantID uuid.UUID) ([]*entity.Integration, error) {
	integrations, err := s.integrationRepo.FindActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active integrations: %w", err)
	}
	return integrations, nil
}

// UpdateIntegration updates an integration.
func (s *integrationServiceImpl) UpdateIntegration(ctx context.Context, integration *entity.Integration) error {
	// Check if integration exists
	existing, err := s.integrationRepo.FindByID(ctx, integration.ID)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	// Validate updated integration
	if err := s.ValidateIntegration(ctx, integration); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check for provider change conflicts
	if integration.Provider != existing.Provider {
		conflict, err := s.integrationRepo.FindByTenantAndProvider(ctx, integration.TenantID, integration.Provider)
		if err == nil && conflict != nil && conflict.ID != integration.ID {
			return fmt.Errorf("another integration with provider '%s' already exists", integration.Provider)
		}
	}

	// Update integration
	if err := s.integrationRepo.Update(ctx, integration); err != nil {
		return fmt.Errorf("failed to update integration: %w", err)
	}

	return nil
}

// DeleteIntegration deletes an integration.
func (s *integrationServiceImpl) DeleteIntegration(ctx context.Context, id uuid.UUID) error {
	// Check if integration exists
	_, err := s.integrationRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	// Revoke all OAuth tokens for this integration
	if err := s.oauthTokenRepo.RevokeByIntegration(ctx, id); err != nil {
		return fmt.Errorf("failed to revoke tokens: %w", err)
	}

	// Delete integration
	if err := s.integrationRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete integration: %w", err)
	}

	return nil
}

// ActivateIntegration activates an integration.
func (s *integrationServiceImpl) ActivateIntegration(ctx context.Context, id uuid.UUID) error {
	integration, err := s.integrationRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	integration.IsActive = true

	if err := s.integrationRepo.Update(ctx, integration); err != nil {
		return fmt.Errorf("failed to activate integration: %w", err)
	}

	return nil
}

// DeactivateIntegration deactivates an integration.
func (s *integrationServiceImpl) DeactivateIntegration(ctx context.Context, id uuid.UUID) error {
	integration, err := s.integrationRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("integration not found: %w", err)
	}

	integration.IsActive = false

	if err := s.integrationRepo.Update(ctx, integration); err != nil {
		return fmt.Errorf("failed to deactivate integration: %w", err)
	}

	return nil
}

// ValidateIntegration validates integration configuration.
func (s *integrationServiceImpl) ValidateIntegration(ctx context.Context, integration *entity.Integration) error {
	// Basic validations
	if integration.TenantID == uuid.Nil {
		return fmt.Errorf("tenant ID is required")
	}

	if integration.Name == "" {
		return fmt.Errorf("name is required")
	}

	if integration.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	if !integration.AuthType.IsValid() {
		return fmt.Errorf("invalid auth type: %s", integration.AuthType)
	}

	// Type-specific validations
	switch integration.Type {
	case entity.IntegrationTypeREST:
		// REST is always valid if auth is configured
	case entity.IntegrationTypeGraphQL:
		if integration.GraphQLConfig == nil {
			return fmt.Errorf("GraphQL config is required for GraphQL integration")
		}
		if integration.GraphQLConfig.Endpoint == "" {
			return fmt.Errorf("GraphQL endpoint is required")
		}
	case entity.IntegrationTypeSOAP:
		if integration.SOAPConfig == nil {
			return fmt.Errorf("SOAP config is required for SOAP integration")
		}
		if integration.SOAPConfig.Namespace == "" {
			return fmt.Errorf("SOAP namespace is required")
		}
	case entity.IntegrationTypeWebhook, entity.IntegrationTypeWebSocket:
		// Future implementation
	default:
		return fmt.Errorf("unsupported integration type: %s", integration.Type)
	}

	// OAuth2 specific validations
	if integration.AuthType == entity.AuthTypeOAuth2 {
		if integration.OAuth2Config == nil {
			return fmt.Errorf("OAuth2 config is required for OAuth2 auth type")
		}
		if integration.OAuth2Config.ClientID == "" {
			return fmt.Errorf("OAuth2 client ID is required")
		}
		if integration.OAuth2Config.ClientSecret == "" {
			return fmt.Errorf("OAuth2 client secret is required")
		}
		if integration.OAuth2Config.TokenURL == "" {
			return fmt.Errorf("OAuth2 token URL is required")
		}
	}

	return nil
}
