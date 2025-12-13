// Package apikey contains the API key domain model and business logic.
package apikey

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service encapsulates API key domain business logic.
type Service struct {
	repo Repository
}

// NewService creates a new API key domain service.
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// Create creates a new API key with validation.
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, name string, createdBy uuid.UUID, permissions []string, expiryDays int) (*APIKey, string, error) {
	// Validate inputs
	if err := s.validateName(name); err != nil {
		return nil, "", err
	}

	if err := s.validatePermissions(permissions); err != nil {
		return nil, "", err
	}

	// Check if name already exists for this tenant
	exists, err := s.repo.ExistsByName(ctx, tenantID, name)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check name existence: %w", err)
	}
	if exists {
		return nil, "", ErrNameAlreadyExists
	}

	// Create API key
	apiKey, rawKey, err := NewAPIKey(tenantID, name, createdBy, permissions)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}

	// Set custom expiry if provided
	if expiryDays > 0 {
		expiry := time.Now().UTC().AddDate(0, 0, expiryDays)
		apiKey.ExpiresAt = &expiry
	}

	// Save to repository
	if err := s.repo.Save(ctx, apiKey); err != nil {
		return nil, "", fmt.Errorf("failed to save API key: %w", err)
	}

	// Return sanitized key and raw key (only time it's visible)
	return apiKey.Sanitize(), rawKey, nil
}

// Revoke revokes an API key.
func (s *Service) Revoke(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID) error {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if key.Status == StatusRevoked {
		return ErrKeyRevoked
	}

	key.Revoke(revokedBy)

	if err := s.repo.Update(ctx, key); err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	return nil
}

// RevokeAllForTenant revokes all API keys for a tenant.
func (s *Service) RevokeAllForTenant(ctx context.Context, tenantID uuid.UUID, revokedBy uuid.UUID) error {
	if err := s.repo.RevokeAll(ctx, tenantID, revokedBy); err != nil {
		return fmt.Errorf("failed to revoke all API keys: %w", err)
	}

	return nil
}

// Authenticate authenticates an API key and returns the key if valid.
func (s *Service) Authenticate(ctx context.Context, rawKey string, ipAddress string) (*APIKey, error) {
	// Validate key format
	if !ValidateKeyFormat(rawKey) {
		return nil, ErrInvalidKey
	}

	// Hash the key for lookup
	keyHash := hashKey(rawKey)

	// Find key by hash
	key, err := s.repo.FindByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, ErrInvalidKeyHash
	}

	// Verify key hash
	if !key.VerifyKey(rawKey) {
		return nil, ErrInvalidKeyHash
	}

	// Check if key is active
	if !key.IsActive() {
		if key.IsExpired() {
			return nil, NewAPIKeyExpiredError(key.ID, key.ExpiresAt.Format(time.RFC3339))
		}
		if key.Status == StatusRevoked {
			revokedAt := ""
			if key.RevokedAt != nil {
				revokedAt = key.RevokedAt.Format(time.RFC3339)
			}
			return nil, NewAPIKeyRevokedError(key.ID, revokedAt, "")
		}
		return nil, ErrKeyInactive
	}

	// Check IP whitelist
	if !key.IsAllowedFromIP(ipAddress) {
		return nil, NewIPNotWhitelistedError(ipAddress, key.Metadata.IPWhitelist)
	}

	// Return sanitized key
	return key.Sanitize(), nil
}

// RecordUsage records usage of an API key.
func (s *Service) RecordUsage(ctx context.Context, id uuid.UUID, ipAddress, userAgent string, success bool) error {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if success {
		key.RecordUsage(ipAddress, userAgent)
	} else {
		key.RecordError()
	}

	if err := s.repo.Update(ctx, key); err != nil {
		return fmt.Errorf("failed to record usage: %w", err)
	}

	return nil
}

// UpdateMetadata updates API key metadata.
func (s *Service) UpdateMetadata(ctx context.Context, id uuid.UUID, metadata Metadata) error {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate metadata
	if err := s.validateMetadata(metadata); err != nil {
		return err
	}

	// Preserve usage counters
	metadata.UsageCount = key.Metadata.UsageCount
	metadata.ErrorCount = key.Metadata.ErrorCount
	metadata.LastIP = key.Metadata.LastIP
	metadata.UserAgent = key.Metadata.UserAgent

	key.Metadata = metadata

	if err := s.repo.Update(ctx, key); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}

