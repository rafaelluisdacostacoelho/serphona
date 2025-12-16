package errors

import "errors"

// Authentication errors used across the library.
var (
	ErrUnauthorized            = errors.New("unauthorized")
	ErrInvalidToken            = errors.New("invalid token")
	ErrTokenExpired            = errors.New("token expired")
	ErrMissingToken            = errors.New("missing token")
	ErrInsufficientPermissions = errors.New("insufficient permissions")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrUserNotFound            = errors.New("user not found")
	ErrUserInactive            = errors.New("user is inactive")
	ErrUserNotVerified         = errors.New("user email not verified")
	ErrInvalidRole             = errors.New("invalid role")
)

// AuthError represents an authentication error with code and message.
type AuthError struct {
	Code    string
	Message string
	Err     error
}

// Error implements the error interface.
func (e *AuthError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap allows errors.Is and errors.As to work with AuthError.
func (e *AuthError) Unwrap() error {
	return e.Err
}

// NewAuthError creates a new authentication error instance.
func NewAuthError(code, message string, err error) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Standardized error codes.
const (
	CodeUnauthorized            = "UNAUTHORIZED"
	CodeInvalidToken            = "INVALID_TOKEN"
	CodeTokenExpired            = "TOKEN_EXPIRED"
	CodeMissingToken            = "MISSING_TOKEN"
	CodeInsufficientPermissions = "INSUFFICIENT_PERMISSIONS"
	CodeInvalidCredentials      = "INVALID_CREDENTIALS"
	CodeUserNotFound            = "USER_NOT_FOUND"
	CodeUserInactive            = "USER_INACTIVE"
	CodeUserNotVerified         = "USER_NOT_VERIFIED"
	CodeInvalidRole             = "INVALID_ROLE"
)
