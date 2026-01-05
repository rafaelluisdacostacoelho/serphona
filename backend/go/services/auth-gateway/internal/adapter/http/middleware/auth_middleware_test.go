package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/service/jwt"
	"github.com/stretchr/testify/assert"
)

// Define a mock JWT service
type mockJWTService struct {
	fixedTenantID uuid.UUID
}

func (m *mockJWTService) ValidateAccessToken(tokenString string) (*jwt.Claims, error) {
	if tokenString == "valid-token" {
		return &jwt.Claims{
			UserID:   uuid.New(),
			Email:    "test@example.com",
			TenantID: m.fixedTenantID,
			Role:     "user",
		}, nil
	}
	return nil, jwt.ErrInvalidToken
}

func (m *mockJWTService) ValidateRefreshToken(tokenString string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (m *mockJWTService) GenerateAccessToken(userID, tenantID uuid.UUID, email, role string) (string, error) {
	return "", nil
}

func (m *mockJWTService) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	return "", nil
}

// Ensure mockJWTService implements jwt.JWTService
var _ jwt.JWTService = (*mockJWTService)(nil)

func TestAuthMiddleware_Authenticate(t *testing.T) {
	// Initialize Gin in test mode
	gin.SetMode(gin.TestMode)

	// Define a fixed tenantID for consistency
	fixedTenantID := uuid.MustParse("e92bf6bd-b53c-4759-8d64-9b3d27e5afec")

	// Define test cases
	tests := []struct {
		name           string
		setupRequest   func(req *http.Request)
		expectedStatus int
		expectedTenant uuid.UUID
	}{
		{
			name: "Valid token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer valid-token")
			},
			expectedStatus: http.StatusOK,
			expectedTenant: fixedTenantID,
		},
		{
			name:           "Missing token",
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid token",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer invalid-token")
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the mock service with a fixed tenantID
			mockServiceInstance := &mockJWTService{
				fixedTenantID: fixedTenantID,
			}

			// Pass the renamed variable to the middleware
			middleware := NewAuthMiddleware(mockServiceInstance)

			// Create a new Gin router
			r := gin.New()
			r.Use(middleware.Authenticate())
			r.GET("/test", func(c *gin.Context) {
				if tenantID, exists := c.Get("tenantID"); exists {
					assert.Equal(t, tt.expectedTenant, tenantID, "tenantID should match the expected value")
				}
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			// Create a new HTTP request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			// Create a response recorder
			w := httptest.NewRecorder()

			// Perform the request
			r.ServeHTTP(w, req)

			// Assert the response status code
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Simplified test setup
func TestAuthMiddleware_Simple(t *testing.T) {
	// Define a fixed tenantID
	fixedTenantID := uuid.MustParse("e92bf6bd-b53c-4759-8d64-9b3d27e5afec")

	// Create a mock service instance
	mockService := &mockJWTService{
		fixedTenantID: fixedTenantID,
	}

	// Verify interface implementation
	var _ jwt.JWTService = mockService

	// Create the middleware
	middleware := NewAuthMiddleware(mockService)

	// Assert middleware is not nil
	if middleware == nil {
		t.Fatal("middleware should not be nil")
	}
}
