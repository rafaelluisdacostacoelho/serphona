package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockJWT struct {
	mock.Mock
}

func (m *MockJWT) EnsureSecretLoaded() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockJWT) ValidateTokenFromHeader(header string) (*types.Claims, error) {
	args := m.Called(header)
	return args.Get(0).(*types.Claims), args.Error(1)
}

func TestRequireAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockJWT := new(MockJWT)

	originalValidator := middleware.ValidateTokenFromHeader
	t.Cleanup(func() {
		middleware.ValidateTokenFromHeader = originalValidator
	})

	previousSecret := os.Getenv("JWT_SECRET")
	t.Cleanup(func() {
		if previousSecret == "" {
			os.Unsetenv("JWT_SECRET")
			return
		}
		os.Setenv("JWT_SECRET", previousSecret)
	})

	// Mock EnsureSecretLoaded to simulate a loaded secret
	mockJWT.On("EnsureSecretLoaded").Return(nil)

	// Set a dummy secret to avoid AUTH_CONFIG_MISSING
	os.Setenv("JWT_SECRET", "dummy-secret")

	// Mock ValidateTokenFromHeader to simulate a valid token
	mockJWT.On("ValidateTokenFromHeader", mock.Anything).Return(&types.Claims{
		UserID:   "123",
		Email:    "test@example.com",
		Name:     "Test User",
		Role:     "user",
		TenantID: "tenant-123",
	}, nil)

	// Override the ValidateTokenFromHeader variable to use the mock.
	middleware.ValidateTokenFromHeader = mockJWT.ValidateTokenFromHeader

	r := gin.Default()
	r.Use(middleware.RequireAuth())
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.JSONEq(t, `{"message": "success"}`, w.Body.String())
}
