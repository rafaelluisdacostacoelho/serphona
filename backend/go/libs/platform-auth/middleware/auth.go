package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	autherrors "github.com/serphona/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/serphona/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/serphona/serphona/backend/go/libs/platform-auth/types"
)

// RequireAuth é um middleware que valida JWT e injeta claims no contexto
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extrai token do header Authorization
		authHeader := c.GetHeader("Authorization")

		// Valida token
		claims, err := authjwt.ValidateTokenFromHeader(authHeader)
		if err != nil {
			// Determina código de status apropriado
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

		// Verifica se o usuário está ativo
		// (esta validação pode ser feita aqui ou no auth-gateway)

		// Injeta claims no contexto
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

// RequireRole é um middleware que requer uma role específica
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtém claims do contexto (injetado por RequireAuth)
		claims, err := GetClaimsFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
				"code":  autherrors.CodeUnauthorized,
			})
			c.Abort()
			return
		}

		// Verifica role
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

// RequireAdmin é um middleware que requer role admin ou superadmin
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

// RequireSuperAdmin é um middleware que requer role superadmin
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

// GetClaimsFromContext extrai as claims do contexto da request
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

// GetUserIDFromContext extrai o userID do contexto
func GetUserIDFromContext(c *gin.Context) (string, error) {
	claims, err := GetClaimsFromContext(c)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// GetTenantIDFromContext extrai o tenantID do contexto
func GetTenantIDFromContext(c *gin.Context) (string, error) {
	claims, err := GetClaimsFromContext(c)
	if err != nil {
		return "", err
	}
	return claims.TenantID, nil
}
