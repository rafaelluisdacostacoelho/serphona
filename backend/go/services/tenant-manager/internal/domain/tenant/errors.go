// Package tenant contains the tenant domain model and business logic.
package tenant

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Domain errors
var (
	// ErrNotFound is returned when a tenant is not found
	ErrNotFound = errors.New("tenant not found")

	// ErrEmailAlreadyExists is returned when email is already taken
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrSlugAlreadyExists is returned when slug is already taken
	ErrSlugAlreadyExists = errors.New("slug already exists")

	// ErrTenantNotActive is returned when tenant is not active
	ErrTenantNotActive = errors.New("tenant is not active")

	// ErrTenantDeleted is returned when tenant is deleted
	ErrTenantDeleted = errors.New("tenant is deleted")

	// ErrTenantAlreadyActive is returned when trying to activate an active tenant
	ErrTenantAlreadyActive = errors.New("tenant is already active")

	// ErrTenantAlreadySuspended is returned when trying to suspend a suspended tenant
	ErrTenantAlreadySuspended = errors.New("tenant is already suspended")

	// ErrTenantAlreadyDeleted is returned when trying to delete a deleted tenant
	ErrTenantAlreadyDeleted = errors.New("tenant is already deleted")

	// ErrCallsNotAllowed is returned when tenant cannot make calls
	ErrCallsNotAllowed = errors.New("calls not allowed for this tenant")

	// ErrSamePlan is returned when trying to change to the same plan
	ErrSamePlan = errors.New("tenant is already on this plan")
)

// Validation errors
var (
	// ErrEmptyName is returned when name is empty
	ErrEmptyName = errors.New("name cannot be empty")

	// ErrNameTooShort is returned when name is too short
	ErrNameTooShort = errors.New("name must be at least 2 characters")

	// ErrNameTooLong is returned when name is too long
	ErrNameTooLong = errors.New("name must not exceed 100 characters")

	// ErrEmptyEmail is returned when email is empty
	ErrEmptyEmail = errors.New("email cannot be empty")

	// ErrInvalidEmail is returned when email format is invalid
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidPhone is returned when phone format is invalid
	ErrInvalidPhone = errors.New("invalid phone number format")

	// ErrInvalidSlug is returned when slug format is invalid
	ErrInvalidSlug = errors.New("invalid slug format: must contain only lowercase letters, numbers, and hyphens")

	// ErrInvalidPlan is returned when plan is invalid
	ErrInvalidPlan = errors.New("invalid plan: must be starter, professional, or enterprise")

	// ErrInvalidSettings is returned when settings are invalid
	ErrInvalidSettings = errors.New("invalid tenant settings")

	// ErrInvalidStatus is returned when status is invalid
	ErrInvalidStatus = errors.New("invalid tenant status")
)

// TenantNotFoundError represents a specific tenant not found error.
type TenantNotFoundError struct {
	ID uuid.UUID
}

// Error implements the error interface.
func (e *TenantNotFoundError) Error() string {
	return fmt.Sprintf("tenant not found: %s", e.ID)
}

// Is checks if the error is ErrNotFound.
func (e *TenantNotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// NewTenantNotFoundError creates a new TenantNotFoundError.
func NewTenantNotFoundError(id uuid.UUID) error {
	return &TenantNotFoundError{ID: id}
}

// TenantEmailExistsError represents an email already exists error.
type TenantEmailExistsError struct {
	Email string
}

// Error implements the error interface.
func (e *TenantEmailExistsError) Error() string {
	return fmt.Sprintf("tenant with email %s already exists", e.Email)
}

// Is checks if the error is ErrEmailAlreadyExists.
func (e *TenantEmailExistsError) Is(target error) bool {
	return target == ErrEmailAlreadyExists
}

// NewTenantEmailExistsError creates a new TenantEmailExistsError.
func NewTenantEmailExistsError(email string) error {
	return &TenantEmailExistsError{Email: email}
}

// TenantSlugExistsError represents a slug already exists error.
type TenantSlugExistsError struct {
	Slug string
}

// Error implements the error interface.
func (e *TenantSlugExistsError) Error() string {
	return fmt.Sprintf("tenant with slug %s already exists", e.Slug)
}

// Is checks if the error is ErrSlugAlreadyExists.
func (e *TenantSlugExistsError) Is(target error) bool {
	return target == ErrSlugAlreadyExists
}

// NewTenantSlugExistsError creates a new TenantSlugExistsError.
func NewTenantSlugExistsError(slug string) error {
	return &TenantSlugExistsError{Slug: slug}
}

// ValidationError represents a validation error with field information.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// ValidationErrors represents multiple validation errors.
type ValidationErrors struct {
	Errors []ValidationError
}

// Error implements the error interface.
func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation errors"
	}

	msg := fmt.Sprintf("validation errors (%d):", len(e.Errors))
	for _, err := range e.Errors {
		msg += fmt.Sprintf("\n  - %s: %s", err.Field, err.Message)
	}

	return msg
}

// Add adds a validation error.
func (e *ValidationErrors) Add(field, message string) {
	e.Errors = append(e.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// HasErrors returns true if there are validation errors.
func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// NewValidationErrors creates a new ValidationErrors.
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]ValidationError, 0),
	}
}
