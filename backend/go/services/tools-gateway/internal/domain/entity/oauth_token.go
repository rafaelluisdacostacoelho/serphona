// Package entity contains domain entities for tools gateway.
package entity

import (
	"time"

	"github.com/google/uuid"
)

// OAuthToken represents an OAuth 2.0 access/refresh token pair.
type OAuthToken struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	IntegrationID uuid.UUID  `json:"integration_id" gorm:"type:uuid;not null;index"`
	TenantID      uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID        *uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;index"` // null for client_credentials

	// Tokens (encrypted in DB)
	AccessToken  string `json:"-" gorm:"type:text;not null"` // Never expose in JSON
	RefreshToken string `json:"-" gorm:"type:text"`          // Never expose in JSON
	TokenType    string `json:"token_type" gorm:"type:varchar(50);default:'Bearer'"`

	// Expiration
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RefreshAt *time.Time `json:"refresh_at,omitempty"` // When to proactively refresh

	// OAuth metadata
	Scopes []string               `json:"scopes" gorm:"type:jsonb"`
	Extra  map[string]interface{} `json:"extra,omitempty" gorm:"type:jsonb"` // For provider-specific fields

	// Status
	IsValid   bool       `json:"is_valid" gorm:"default:true"`
	IsRevoked bool       `json:"is_revoked" gorm:"default:false"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (OAuthToken) TableName() string {
	return "oauth_tokens"
}

// BeforeCreate hook for GORM.
func (t *OAuthToken) BeforeCreate() error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now
	return nil
}

// BeforeUpdate hook for GORM.
func (t *OAuthToken) BeforeUpdate() error {
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// IsExpired checks if the token is expired.
func (t *OAuthToken) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*t.ExpiresAt)
}

// NeedsRefresh checks if the token should be refreshed.
func (t *OAuthToken) NeedsRefresh() bool {
	if t.RefreshToken == "" {
		return false
	}
	if t.IsExpired() {
		return true
	}
	if t.RefreshAt != nil && time.Now().UTC().After(*t.RefreshAt) {
		return true
	}
	return false
}

// CanBeUsed checks if the token can be used for requests.
func (t *OAuthToken) CanBeUsed() bool {
	return t.IsValid && !t.IsRevoked && !t.IsExpired()
}

// Revoke marks the token as revoked.
func (t *OAuthToken) Revoke() {
	now := time.Now().UTC()
	t.IsRevoked = true
	t.IsValid = false
	t.RevokedAt = &now
}

// OAuthState represents OAuth 2.0 authorization state for CSRF protection.
type OAuthState struct {
	ID            uuid.UUID         `json:"id" gorm:"type:uuid;primary_key"`
	IntegrationID uuid.UUID         `json:"integration_id" gorm:"type:uuid;not null;index"`
	TenantID      uuid.UUID         `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID        uuid.UUID         `json:"user_id" gorm:"type:uuid;not null;index"`
	State         string            `json:"state" gorm:"type:varchar(255);not null;uniqueIndex"`
	CodeVerifier  string            `json:"code_verifier,omitempty" gorm:"type:varchar(255)"` // For PKCE
	RedirectURI   string            `json:"redirect_uri" gorm:"type:text;not null"`
	Scopes        []string          `json:"scopes" gorm:"type:jsonb"`
	ExtraParams   map[string]string `json:"extra_params,omitempty" gorm:"type:jsonb"`

	// Status
	IsUsed    bool       `json:"is_used" gorm:"default:false"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`

	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name for GORM.
func (OAuthState) TableName() string {
	return "oauth_states"
}

// BeforeCreate hook for GORM.
func (s *OAuthState) BeforeCreate() error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	s.CreatedAt = time.Now().UTC()
	return nil
}

// IsExpired checks if the state has expired.
func (s *OAuthState) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// IsValid checks if the state is valid for use.
func (s *OAuthState) IsValid() bool {
	return !s.IsUsed && !s.IsExpired()
}

// MarkAsUsed marks the state as used.
func (s *OAuthState) MarkAsUsed() {
	now := time.Now().UTC()
	s.IsUsed = true
	s.UsedAt = &now
}
