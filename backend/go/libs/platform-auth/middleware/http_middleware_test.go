package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	autherrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/errors"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

func TestRequireAuthHTTPSuccess(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()
	middleware.SetMetricsRegisterer(prometheus.NewRegistry())
	t.Cleanup(func() { middleware.SetMetricsRegisterer(nil) })

	var capturedCtx context.Context
	handler := middleware.RequireAuthHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		claims, err := middleware.ClaimsFromContext(r.Context())
		if err != nil {
			t.Fatalf("expected claims in context, got error %v", err)
		}
		if claims.UserID != "11111111-1111-1111-1111-111111111111" {
			t.Fatalf("unexpected user id %s", claims.UserID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	token := signedMiddlewareToken(t, "user", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	fields, ok := middleware.SafeRequestFieldsFromContext(capturedCtx)
	if !ok {
		t.Fatalf("expected safe request fields in context")
	}
	if fields["request_id"] == "" {
		t.Fatalf("expected request_id in safe request fields")
	}
	headers, _ := fields["headers"].(http.Header)
	if headers.Get("Authorization") != "[REDACTED]" {
		t.Fatalf("expected authorization to be redacted")
	}

	assertMetricCounter(t, middleware.MetricAuthRequestsTotal, map[string]string{"transport": "http", "result": "ok"}, 1)
}

func TestRequireAuthHTTPMissingToken(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()
	middleware.SetMetricsRegisterer(prometheus.NewRegistry())
	t.Cleanup(func() { middleware.SetMetricsRegisterer(nil) })

	handler := middleware.RequireAuthHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["code"] != autherrors.CodeMissingToken {
		t.Fatalf("expected code %s, got %v", autherrors.CodeMissingToken, body["code"])
	}

	assertMetricCounter(t, middleware.MetricAuthRequestsTotal, map[string]string{"transport": "http", "result": "unauthorized"}, 1)
}

func TestRequireScopesHTTPDenied(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireAuthHTTP(middleware.RequireScopesHTTP("read:reports")(protected))

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:invoices"})

	req := httptest.NewRequest(http.MethodGet, "/reports", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}

	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["code"] != autherrors.CodeInsufficientPermissions {
		t.Fatalf("expected code %s, got %v", autherrors.CodeInsufficientPermissions, body["code"])
	}
}

func TestChiRequireAnyScopeAlias(t *testing.T) {
	authjwt.SetSecret(middlewareTestSecret)
	authjwt.ResetSecretOnceForTests()

	protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.ChiRequireAuth(middleware.ChiRequireAnyScope("read:reports", "read:invoices")(protected))

	token := signedTokenWithScopes(t, "user", time.Now().Add(time.Hour), []string{"read:reports"})

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestHTTPRecoveryHandlesPanic(t *testing.T) {
	handler := middleware.HTTPRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}

	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["code"] != "internal_error" {
		t.Fatalf("expected internal_error code, got %v", body["code"])
	}
}

func TestLimitBodySizeHTTPRejectsLargePayload(t *testing.T) {
	handler := middleware.LimitBodySizeHTTP(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("123456"))
	req.Header.Set("Content-Length", "6")

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["code"] != "request_too_large" {
		t.Fatalf("expected request_too_large code, got %v", body["code"])
	}
}

func assertMetricCounter(t *testing.T, name string, labels map[string]string, expected float64) {
	t.Helper()

	mfs, err := middleware.MetricsGatherer().Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.Metric {
			match := true
			for _, lp := range m.Label {
				if labels[lp.GetName()] != lp.GetValue() {
					match = false
					break
				}
			}
			if match {
				if m.Counter == nil {
					t.Fatalf("metric %s is not a counter", name)
				}
				if m.Counter.GetValue() != expected {
					t.Fatalf("expected counter %s with labels %v to be %v, got %v", name, labels, expected, m.Counter.GetValue())
				}
				return
			}
		}
	}

	t.Fatalf("metric %s with labels %v not found", name, labels)
}
