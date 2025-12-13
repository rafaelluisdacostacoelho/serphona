// Package apikey contains the API key domain model and business logic.
package apikey

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Domain errors
var (
	// ErrNotFound is returned when an API key is not found
	ErrNotFound = errors.New("api key not found")

	// ErrKeyExpired is returned when the key has expired
	ErrKeyExpired = errors.New("api key has expired")

	// ErrKeyRevoked is returned when the key has been revoked
	ErrKeyRevoked = errors.New("api key has been revoked")

	// ErrKeyInactive is returned when the key is not active
	ErrKeyInactive = errors.New("api key is not active")

	// ErrInvalidKey is returned when the key format is invalid
	ErrInvalidKey = errors.New("invalid api key format")

	// ErrInvalidKeyHash is returned when key hash verification fails
	ErrInvalidKeyHash = errors.New("invalid api key: authentication failed")

	// ErrNameAlreadyExists is returned when an API key name already exists for the tenant
	ErrNameAlreadyExists = errors.New("api key name already exists")

	// ErrInsufficientPermissions is returned when key lacks required permissions
	ErrInsufficientPermissions = errors.New("insufficient permissions")

	// ErrIPNotWhitelisted is returned when request IP is not in whitelist
	ErrIPNotWhitelisted = errors.New("ip address not whitelisted")

	// ErrQuotaExceeded is returned when tenant API key quota is exceeded
	ErrQuotaExceeded = errors.New("api key quota exceeded")

	// ErrRateLimitExceeded is returned when rate limit is exceeded
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// Validation errors
var (
	// ErrEmptyName is returned when name is empty
	ErrEmptyName = errors.New("api key name cannot be empty")

	// ErrNameTooShort is returned when name is too short
	ErrNameTooShort = errors.New("api key name must be at least 3 characters")

	// ErrNameTooLong is returned when name is too long
	ErrNameTooLong = errors.New("api key name must not exceed 100 characters")

	// ErrInvalidPermissions is returned when permissions are invalid
	ErrInvalidPermissions = errors.New("invalid permissions")

	// ErrEmptyPermissions is returned when no permissions are specified
	ErrEmptyPermissions = errors.New("at least one permission is required")

	// ErrInvalidExpiry is returned when expiry date is invalid
	ErrInvalidExpiry = errors.New("invalid expiry date")

	// ErrInvalidIPFormat is returned when IP address format is invalid
	ErrInvalidIPFormat = errors.New("invalid IP address format")

	// ErrInvalidRateLimit is returned when rate limit value is invalid
	ErrInvalidRateLimit = errors.New("invalid rate limit value")
)

// APIKeyNotFoundError represents a specific API key not found error.
type APIKeyNotFoundError struct {
	ID uuid.UUID
}

// Error implements the error interface.
func (e *APIKeyNotFoundError) Error() string {
	return fmt.Sprintf("api key not found: %s", e.ID)
}

// Is checks if the error is ErrNotFound.
func (e *APIKeyNotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// NewAPIKeyNotFoundError creates a new APIKeyNotFoundError.
func NewAPIKeyNotFoundError(id uuid.UUID) error {
	return &APIKeyNotFoundError{ID: id}
}

// APIKeyExpiredError represents an expired API key error.
type APIKeyExpiredError struct {
	ID        uuid.UUID
	ExpiredAt string
}

// Error implements the error interface.
func (e *APIKeyExpiredError) Error() string {
	return fmt.Sprintf("api key %s expired at %s", e.ID, e.ExpiredAt)
}

// Is checks if the error is ErrKeyExpired.
func (e *APIKeyExpiredError) Is(target error) bool {
	return target == ErrKeyExpired
}

// NewAPIKeyExpiredError creates a new APIKeyExpiredError.
func NewAPIKeyExpiredError(id uuid.UUID, expiredAt string) error {
	return &APIKeyExpiredError{
		ID:        id,
		ExpiredAt: expiredAt,
	}
}

// APIKeyRevokedError represents a revoked API key error.
type APIKeyRevokedError struct {
	ID        uuid.UUID
	RevokedAt string
	Reason    string
}

// Error implements the error interface.
func (e *APIKeyRevokedError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("api key %s was revoked at %s: %s", e.ID, e.RevokedAt, e.Reason)
	}
	return fmt.Sprintf("api key %s was revoked at %s", e.ID, e.RevokedAt)
}

// Is checks if the error is ErrKeyRevoked.
func (e *APIKeyRevokedError) Is(target error) bool {
	return target == ErrKeyRevoked
}

// NewAPIKeyRevokedError creates a new APIKeyRevokedError.
func NewAPIKeyRevokedError(id uuid.UUID, revokedAt, reason string) error {
	return &APIKeyRevokedError{
		ID:        id,
		RevokedAt: revokedAt,
		Reason:    reason,
	}
}

// PermissionDeniedError represents a permission denied error.
type PermissionDeniedError struct {
	Required string
	Has      []string
}

// Error implements the error interface.
func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf("permission denied: requires '%s', has %v", e.Required, e.Has)
}

// Is checks if the error is ErrInsufficientPermissions.
func (e *PermissionDeniedError) Is(target error) bool {
	return target == ErrInsufficientPermissions
}

// NewPermissionDeniedError creates a new PermissionDeniedError.
func NewPermissionDeniedError(required string, has []string) error {
	return &PermissionDeniedError{
		Required: required,
		Has:      has,
	}
}

// IPNotWhitelistedError represents an IP not whitelisted error.
type IPNotWhitelistedError struct {
	IP        string
	Whitelist []string
}

// Error implements the error interface.
func (e *IPNotWhitelistedError) Error() string {
	return fmt.Sprintf("ip %s not in whitelist %v", e.IP, e.Whitelist)
}

// Is checks if the error is ErrIPNotWhitelisted.
func (e *IPNotWhitelistedError) Is(target error) bool {
	return target == ErrIPNotWhitelisted
}

// NewIPNotWhitelistedError creates a new IPNotWhitelistedError.
func NewIPNotWhitelistedError(ip string, whitelist []string) error {
	return &IPNotWhitelistedError{
		IP:        ip,
		Whitelist: whitelist,
	}
}

// QuotaExceededError represents a quota exceeded error.
type QuotaExceededError struct {
	Current int64
	Limit   int64
}

// Error implements the error interface.
func (e *QuotaExceededError) Error() string {
	return fmt.Sprintf("api key quota exceeded: %d/%d", e.Current, e.Limit)
}

// Is checks if the error is ErrQuotaExceeded.
func (e *QuotaExceededError) Is(target error) bool {
	return target == ErrQuotaExceeded
}

// NewQuotaExceededError creates a new QuotaExceededError.
func NewQuotaExceededError(current, limit int64) error {
	return &QuotaExceededError{
		Current: current,
		Limit:   limit,
	}
}
