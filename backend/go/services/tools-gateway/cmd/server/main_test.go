package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	authjwt "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/jwt"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/adapter/http/handler"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/adapter/http/middleware"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/entity"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/domain/repository"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/services/tools-gateway/internal/usecase"
)

const testJWTSecret = "test-secret"

func TestRouterListsToolsAndExposesMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, token := newTestRouter(t)

	req := authedRequest(http.MethodGet, "/api/v1/tools?limit=5&offset=0", nil, token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var body struct {
		Data struct {
			Tools []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Data.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(body.Data.Tools))
	}

	metricsResp := httptest.NewRecorder()
	router.ServeHTTP(metricsResp, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if metricsResp.Code != http.StatusOK {
		t.Fatalf("expected metrics status %d, got %d", http.StatusOK, metricsResp.Code)
	}

	metricsBody := metricsResp.Body.String()
	if !strings.Contains(metricsBody, `tools_gateway_requests_total{method="GET",path="/api/v1/tools",service="tools-gateway",status="2xx"`) {
		t.Fatalf("metrics did not record tools list handler: %s", metricsBody)
	}
	if !strings.Contains(metricsBody, `tools_gateway_auth_total{result="ok",service="tools-gateway",tenant_id="`) {
		t.Fatalf("auth metrics missing success: %s", metricsBody)
	}
}

func TestRouterRecordsErrorMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, token := newTestRouter(t)

	req := authedRequest(http.MethodGet, "/api/v1/tools/not-a-uuid", nil, token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}

	metricsResp := httptest.NewRecorder()
	router.ServeHTTP(metricsResp, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if metricsResp.Code != http.StatusOK {
		t.Fatalf("expected metrics status %d, got %d", http.StatusOK, metricsResp.Code)
	}

	metricsBody := metricsResp.Body.String()
	if !strings.Contains(metricsBody, `tools_gateway_requests_total{method="GET",path="/api/v1/tools/:id",service="tools-gateway",status="4xx"`) {
		t.Fatalf("metrics did not record error handler invocation: %s", metricsBody)
	}
}

func TestHealthAndServerErrorMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, token := newTestRouter(t)

	// Health check
	healthResp := httptest.NewRecorder()
	router.ServeHTTP(healthResp, httptest.NewRequest(http.MethodGet, "/health", nil))
	if healthResp.Code != http.StatusOK {
		t.Fatalf("expected health status %d, got %d", http.StatusOK, healthResp.Code)
	}

	// Force a 500 to exercise 5xx metric bucket
	router.GET("/boom", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})

	boomResp := httptest.NewRecorder()
	router.ServeHTTP(boomResp, authedRequest(http.MethodGet, "/boom", nil, token))
	if boomResp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, boomResp.Code)
	}

	metricsResp := httptest.NewRecorder()
	router.ServeHTTP(metricsResp, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metricsResp.Code != http.StatusOK {
		t.Fatalf("expected metrics status %d, got %d", http.StatusOK, metricsResp.Code)
	}

	metricsBody := metricsResp.Body.String()
	if !strings.Contains(metricsBody, `tools_gateway_requests_total{method="GET",path="/boom",service="tools-gateway",status="5xx"`) {
		t.Fatalf("metrics did not record 5xx handler invocation: %s", metricsBody)
	}
}

func TestUnknownPathMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := newTestRouter(t)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.Code)
	}

	metricsResp := httptest.NewRecorder()
	router.ServeHTTP(metricsResp, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metricsResp.Code != http.StatusOK {
		t.Fatalf("expected metrics status %d, got %d", http.StatusOK, metricsResp.Code)
	}

	metricsBody := metricsResp.Body.String()
	if !strings.Contains(metricsBody, `tools_gateway_requests_total{method="GET",path="unknown",service="tools-gateway",status="4xx"`) {
		t.Fatalf("metrics did not record unknown path: %s", metricsBody)
	}
}

func TestExecuteToolRequiresScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router, _ := newTestRouter(t)

	token := newTestToken(t, "tools:read")
	body := strings.NewReader(`{"input":{}}`)
	path := "/api/v1/tools/" + uuid.NewString() + "/execute"
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, authedRequest(http.MethodPost, path, body, token))

	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, resp.Code)
	}
}

