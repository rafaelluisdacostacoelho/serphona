package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
)

const benchSecret = "bench-secret-key-32-bytes-minimum!"

func benchmarkToken(t testing.TB) string {
	t.Helper()

	claims := types.Claims{
		UserID:   "bench-user",
		TenantID: "bench-tenant",
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(benchSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return token
}

func BenchmarkRequireAuthHTTP(b *testing.B) {
	authjwt.SetSecret(benchSecret)
	token := benchmarkToken(b)

	handler := RequireAuthHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/resource", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rr.Code)
		}
	}
}

func BenchmarkSafeRequestFields(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/v1/resource?token=secret", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Trace", "abc")
	req = req.WithContext(WithRequestID(req.Context(), "req-bench"))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = SafeRequestFields(req)
	}
}
