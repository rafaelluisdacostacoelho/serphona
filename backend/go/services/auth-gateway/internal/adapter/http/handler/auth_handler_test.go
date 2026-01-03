package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// stubUserRepoNoExisting simulates no pre-existing user for registration path.
type stubUserRepoNoExisting struct{}

func (stubUserRepoNoExisting) Create(_ context.Context, _ *user.User) error { return nil }
func (stubUserRepoNoExisting) GetByID(_ context.Context, _ uuid.UUID) (*user.User, error) {
	return &user.User{}, nil
}
func (stubUserRepoNoExisting) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, errors.New("not found")
}
func (stubUserRepoNoExisting) GetByProvider(_ context.Context, _, _ string) (*user.User, error) {
	return &user.User{}, nil
}
func (stubUserRepoNoExisting) Update(_ context.Context, _ *user.User) error           { return nil }
func (stubUserRepoNoExisting) Delete(_ context.Context, _ uuid.UUID) error            { return nil }
func (stubUserRepoNoExisting) CreateSession(_ context.Context, _ *user.Session) error { return nil }
func (stubUserRepoNoExisting) GetSession(_ context.Context, _ string) (*user.Session, error) {
	return &user.Session{}, nil
}
func (stubUserRepoNoExisting) RevokeSession(_ context.Context, _ string) error            { return nil }
func (stubUserRepoNoExisting) RevokeAllUserSessions(_ context.Context, _ uuid.UUID) error { return nil }
func (stubUserRepoNoExisting) CleanupExpiredSessions(_ context.Context) error             { return nil }
func (stubUserRepoNoExisting) CreateOAuthState(_ context.Context, _ *user.OAuthState) error {
	return nil
}
func (stubUserRepoNoExisting) GetOAuthState(_ context.Context, _ string) (*user.OAuthState, error) {
	return &user.OAuthState{}, nil
}
func (stubUserRepoNoExisting) DeleteOAuthState(_ context.Context, _ string) error { return nil }
func (stubUserRepoNoExisting) CleanupExpiredOAuthStates(_ context.Context) error  { return nil }

// stubTenantService satisfies auth.TenantService for tests.
type stubTenantService struct {
	id uuid.UUID
}

func (s stubTenantService) CreateTenant(_ context.Context, _, _, _, _, _ string) (uuid.UUID, error) {
	if s.id == uuid.Nil {
		return uuid.New(), nil
	}
	return s.id, nil
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

func TestRegisterContractIncludesTenantIDAnd201(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fixedID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	jtwSvc := &jwt.Service{}
	uc := auth.NewUseCase(stubUserRepoNoExisting{}, jtwSvc, stubTenantService{id: fixedID}, time.Hour)
	h := NewAuthHandler(uc, jtwSvc, zaptest.NewLogger(t))

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := map[string]any{
		"email":        "user@example.com",
		"password":     "strongpassword",
		"name":         "Example User",
		"tenantName":   "Example Org",
		"plan":         "starter",
		"billingEmail": "billing@example.com",
		"phone":        "+15551234567",
	}

	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	ctx := authmw.WithRequestID(context.Background(), "req-register-1")
	req = req.WithContext(ctx)

	c.Request = req

	h.Register(c)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var payload struct {
		Data auth.AuthResponse `json:"data"`
		Meta response.Meta     `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Data.User.TenantID != fixedID {
		t.Fatalf("expected tenant id %s, got %s", fixedID, payload.Data.User.TenantID)
	}
	if payload.Meta.RequestID != "req-register-1" {
		t.Fatalf("expected request_id req-register-1, got %s", payload.Meta.RequestID)
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
