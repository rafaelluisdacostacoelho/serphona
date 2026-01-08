package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
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

var ensureSecretOnce sync.Once

func init() {
	gin.SetMode(gin.TestMode)
	// Unset the JWT_SECRET environment variable to simulate a missing secret.
	os.Unsetenv("JWT_SECRET")
	// Reset the sync.Once mechanism to ensure EnsureSecretLoaded re-evaluates the secret loading logic.
	ensureSecretOnce = sync.Once{}
	// Use the helper function to reset the jwtSecret and sync.Once.
	authjwt.ResetSecretForTesting()
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
		fields, ok := middleware.SafeRequestFieldsFromContext(c.Request.Context())
		if !ok || fields["request_id"] == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "missing safe fields"})
			return
		}
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

func TestRequireRoleAllowsMatch(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)

	router := gin.New()
	router.Use(middleware.RequireAuth(), middleware.RequireRole("user"))
	router.GET("/role", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	token := signedMiddlewareToken(t, "user", time.Now().Add(time.Hour))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/role", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireAdminUnauthorizedWhenMissingClaims(t *testing.T) {
	router := gin.New()
	router.Use(middleware.RequireAdmin())
	router.GET("/admin", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireAnyScopeAllowsOne(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)

	router := gin.New()
	router.Use(middleware.RequireAuth(), middleware.RequireAnyScope("read:a", "write:b"))
	router.GET("/scoped", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"write:b"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireScopesUnauthorizedWhenMissingClaims(t *testing.T) {
	router := gin.New()
	router.Use(middleware.RequireScopes("s1"))
	router.GET("/needs-scopes", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/needs-scopes", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuthMissingSecretReturns500(t *testing.T) {
	authjwt.SetSecret("")
	authjwt.ResetSecretOnceForTests()
	os.Unsetenv("JWT_SECRET")

	router := gin.New()
	router.Use(middleware.RequireAuth())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when secret missing, got %d", w.Code)
	}

	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != autherrors.CodeAuthConfigMissing {
		t.Fatalf("expected code %s, got %v", autherrors.CodeAuthConfigMissing, body["code"])
	}
}

func signedTokenWithScopes(t *testing.T, role string, exp time.Time, scopes []string) string {
	t.Helper()

	claims := types.Claims{
		UserID:    "11111111-1111-1111-1111-111111111111",
		Email:     "user@example.com",
		Name:      "Test User",
		Role:      role,
		TenantID:  "22222222-2222-2222-2222-222222222222",
		SessionID: "33333333-3333-3333-3333-333333333333",
		Scopes:    scopes,
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

func TestRequireScopesAllowed(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireScopes("read:invoices", "write:invoices"))
	router.GET("/invoices", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:invoices", "write:invoices", "extra:scope"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireScopesDenied(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireScopes("read:invoices", "write:invoices"))
	router.GET("/invoices", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:invoices"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
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

func TestRequireAllScopesAlias(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireAllScopes("read:all", "write:all"))
	router.GET("/data", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:all", "write:all"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireAnyScopeAllowed(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireAnyScope("read:invoices", "read:reports"))
	router.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:reports"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireAnyScopeDenied(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireAnyScope("read:invoices", "read:reports"))
	router.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"write:invoices"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRequireAnyScopeAllowsSingleMatch(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	router := gin.New()
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireAnyScope("read:reports", "write:reports"))
	router.GET("/reports", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"write:reports"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/reports", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Set("claims", &types.Claims{UserID: "user-1", TenantID: "tenant-1"})

	id, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "user-1" {
		t.Fatalf("expected user-1, got %s", id)
	}
}

func TestGetUserIDFromContextMissingClaims(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	_, err := middleware.GetUserIDFromContext(c)
	if !errors.Is(err, autherrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestGetTenantIDFromContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Set("claims", &types.Claims{UserID: "user-1", TenantID: "tenant-1"})

	id, err := middleware.GetTenantIDFromContext(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "tenant-1" {
		t.Fatalf("expected tenant-1, got %s", id)
	}
}
