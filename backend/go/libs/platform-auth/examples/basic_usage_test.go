package main_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status": "healthy"}`, w.Body.String())
}

func TestProtectedProfileEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authjwt.SetSecret("example-secret")
	authjwt.ResetSecretOnceForTests()
	t.Cleanup(authjwt.ResetSecretForTesting)

	r := gin.Default()

	protected := r.Group("/api/v1")
	protected.Use(middleware.RequireAuth())
	protected.GET("/profile", func(c *gin.Context) {
		claims, _ := middleware.GetClaimsFromContext(c)
		c.JSON(http.StatusOK, gin.H{
			"message": "Perfil do usuario",
			"user": gin.H{
				"id":       claims.UserID,
				"email":    claims.Email,
				"name":     claims.Name,
				"role":     claims.Role,
				"tenantId": claims.TenantID,
			},
		})
	})

	// Mock request without token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	r.ServeHTTP(w, req)

	// Assert unauthorized due to missing token (secret configured so middleware should return 401)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
