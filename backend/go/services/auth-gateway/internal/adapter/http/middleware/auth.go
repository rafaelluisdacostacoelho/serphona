package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/observability"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/service/jwt"
)

// AuthMiddleware validates JWT tokens
type AuthMiddleware struct {
	jwtService jwt.JWTService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtService jwt.JWTService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
	}
}

// Authenticate validates the JWT token from the request
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Missing authorization header",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Check Bearer scheme
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid authorization header format",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := m.jwtService.ValidateAccessToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid or expired token",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid token claims",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		tenantID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid token claims",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("userID", userID)
		c.Set("email", claims.Email)
		c.Set("tenantID", tenantID)
		c.Set("role", claims.Role)

		if tenantID == uuid.Nil {
			observability.RecordTenantMissing(c.FullPath())
		}

		c.Next()
	}
}

// RequireRole checks if the user has the required role
func (m *AuthMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "Unauthorized",
				"code":    "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		role := userRole.(string)
		for _, requiredRole := range roles {
			if role == requiredRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"message": "Insufficient permissions",
			"code":    "FORBIDDEN",
		})
		c.Abort()
	}
}

// CORS middleware for handling cross-origin requests with allowlist and fail-closed defaults.
func CORS(allowedOrigins []string, allowCredentials bool) gin.HandlerFunc {
	originAllowed := func(origin string) bool {
		if origin == "" {
			return false
		}
		for _, o := range allowedOrigins {
			if strings.EqualFold(o, origin) {
				return true
			}
		}
		return false
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		// Allow internal/server-to-server calls (no Origin header), including health checks.
		if origin == "" {
			c.Next()
			return
		}
		allowed := originAllowed(origin)

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			if allowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		} else {
			// Fail closed: no wildcard; credentials not allowed.
			c.Writer.Header().Set("Access-Control-Allow-Origin", "")
			c.Writer.Header().Del("Access-Control-Allow-Credentials")
		}

		headers := "Content-Type, Content-Length, Accept-Encoding, Authorization, Accept, Origin, Cache-Control, X-Requested-With"
		c.Writer.Header().Set("Access-Control-Allow-Headers", headers)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == http.MethodOptions {
			if !allowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "origin not allowed", "code": "FORBIDDEN"})
			return
		}

		c.Next()
	}
}

// CSRFProtectionMiddleware validates CSRF tokens for cookie-based flows
func CSRFProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract CSRF token from header
		csrfToken := c.GetHeader("X-CSRF-Token")
		if csrfToken == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Missing CSRF token",
				"code":    "CSRF_FORBIDDEN",
			})
			c.Abort()
			return
		}

		// Validate CSRF token (example: match against a value in the session or a secure store)
		// For now, assume a placeholder validation
		if csrfToken != "expected-csrf-token" {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "Invalid CSRF token",
				"code":    "CSRF_FORBIDDEN",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
