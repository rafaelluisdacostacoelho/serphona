package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	autherrors "github.com/serphona/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/serphona/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/serphona/serphona/backend/go/libs/platform-auth/types"
)

// RequireAuth is a Gin middleware that validates a JWT and injects claims into the request context.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		claims, err := authjwt.ValidateTokenFromHeader(authHeader)
		if err != nil {
			statusCode := http.StatusUnauthorized
			errorCode := autherrors.CodeUnauthorized
			errorMessage := "Unauthorized"

			switch err {
			case autherrors.ErrMissingToken:
				errorCode = autherrors.CodeMissingToken
				errorMessage = "Missing authentication token"
			case autherrors.ErrInvalidToken:
				errorCode = autherrors.CodeInvalidToken
				errorMessage = "Invalid authentication token"
			case autherrors.ErrTokenExpired:
				errorCode = autherrors.CodeTokenExpired
				errorMessage = "Authentication token has expired"
			}

			c.JSON(statusCode, gin.H{
				"error": errorMessage,
				"code":  errorCode,
			})
			c.Abort()
			return
		}

		// Claims are added to the context for downstream handlers.
		c.Set("claims", claims)
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Set("role", claims.Role)
		c.Set("tenantID", claims.TenantID)
		c.Set("sessionID", claims.SessionID)

		c.Next()
	}
}

// RequireRole enforces a specific role on the request context.
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := GetClaimsFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
				"code":  autherrors.CodeUnauthorized,
			})
			c.Abort()
			return
		}

		if !claims.HasRole(requiredRole) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  autherrors.CodeInsufficientPermissions,
			})
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
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
				"code":  autherrors.CodeUnauthorized,
			})
			c.Abort()
			return
		}

		if !claims.IsAdmin() {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Admin access required",
				"code":  autherrors.CodeInsufficientPermissions,
			})
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
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
				"code":  autherrors.CodeUnauthorized,
			})
			c.Abort()
			return
		}

		if !claims.IsSuperAdmin() {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Superadmin access required",
				"code":  autherrors.CodeInsufficientPermissions,
			})
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
