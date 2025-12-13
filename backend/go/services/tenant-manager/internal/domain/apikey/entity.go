// Package apikey contains the API key domain model and business logic.
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// PrefixLength is the length of the key prefix for identification
	PrefixLength = 8
	// KeyLength is the total length of the generated key
	KeyLength = 32
	// DefaultExpiryDays is the default expiry period in days
	DefaultExpiryDays = 365
)

// APIKey represents an API key for tenant authentication.
type APIKey struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`        // Human-readable name
	KeyPrefix   string     `json:"key_prefix"`  // First 8 chars for identification (e.g., "sk_prod_")
	KeyHash     string     `json:"-"`           // SHA-256 hash of the full key (never exposed)
	Permissions []string   `json:"permissions"` // List of permissions
	Status      Status     `json:"status"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   uuid.UUID  `json:"created_by"` // User who created the key
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	RevokedBy   *uuid.UUID `json:"revoked_by,omitempty"`
	Metadata    Metadata   `json:"metadata"`
}

// Status represents the API key status.
type Status string

const (
	StatusActive  Status = "active"
	StatusRevoked Status = "revoked"
	StatusExpired Status = "expired"
)

// Metadata contains additional API key information.
type Metadata struct {
	Description string            `json:"description,omitempty"`
	IPWhitelist []string          `json:"ip_whitelist,omitempty"`
	RateLimit   int               `json:"rate_limit,omitempty"`  // Requests per minute
	Environment string            `json:"environment,omitempty"` // e.g., "production", "development"
	Custom      map[string]string `json:"custom,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	LastIP      string            `json:"last_ip,omitempty"`
	UsageCount  int64             `json:"usage_count"`
	ErrorCount  int64             `json:"error_count"`
}

// Permission constants
const (
	PermissionAll            = "*"
	PermissionReadTenants    = "tenants:read"
	PermissionWriteTenants   = "tenants:write"
	PermissionReadCalls      = "calls:read"
	PermissionWriteCalls     = "calls:write"
	PermissionReadAnalytics  = "analytics:read"
	PermissionManageAPIKeys  = "apikeys:manage"
	PermissionManageWebhooks = "webhooks:manage"
)

// NewAPIKey creates a new API key with a generated key.
func NewAPIKey(tenantID uuid.UUID, name string, createdBy uuid.UUID, permissions []string) (*APIKey, string, error) {
	now := time.Now().UTC()
	expiresAt := now.AddDate(0, 0, DefaultExpiryDays)

	// Generate random key
	rawKey, err := generateRandomKey(KeyLength)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Create prefix (first 8 chars for identification)
	prefix := rawKey[:PrefixLength]

	// Hash the key for storage
	keyHash := hashKey(rawKey)

	apiKey := &APIKey{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		KeyPrefix:   prefix,
		KeyHash:     keyHash,
		Permissions: permissions,
		Status:      StatusActive,
		ExpiresAt:   &expiresAt,
		CreatedAt:   now,
		CreatedBy:   createdBy,
		Metadata: Metadata{
			UsageCount: 0,
			ErrorCount: 0,
		},
	}

	// Return both the API key entity and the full raw key (only time it's available)
	return apiKey, rawKey, nil
}

// IsActive returns true if the key is active and not expired.
func (k *APIKey) IsActive() bool {
	if k.Status != StatusActive {
		return false
	}

	if k.ExpiresAt != nil && time.Now().UTC().After(*k.ExpiresAt) {
		return false
	}

	return true
}

// IsExpired returns true if the key has expired.
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}

	return time.Now().UTC().After(*k.ExpiresAt)
}

// Revoke revokes the API key.
func (k *APIKey) Revoke(revokedBy uuid.UUID) {
	now := time.Now().UTC()
	k.Status = StatusRevoked
	k.RevokedAt = &now
	k.RevokedBy = &revokedBy
}

// MarkExpired marks the key as expired.
func (k *APIKey) MarkExpired() {
	k.Status = StatusExpired
}

// RecordUsage records a successful usage of the key.
func (k *APIKey) RecordUsage(ipAddress, userAgent string) {
	now := time.Now().UTC()
	k.LastUsedAt = &now
	k.Metadata.LastIP = ipAddress
	k.Metadata.UserAgent = userAgent
	k.Metadata.UsageCount++
}

// RecordError records an error when using the key.
func (k *APIKey) RecordError() {
	k.Metadata.ErrorCount++
}

// HasPermission checks if the key has a specific permission.
func (k *APIKey) HasPermission(permission string) bool {
	// Check for wildcard permission
	for _, p := range k.Permissions {
		if p == PermissionAll {
			return true
		}
		if p == permission {
			return true
		}
	}

	return false
}

// HasAnyPermission checks if the key has any of the specified permissions.
func (k *APIKey) HasAnyPermission(permissions ...string) bool {
	for _, permission := range permissions {
		if k.HasPermission(permission) {
			return true
		}
	}

	return false
}

// HasAllPermissions checks if the key has all specified permissions.
func (k *APIKey) HasAllPermissions(permissions ...string) bool {
	for _, permission := range permissions {
		if !k.HasPermission(permission) {
			return false
		}
	}

	return true
}

// IsAllowedFromIP checks if the key can be used from the given IP address.
func (k *APIKey) IsAllowedFromIP(ip string) bool {
	// If no whitelist, allow from any IP
	if len(k.Metadata.IPWhitelist) == 0 {
		return true
	}

	// Check if IP is in whitelist
	for _, allowedIP := range k.Metadata.IPWhitelist {
		if allowedIP == ip {
			return true
		}
	}

	return false
}

// VerifyKey verifies if the provided raw key matches this API key.
func (k *APIKey) VerifyKey(rawKey string) bool {
	return hashKey(rawKey) == k.KeyHash
}

// Sanitize returns a safe representation without sensitive data.
func (k *APIKey) Sanitize() *APIKey {
	sanitized := *k
	sanitized.KeyHash = "" // Never expose the hash
	return &sanitized
}

// generateRandomKey generates a cryptographically secure random key.
func generateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to base64 for readability
	key := base64.RawURLEncoding.EncodeToString(bytes)

	// Trim to desired length
	if len(key) > length {
		key = key[:length]
	}

	// Add prefix for identification
	return fmt.Sprintf("sk_%s", key), nil
}

// hashKey creates a SHA-256 hash of the key for secure storage.
func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}

// ValidateKeyFormat validates the format of an API key.
func ValidateKeyFormat(key string) bool {
	// Must start with "sk_"
	if len(key) < 3 || key[:3] != "sk_" {
		return false
	}

	// Must be at least a reasonable length
	if len(key) < 20 {
		return false
	}

	return true
}

// ParseKeyPrefix extracts the prefix from a full key.
func ParseKeyPrefix(rawKey string) string {
	if len(rawKey) < PrefixLength {
		return rawKey
	}

	return rawKey[:PrefixLength]
}