func newToolHandler() *handler.ToolHandler {
	tool := &entity.Tool{
		ID:                 uuid.New(),
		Name:               "search",
		DisplayName:        "Search",
		Description:        "Simple search tool",
		Category:           entity.CategorySearch,
		Method:             entity.HTTPMethodGET,
		BaseURL:            "https://example.com",
		EndpointPath:       "/search",
		Headers:            json.RawMessage(`{"Content-Type":"application/json"}`),
		AuthType:           entity.AuthTypeNone.String(),
		AuthConfig:         json.RawMessage(`{}`),
		InputSchema:        json.RawMessage(`{"type":"object"}`),
		OutputSchema:       json.RawMessage(`{"type":"object"}`),
		TimeoutSeconds:     30,
		MaxRetries:         1,
		RetryDelaySeconds:  1,
		RateLimitPerMinute: 60,
		RateLimitPerHour:   100,
		CreditCost:         1,
		IsActive:           true,
		IsPublic:           true,
	}

	return handler.NewToolHandler(&stubToolService{
		tools: []*entity.Tool{tool},
		total: 1,
	}, &stubExecutorService{})
}

type stubToolService struct {
	tools   []*entity.Tool
	total   int64
	listErr error
}

func (s *stubToolService) CreateTool(ctx context.Context, tool *entity.Tool) error { return nil }
func (s *stubToolService) GetTool(ctx context.Context, tenantID uuid.UUID, toolID uuid.UUID) (*entity.Tool, error) {
	return s.tools[0], nil
}
func (s *stubToolService) GetToolByName(ctx context.Context, tenantID uuid.UUID, name string) (*entity.Tool, error) {
	return nil, nil
}
func (s *stubToolService) ListTools(ctx context.Context, tenantID uuid.UUID, filters repository.ToolFilters) ([]*entity.Tool, int64, error) {
	return s.tools, s.total, s.listErr
}
func (s *stubToolService) UpdateTool(ctx context.Context, tool *entity.Tool) error { return nil }
func (s *stubToolService) DeleteTool(ctx context.Context, toolID uuid.UUID) error  { return nil }
func (s *stubToolService) ListPublicTools(ctx context.Context) ([]*entity.Tool, error) {
	return s.tools, nil
}
func (s *stubToolService) ListToolsByCategory(ctx context.Context, category string) ([]*entity.Tool, error) {
	return s.tools, nil
}

type stubExecutorService struct{}

func (s *stubExecutorService) Execute(ctx context.Context, req *usecase.ExecutionRequest) (*usecase.ExecutionResponse, error) {
	return &usecase.ExecutionResponse{ExecutionID: uuid.New(), ToolID: req.ToolID, ToolName: "search", Status: "ok"}, nil
}
func (s *stubExecutorService) GetExecution(ctx context.Context, executionID uuid.UUID) (*entity.ToolExecution, error) {
	return nil, nil
}
func (s *stubExecutorService) ListExecutions(ctx context.Context, tenantID uuid.UUID, filters repository.ExecutionFilters) ([]*entity.ToolExecution, int64, error) {
	return nil, 0, nil
}
func (s *stubExecutorService) GetExecutionStats(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*repository.ExecutionStats, error) {
	return nil, nil
}
func (s *stubExecutorService) GetCostBreakdown(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]*repository.CostBreakdown, error) {
	return nil, nil
}

func newTestRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()

	t.Setenv("JWT_SECRET", testJWTSecret)
	authjwt.ResetValidationConfigForTests()

	token := newTestToken(t)

	reg := prometheus.NewRegistry()
	middleware.SetMetricsRegisterer(reg)
	t.Cleanup(func() { middleware.SetMetricsRegisterer(nil) })

	return setupRouter(newToolHandler(), "tools-gateway", 1024*1024), token
}

func newTestToken(t *testing.T, scopes ...string) string {
	t.Helper()

	if len(scopes) == 0 {
		scopes = []string{"tools:read", "tools:write", "tools:execute"}
	}

	claims := types.Claims{
		UserID:   uuid.NewString(),
		Email:    "user@example.com",
		Name:     "Test User",
		Role:     "user",
		TenantID: uuid.NewString(),
		Scopes:   scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return signed
}

func authedRequest(method, path string, body io.Reader, token string) *http.Request {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}
