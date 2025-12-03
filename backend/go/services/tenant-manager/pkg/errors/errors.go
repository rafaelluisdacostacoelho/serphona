// Package errors provides custom error types for the application.
package errors

import "fmt"

// ErrorCode represents application error codes.
type ErrorCode string

const (
	ErrInternal       ErrorCode = "internal_error"
	ErrNotFound       ErrorCode = "not_found"
	ErrConflict       ErrorCode = "conflict"
	ErrValidation     ErrorCode = "validation_error"
	ErrUnauthorized   ErrorCode = "unauthorized"
	ErrForbidden      ErrorCode = "forbidden"
	ErrBadRequest     ErrorCode = "bad_request"
	ErrTooManyReqs    ErrorCode = "too_many_requests"
	ErrServiceUnavail ErrorCode = "service_unavailable"
)

// AppError represents an application error.
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error.
func NewAppError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// NewInternalError creates a new internal error.
func NewInternalError(message string) *AppError {
	return &AppError{
		Code:    ErrInternal,
		Message: message,
	}
}

// NewNotFoundError creates a new not found error.
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Code:    ErrNotFound,
		Message: message,
	}
}

// NewConflictError creates a new conflict error.
func NewConflictError(message string) *AppError {
	return &AppError{
		Code:    ErrConflict,
		Message: message,
	}
}

// NewValidationError creates a new validation error.
func NewValidationError(message string) *AppError {
	return &AppError{
		Code:    ErrValidation,
		Message: message,
	}
}

// NewUnauthorizedError creates a new unauthorized error.
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:    ErrUnauthorized,
		Message: message,
	}
}

// NewForbiddenError creates a new forbidden error.
func NewForbiddenError(message string) *AppError {
	return &AppError{
		Code:    ErrForbidden,
		Message: message,
	}
}

// NewBadRequestError creates a new bad request error.
func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:    ErrBadRequest,
		Message: message,
	}
}
