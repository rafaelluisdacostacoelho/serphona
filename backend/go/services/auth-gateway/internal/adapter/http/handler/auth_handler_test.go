package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/response"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/domain/user"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/service/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/auth-gateway/internal/usecase/auth"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zaptest"
)

// stubUserRepo satisfies user.Repository for contract tests.
type stubUserRepo struct{}

func (stubUserRepo) Create(_ context.Context, _ *user.User) error { return nil }
func (stubUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*user.User, error) {
	return &user.User{}, nil
}
func (stubUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return &user.User{}, nil
}
func (stubUserRepo) GetByProvider(_ context.Context, _, _ string) (*user.User, error) {
	return &user.User{}, nil
}
func (stubUserRepo) Update(_ context.Context, _ *user.User) error           { return nil }
func (stubUserRepo) Delete(_ context.Context, _ uuid.UUID) error            { return nil }
func (stubUserRepo) CreateSession(_ context.Context, _ *user.Session) error { return nil }
func (stubUserRepo) GetSession(_ context.Context, _ string) (*user.Session, error) {
	return &user.Session{}, nil
}
func (stubUserRepo) RevokeSession(_ context.Context, _ string) error              { return nil }
func (stubUserRepo) RevokeAllUserSessions(_ context.Context, _ uuid.UUID) error   { return nil }
func (stubUserRepo) CleanupExpiredSessions(_ context.Context) error               { return nil }
func (stubUserRepo) CreateOAuthState(_ context.Context, _ *user.OAuthState) error { return nil }
func (stubUserRepo) GetOAuthState(_ context.Context, _ string) (*user.OAuthState, error) {
	return &user.OAuthState{}, nil
}
func (stubUserRepo) DeleteOAuthState(_ context.Context, _ string) error { return nil }
func (stubUserRepo) CleanupExpiredOAuthStates(_ context.Context) error  { return nil }

// stubTenantService satisfies auth.TenantService for tests.
type stubTenantService struct{}

func (stubTenantService) CreateTenant(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.New(), nil
}

// stubProvider implements auth.OAuthProvider for deterministic URL.
type stubProvider struct{}

func (stubProvider) GetAuthURL(state string) string { return "https://auth.example/" + state }
func (stubProvider) ExchangeCode(_ context.Context, _ string) (*auth.OAuthUserInfo, error) {
	return nil, nil
}

func TestGetOAuthURLEnvelopeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSvc := &jwt.Service{}
	uc := auth.NewUseCase(stubUserRepo{}, jwtSvc, stubTenantService{}, time.Hour)
	uc.RegisterOAuthProvider("google", stubProvider{})

	h := NewAuthHandler(uc, jwtSvc, zaptest.NewLogger(t))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/auth/oauth/google", nil)

	tracer := sdktrace.NewTracerProvider()
	ctx, span := tracer.Tracer("test").Start(req.Context(), "oauth-url")
	ctx = authmw.WithRequestID(ctx, "req-auth-1")
	req = req.WithContext(ctx)
	span.End()

	c.Params = gin.Params{gin.Param{Key: "provider", Value: "google"}}
	c.Request = req

	h.GetOAuthURL(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload struct {
		Data auth.OAuthURLResponse `json:"data"`
		Meta response.Meta         `json:"meta"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Meta.RequestID != "req-auth-1" {
		t.Fatalf("expected request_id req-auth-1, got %s", payload.Meta.RequestID)
	}
	if payload.Meta.TraceID == "" {
		t.Fatalf("expected trace_id to be populated")
	}
	if payload.Data.URL == "" {
		t.Fatalf("expected auth URL to be set")
	}
}

func TestGetOAuthURLErrorEnvelopeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSvc := &jwt.Service{}
	uc := auth.NewUseCase(stubUserRepo{}, jwtSvc, stubTenantService{}, time.Hour)
	h := NewAuthHandler(uc, jwtSvc, zaptest.NewLogger(t))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/auth/oauth/unknown", nil)

	tracer := sdktrace.NewTracerProvider()
	ctx, span := tracer.Tracer("test").Start(req.Context(), "oauth-url-error")
	ctx = authmw.WithRequestID(ctx, "req-auth-err")
	req = req.WithContext(ctx)
	span.End()

	c.Params = gin.Params{gin.Param{Key: "provider", Value: "unknown"}}
	c.Request = req

	h.GetOAuthURL(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var payload struct {
		Error response.ErrorPayload `json:"error"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Error.RequestID != "req-auth-err" {
		t.Fatalf("expected request_id req-auth-err, got %s", payload.Error.RequestID)
	}
	if payload.Error.TraceID == "" {
		t.Fatalf("expected trace_id to be populated")
	}
	if payload.Error.Code == "" || payload.Error.Message == "" {
		t.Fatalf("expected error code/message to be set")
	}
}

func TestLoginValidationErrorEnvelopeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSvc := &jwt.Service{}
	uc := auth.NewUseCase(stubUserRepo{}, jwtSvc, stubTenantService{}, time.Hour)
	h := NewAuthHandler(uc, jwtSvc, zaptest.NewLogger(t))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("{"))

	tracer := sdktrace.NewTracerProvider()
	ctx, span := tracer.Tracer("test").Start(req.Context(), "login-error")
	ctx = authmw.WithRequestID(ctx, "req-auth-login-err")
	req = req.WithContext(ctx)
	span.End()

	c.Request = req

	h.Login(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var payload struct {
		Error response.ErrorPayload `json:"error"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Error.RequestID != "req-auth-login-err" {
		t.Fatalf("expected request_id req-auth-login-err, got %s", payload.Error.RequestID)
	}
	if payload.Error.TraceID == "" {
		t.Fatalf("expected trace_id to be populated")
	}
	if payload.Error.Code == "" || payload.Error.Message == "" {
		t.Fatalf("expected error code/message to be set")
	}
}

func TestRegisterValidationErrorEnvelopeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSvc := &jwt.Service{}
	uc := auth.NewUseCase(stubUserRepo{}, jwtSvc, stubTenantService{}, time.Hour)
	h := NewAuthHandler(uc, jwtSvc, zaptest.NewLogger(t))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("{"))

	tracer := sdktrace.NewTracerProvider()
	ctx, span := tracer.Tracer("test").Start(req.Context(), "register-error")
	ctx = authmw.WithRequestID(ctx, "req-auth-register-err")
	req = req.WithContext(ctx)
	span.End()

	c.Request = req

	h.Register(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var payload struct {
		Error response.ErrorPayload `json:"error"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Error.RequestID != "req-auth-register-err" {
		t.Fatalf("expected request_id req-auth-register-err, got %s", payload.Error.RequestID)
	}
	if payload.Error.TraceID == "" {
		t.Fatalf("expected trace_id to be populated")
	}
	if payload.Error.Code == "" || payload.Error.Message == "" {
		t.Fatalf("expected error code/message to be set")
	}
}

func TestRefreshValidationErrorEnvelopeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSvc := &jwt.Service{}
	uc := auth.NewUseCase(stubUserRepo{}, jwtSvc, stubTenantService{}, time.Hour)
	h := NewAuthHandler(uc, jwtSvc, zaptest.NewLogger(t))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString("{"))

	tracer := sdktrace.NewTracerProvider()
	ctx, span := tracer.Tracer("test").Start(req.Context(), "refresh-error")
	ctx = authmw.WithRequestID(ctx, "req-auth-refresh-err")
	req = req.WithContext(ctx)
	span.End()

	c.Request = req

	h.RefreshToken(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var payload struct {
		Error response.ErrorPayload `json:"error"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Error.RequestID != "req-auth-refresh-err" {
		t.Fatalf("expected request_id req-auth-refresh-err, got %s", payload.Error.RequestID)
	}
	if payload.Error.TraceID == "" {
		t.Fatalf("expected trace_id to be populated")
	}
	if payload.Error.Code == "" || payload.Error.Message == "" {
		t.Fatalf("expected error code/message to be set")
	}
}

func TestLogoutUnauthorizedEnvelopeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSvc := &jwt.Service{}
	uc := auth.NewUseCase(stubUserRepo{}, jwtSvc, stubTenantService{}, time.Hour)
	h := NewAuthHandler(uc, jwtSvc, zaptest.NewLogger(t))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)

	tracer := sdktrace.NewTracerProvider()
	ctx, span := tracer.Tracer("test").Start(req.Context(), "logout-error")
	ctx = authmw.WithRequestID(ctx, "req-auth-logout-err")
	req = req.WithContext(ctx)
	span.End()

	c.Request = req

	h.Logout(c)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	var payload struct {
		Error response.ErrorPayload `json:"error"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Error.RequestID != "req-auth-logout-err" {
		t.Fatalf("expected request_id req-auth-logout-err, got %s", payload.Error.RequestID)
	}
	if payload.Error.TraceID == "" {
		t.Fatalf("expected trace_id to be populated")
	}
	if payload.Error.Code == "" || payload.Error.Message == "" {
		t.Fatalf("expected error code/message to be set")
	}
}
