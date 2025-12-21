package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

const middlewareTestSecret = "test-secret-key-32-characters-minimum!"

func init() {
	gin.SetMode(gin.TestMode)
}

func signedMiddlewareToken(t *testing.T, role string, exp time.Time) string {
	t.Helper()

	claims := types.Claims{
		UserID:    "11111111-1111-1111-1111-111111111111",
		Email:     "user@example.com",
		Name:      "Test User",
		Role:      role,
		TenantID:  "22222222-2222-2222-2222-222222222222",
		SessionID: "33333333-3333-3333-3333-333333333333",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(middlewareTestSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return token
}

func TestRequireAuthSuccess(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		claims, _ := middleware.GetClaimsFromContext(c)
		c.JSON(http.StatusOK, gin.H{"userId": claims.UserID})
	})

	token := signedMiddlewareToken(t, "user", time.Now().Add(time.Hour))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["userId"] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestRequireAuthMissingToken(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != autherrors.CodeMissingToken {
		t.Fatalf("expected code %s, got %v", autherrors.CodeMissingToken, body["code"])
	}
}

func TestRequireAuthExpiredToken(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := signedMiddlewareToken(t, "user", time.Now().Add(-time.Minute))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != autherrors.CodeTokenExpired {
		t.Fatalf("expected code %s, got %v", autherrors.CodeTokenExpired, body["code"])
	}
}

func TestRequireAuthInvalidToken(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != autherrors.CodeInvalidToken {
		t.Fatalf("expected code %s, got %v", autherrors.CodeInvalidToken, body["code"])
	}
}

func TestRequireRoleDenied(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireRole("admin"))
	router.GET("/admin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := signedMiddlewareToken(t, "user", time.Now().Add(time.Hour))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != autherrors.CodeInsufficientPermissions {
		t.Fatalf("expected code %s, got %v", autherrors.CodeInsufficientPermissions, body["code"])
	}
}

func TestRequireAdminAllowsAdmin(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireAdmin())
	router.GET("/admin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := signedMiddlewareToken(t, "admin", time.Now().Add(time.Hour))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireSuperAdmin(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireSuperAdmin())
	router.GET("/super", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	superToken := signedMiddlewareToken(t, "superadmin", time.Now().Add(time.Hour))
	adminToken := signedMiddlewareToken(t, "admin", time.Now().Add(time.Hour))

	// allow superadmin
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/super", nil)
	req.Header.Set("Authorization", "Bearer "+superToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// deny admin
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/super", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRequireAuthPanicsWhenSecretMissing(t *testing.T) {
	authjwt.SetSecret("")
	authjwt.ResetSecretOnceForTests()

	router := gin.New()
	router.Use(middleware.RequireAuth())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when JWT secret is not configured")
		}
	}()

	router.ServeHTTP(w, req)
}
