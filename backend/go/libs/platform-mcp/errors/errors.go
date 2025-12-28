package errors

import "errors"

// Error codes standardized for MCP interactions.
type Code string

const (
	ErrUnsupportedVersion Code = "unsupported_version"
	ErrMissingTenant      Code = "missing_tenant"
	ErrUnauthorized       Code = "unauthorized"
	ErrForbidden          Code = "forbidden"
	ErrRateLimited        Code = "rate_limited"
	ErrInvalidRequest     Code = "invalid_request"
	ErrInvalidSchema      Code = "invalid_schema"
	ErrToolNotFound       Code = "tool_not_found"
	ErrPolicyDenied       Code = "policy_denied"
	ErrCancelled          Code = "cancelled"
	ErrInternal           Code = "internal"
)

// Error wraps a code and message for propagation across transports.
type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string { return e.Message }

// Helpers to build typed errors.
func New(code Code, msg string) Error {
	return Error{Code: code, Message: msg}
}

// Sentinel errors for quick checks.
var (
	ErrUnsupportedVersionSentinel = errors.New(string(ErrUnsupportedVersion))
	ErrMissingTenantSentinel      = errors.New(string(ErrMissingTenant))
	ErrPolicyDeniedSentinel       = errors.New(string(ErrPolicyDenied))
)
