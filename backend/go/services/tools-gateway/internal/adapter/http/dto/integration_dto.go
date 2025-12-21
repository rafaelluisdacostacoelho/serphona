// Package dto contains Data Transfer Objects for HTTP API.
package dto

import (
	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
)

// CreateIntegrationRequest represents a create integration request.
type CreateIntegrationRequest struct {
	Name           string                 `json:"name" binding:"required"`
	DisplayName    string                 `json:"display_name" binding:"required"`
	Description    string                 `json:"description"`
	Provider       string                 `json:"provider"`
	Type           string                 `json:"type" binding:"required"`
	BaseURL        string                 `json:"base_url" binding:"required"`
	AuthType       string                 `json:"auth_type" binding:"required"`
	AuthConfig     map[string]interface{} `json:"auth_config"`
	OAuth2Config   *entity.OAuth2Config   `json:"oauth2_config,omitempty"`
	GraphQLConfig  *entity.GraphQLConfig  `json:"graphql_config,omitempty"`
	SOAPConfig     *entity.SOAPConfig     `json:"soap_config,omitempty"`
	DefaultHeaders map[string]string      `json:"default_headers"`
	Settings       map[string]interface{} `json:"settings"`
}

// ToEntity converts DTO to entity.
func (r *CreateIntegrationRequest) ToEntity(tenantID uuid.UUID) *entity.Integration {
	return &entity.Integration{
		TenantID:       tenantID,
		Name:           r.Name,
		DisplayName:    r.DisplayName,
		Description:    r.Description,
		Provider:       r.Provider,
		Type:           entity.IntegrationType(r.Type),
		BaseURL:        r.BaseURL,
		AuthType:       entity.AuthType(r.AuthType),
		AuthConfig:     r.AuthConfig,
		OAuth2Config:   r.OAuth2Config,
		GraphQLConfig:  r.GraphQLConfig,
		SOAPConfig:     r.SOAPConfig,
		DefaultHeaders: r.DefaultHeaders,
		Settings:       r.Settings,
		IsActive:       true,
	}
}

// UpdateIntegrationRequest represents an update integration request.
type UpdateIntegrationRequest struct {
	Name           string                 `json:"name"`
	DisplayName    string                 `json:"display_name"`
	Description    string                 `json:"description"`
	Provider       string                 `json:"provider"`
	Type           string                 `json:"type"`
	BaseURL        string                 `json:"base_url"`
	AuthType       string                 `json:"auth_type"`
	AuthConfig     map[string]interface{} `json:"auth_config"`
	OAuth2Config   *entity.OAuth2Config   `json:"oauth2_config"`
	GraphQLConfig  *entity.GraphQLConfig  `json:"graphql_config"`
	SOAPConfig     *entity.SOAPConfig     `json:"soap_config"`
	DefaultHeaders map[string]string      `json:"default_headers"`
	Settings       map[string]interface{} `json:"settings"`
	IsActive       *bool                  `json:"is_active"`
}

// ToEntity converts DTO to entity.
func (r *UpdateIntegrationRequest) ToEntity(id, tenantID uuid.UUID) *entity.Integration {
	integration := &entity.Integration{
		ID:       id,
		TenantID: tenantID,
	}

	if r.Name != "" {
		integration.Name = r.Name
	}
	if r.DisplayName != "" {
		integration.DisplayName = r.DisplayName
	}
	if r.Description != "" {
		integration.Description = r.Description
	}
	if r.Provider != "" {
		integration.Provider = r.Provider
	}
	if r.Type != "" {
		integration.Type = entity.IntegrationType(r.Type)
	}
	if r.BaseURL != "" {
		integration.BaseURL = r.BaseURL
	}
	if r.AuthType != "" {
		integration.AuthType = entity.AuthType(r.AuthType)
	}
	if r.AuthConfig != nil {
		integration.AuthConfig = r.AuthConfig
	}
	if r.OAuth2Config != nil {
		integration.OAuth2Config = r.OAuth2Config
	}
	if r.GraphQLConfig != nil {
		integration.GraphQLConfig = r.GraphQLConfig
	}
	if r.SOAPConfig != nil {
		integration.SOAPConfig = r.SOAPConfig
	}
	if r.DefaultHeaders != nil {
		integration.DefaultHeaders = r.DefaultHeaders
	}
	if r.Settings != nil {
		integration.Settings = r.Settings
	}
	if r.IsActive != nil {
		integration.IsActive = *r.IsActive
	}

	return integration
}

// IntegrationResponse represents an integration response.
type IntegrationResponse struct {
	Integration *entity.Integration `json:"integration"`
}

// ListIntegrationsResponse represents a list integrations response.
type ListIntegrationsResponse struct {
	Integrations []*entity.Integration `json:"integrations"`
	Total        int64                 `json:"total"`
	Limit        int                   `json:"limit"`
	Offset       int                   `json:"offset"`
}

// OAuthAuthorizeResponse represents OAuth authorize response.
type OAuthAuthorizeResponse struct {
	AuthURL string `json:"auth_url"`
}

// OAuthCallbackResponse represents OAuth callback response.
type OAuthCallbackResponse struct {
	TokenID   uuid.UUID `json:"token_id"`
	Message   string    `json:"message"`
	ExpiresAt *string   `json:"expires_at,omitempty"`
}
