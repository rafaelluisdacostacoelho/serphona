package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
)

type authMappedError struct {
	status  int
	code    string
	message string
}

var (
	internalErrorMapped   = authMappedError{status: http.StatusInternalServerError, code: "internal_error", message: "Internal server error"}
	requestTooLargeMapped = authMappedError{status: http.StatusRequestEntityTooLarge, code: "request_too_large", message: "Request body too large"}
	badRequestMapped      = authMappedError{status: http.StatusBadRequest, code: "bad_request", message: "Bad request"}
)

func mapAuthError(err error) authMappedError {
	mapped := authMappedError{
		status:  http.StatusUnauthorized,
		code:    autherrors.CodeUnauthorized,
		message: "Unauthorized",
	}

	switch {
	case errors.Is(err, autherrors.ErrAuthConfigMissing), errors.Is(err, autherrors.ErrSecretNotConfigured):
		mapped.status = http.StatusInternalServerError
		mapped.code = autherrors.CodeAuthConfigMissing
		mapped.message = "Auth configuration missing"
	case errors.Is(err, autherrors.ErrMissingToken):
		mapped.code = autherrors.CodeMissingToken
		mapped.message = "Missing authentication token"
	case errors.Is(err, autherrors.ErrInvalidToken), errors.Is(err, autherrors.ErrInvalidIssuer), errors.Is(err, autherrors.ErrInvalidAudience), errors.Is(err, autherrors.ErrInvalidAlgorithm):
		mapped.code = autherrors.CodeInvalidToken
		mapped.message = "Invalid authentication token"
	case errors.Is(err, autherrors.ErrInvalidKeyID):
		mapped.code = autherrors.CodeInvalidKeyID
		mapped.message = "Invalid authentication token"
	case errors.Is(err, autherrors.ErrJWKSFetchFailed):
		mapped.code = autherrors.CodeJWKSFetchFailed
		mapped.message = "Unable to validate authentication token"
	case errors.Is(err, autherrors.ErrTokenExpired):
		mapped.code = autherrors.CodeTokenExpired
		mapped.message = "Authentication token has expired"
	case errors.Is(err, autherrors.ErrTokenTooLarge):
		mapped.code = autherrors.CodeTokenTooLarge
		mapped.message = "Authentication token too large"
	case errors.Is(err, autherrors.ErrInsufficientPermissions):
		mapped.status = http.StatusForbidden
		mapped.code = autherrors.CodeInsufficientPermissions
		mapped.message = "Insufficient permissions"
	case errors.Is(err, autherrors.ErrInvalidRole):
		mapped.status = http.StatusForbidden
		mapped.code = autherrors.CodeInvalidRole
		mapped.message = "Invalid role"
	}

	return mapped
}

func needsSecret(cfg authjwt.ValidationConfig) bool {
	if cfg.JWKSURL == "" {
		return true
	}

	for _, alg := range cfg.AllowedAlgs {
		if strings.HasPrefix(strings.ToUpper(alg), "HS") {
			return true
		}
	}

	return false
}

func extractOrGenerateRequestID(headerVal string) string {
	if headerVal != "" {
		return headerVal
	}

	return uuid.NewString()
}

const requestIDHeader = "X-Request-Id"

func errorPayload(mapped authMappedError) map[string]string {
	return map[string]string{
		"error": mapped.message,
		"code":  mapped.code,
	}
}
