package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

// RequireAuth is a Gin middleware that validates a JWT and injects claims into the request context.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		reqID := extractOrGenerateRequestID(c.GetHeader(requestIDHeader))
		c.Writer.Header().Set(requestIDHeader, reqID)
		ctx := WithRequestID(c.Request.Context(), reqID)
		safeFields := SafeRequestFieldsFromGin(c)
		ctx = WithSafeRequestFields(ctx, safeFields)
		ctx, span := startAuthSpan(ctx, "platform-auth.require-auth", reqID)
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		cfg := authjwt.GetValidationConfig()
		if needsSecret(cfg) {
			if err := authjwt.EnsureSecretLoaded(); err != nil {
				mapped := mapAuthError(err)
				tenant := tenantFromHeaders(c.Request.Header)
				recordAuthError("gin", mapped, tenant, start)
				recordSpanError(span, mapped, err)
				c.JSON(mapped.status, errorPayload(mapped))
				c.Abort()
				return
			}
		} else {
			// Ensure the error from EnsureSecretLoaded is always handled.
			if err := authjwt.EnsureSecretLoaded(); err != nil {
				// Add logging to verify the error returned by EnsureSecretLoaded.
				log.Printf("EnsureSecretLoaded error: %v", err)
				mapped := mapAuthError(err)
				tenant := tenantFromHeaders(c.Request.Header)
				recordAuthError("gin", mapped, tenant, start)
				recordSpanError(span, mapped, err)
				c.JSON(mapped.status, errorPayload(mapped))
				c.Abort()
				return
			}
		}
		authHeader := c.GetHeader("Authorization")

		claims, err := ValidateTokenFromHeader(authHeader)
		if err != nil {
			mapped := mapAuthError(err)
			tenant := tenantFromHeaders(c.Request.Header)
			recordAuthError("gin", mapped, tenant, start)
			recordSpanError(span, mapped, err)
			c.JSON(mapped.status, errorPayload(mapped))
			c.Abort()
			return
		}

		annotateSpanWithClaims(span, claims)
		c.Request = c.Request.WithContext(WithClaims(c.Request.Context(), claims))
		// Claims are added to the context for downstream handlers.
		c.Set("claims", claims)
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Set("role", claims.Role)
		c.Set("tenantID", claims.TenantID)
		c.Set("sessionID", claims.SessionID)
		c.Set("requestID", reqID)

		recordAuthSuccess("gin", claims.TenantID, start)
		recordSpanSuccess(span)

		c.Next()
	}
}

// RequireRole enforces a specific role on the request context.
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := GetClaimsFromContext(c)
		if err != nil {
			mapped := mapAuthError(autherrors.ErrUnauthorized)
			c.JSON(mapped.status, errorPayload(mapped))
			c.Abort()
			return
		}

		if !claims.HasRole(requiredRole) {
			c.JSON(http.StatusForbidden, errorPayload(mapAuthError(autherrors.ErrInsufficientPermissions)))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin allows only admin or superadmin roles.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := GetClaimsFromContext(c)
		if err != nil {
			mapped := mapAuthError(autherrors.ErrUnauthorized)
			c.JSON(mapped.status, errorPayload(mapped))
			c.Abort()
			return
		}

		if !claims.IsAdmin() {
			mapped := mapAuthError(autherrors.ErrInsufficientPermissions)
			mapped.message = "Admin access required"
			c.JSON(mapped.status, errorPayload(mapped))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireSuperAdmin allows only the superadmin role.
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := GetClaimsFromContext(c)
		if err != nil {
			mapped := mapAuthError(autherrors.ErrUnauthorized)
			c.JSON(mapped.status, errorPayload(mapped))
			c.Abort()
			return
		}

		if !claims.IsSuperAdmin() {
			mapped := mapAuthError(autherrors.ErrInsufficientPermissions)
			mapped.message = "Superadmin access required"
			c.JSON(mapped.status, errorPayload(mapped))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetClaimsFromContext extracts claims from the request context.
func GetClaimsFromContext(c *gin.Context) (*types.Claims, error) {
	claimsValue, exists := c.Get("claims")
	if !exists {
		return nil, autherrors.ErrUnauthorized
	}

	claims, ok := claimsValue.(*types.Claims)
	if !ok {
		return nil, autherrors.ErrUnauthorized
	}

	return claims, nil
}

// GetUserIDFromContext extracts the userID from context.
func GetUserIDFromContext(c *gin.Context) (string, error) {
	claims, err := GetClaimsFromContext(c)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// GetTenantIDFromContext extracts the tenantID from context.
func GetTenantIDFromContext(c *gin.Context) (string, error) {
	claims, err := GetClaimsFromContext(c)
	if err != nil {
		return "", err
	}
	return claims.TenantID, nil
}

// RequireScopes enforces that the user has all of the specified scopes.
func RequireScopes(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := GetClaimsFromContext(c)
		if err != nil {
			mapped := mapAuthError(autherrors.ErrUnauthorized)
			c.JSON(mapped.status, errorPayload(mapped))
			c.Abort()
			return
		}

		if !claims.HasAllScopes(scopes...) {
			c.JSON(http.StatusForbidden, errorPayload(mapAuthError(autherrors.ErrInsufficientPermissions)))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyScope enforces that the user has at least one of the specified scopes.
func RequireAnyScope(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := GetClaimsFromContext(c)
		if err != nil {
			mapped := mapAuthError(autherrors.ErrUnauthorized)
			c.JSON(mapped.status, errorPayload(mapped))
			c.Abort()
			return
		}

		if !claims.HasAnyScope(scopes...) {
			c.JSON(http.StatusForbidden, errorPayload(mapAuthError(autherrors.ErrInsufficientPermissions)))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllScopes is an alias for RequireScopes for clarity.
func RequireAllScopes(scopes ...string) gin.HandlerFunc {
	return RequireScopes(scopes...)
}

// Ensure the validateTokenFromHeader variable is exported for testing purposes.
var ValidateTokenFromHeader = authjwt.ValidateTokenFromHeader