// AddIPToWhitelist adds an IP address to the whitelist.
func (s *Service) AddIPToWhitelist(ctx context.Context, id uuid.UUID, ip string) error {
	if err := s.validateIP(ip); err != nil {
		return err
	}

	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if IP already in whitelist
	for _, existingIP := range key.Metadata.IPWhitelist {
		if existingIP == ip {
			return nil // Already exists
		}
	}

	key.Metadata.IPWhitelist = append(key.Metadata.IPWhitelist, ip)

	if err := s.repo.Update(ctx, key); err != nil {
		return fmt.Errorf("failed to add IP to whitelist: %w", err)
	}

	return nil
}

// RemoveIPFromWhitelist removes an IP address from the whitelist.
func (s *Service) RemoveIPFromWhitelist(ctx context.Context, id uuid.UUID, ip string) error {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Remove IP from whitelist
	newWhitelist := make([]string, 0)
	for _, existingIP := range key.Metadata.IPWhitelist {
		if existingIP != ip {
			newWhitelist = append(newWhitelist, existingIP)
		}
	}

	key.Metadata.IPWhitelist = newWhitelist

	if err := s.repo.Update(ctx, key); err != nil {
		return fmt.Errorf("failed to remove IP from whitelist: %w", err)
	}

	return nil
}

// CheckPermission checks if a key has a specific permission.
func (s *Service) CheckPermission(key *APIKey, permission string) error {
	if !key.HasPermission(permission) {
		return NewPermissionDeniedError(permission, key.Permissions)
	}

	return nil
}

// MarkExpired marks expired API keys as expired.
func (s *Service) MarkExpired(ctx context.Context, limit int) (int, error) {
	expiredKeys, err := s.repo.ListExpired(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("failed to list expired keys: %w", err)
	}

	count := 0
	for _, key := range expiredKeys {
		if key.Status == StatusActive {
			key.MarkExpired()
			if err := s.repo.Update(ctx, key); err != nil {
				// Log error but continue
				continue
			}
			count++
		}
	}

	return count, nil
}

// ListForTenant lists all API keys for a tenant.
func (s *Service) ListForTenant(ctx context.Context, tenantID uuid.UUID, activeOnly bool) ([]*APIKey, error) {
	var keys []*APIKey
	var err error

	if activeOnly {
		keys, err = s.repo.FindActiveByTenantID(ctx, tenantID)
	} else {
		keys, err = s.repo.FindByTenantID(ctx, tenantID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	// Sanitize all keys
	sanitized := make([]*APIKey, len(keys))
	for i, key := range keys {
		sanitized[i] = key.Sanitize()
	}

	return sanitized, nil
}

// GetByID gets an API key by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	key, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return key.Sanitize(), nil
}

// Delete deletes an API key.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if key exists
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}

// CountForTenant counts API keys for a tenant.
func (s *Service) CountForTenant(ctx context.Context, tenantID uuid.UUID, activeOnly bool) (int64, error) {
	var count int64
	var err error

	if activeOnly {
		count, err = s.repo.CountActiveByTenantID(ctx, tenantID)
	} else {
		count, err = s.repo.CountByTenantID(ctx, tenantID)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to count API keys: %w", err)
	}

	return count, nil
}

// validateName validates the API key name.
func (s *Service) validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyName
	}

	if len(name) < 3 {
		return ErrNameTooShort
	}

	if len(name) > 100 {
		return ErrNameTooLong
	}

	return nil
}

// validatePermissions validates the permissions list.
func (s *Service) validatePermissions(permissions []string) error {
	if len(permissions) == 0 {
		return ErrEmptyPermissions
	}

	validPermissions := map[string]bool{
		PermissionAll:            true,
		PermissionReadTenants:    true,
		PermissionWriteTenants:   true,
		PermissionReadCalls:      true,
		PermissionWriteCalls:     true,
		PermissionReadAnalytics:  true,
		PermissionManageAPIKeys:  true,
		PermissionManageWebhooks: true,
	}

	for _, perm := range permissions {
		if !validPermissions[perm] {
			return ErrInvalidPermissions
		}
	}

	return nil
}

// validateMetadata validates API key metadata.
func (s *Service) validateMetadata(metadata Metadata) error {
	// Validate IP whitelist
	for _, ip := range metadata.IPWhitelist {
		if err := s.validateIP(ip); err != nil {
			return err
		}
	}

	// Validate rate limit
	if metadata.RateLimit < 0 || metadata.RateLimit > 100000 {
		return ErrInvalidRateLimit
	}

	return nil
}

// validateIP validates an IP address format.
func (s *Service) validateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return ErrInvalidIPFormat
	}

	return nil
}
