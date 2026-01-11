package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	authtypes "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/types"

	"tools-manager/internal/events"
	"tools-manager/internal/metrics"
	"tools-manager/internal/repository"
)

// Integration test that exercises the catalog handler with real Postgres + RLS, validating ETag behavior and metrics emission.
func TestCatalogResolvedETagAndMetrics(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	pool := setupHandlerTestDB(t)
	repo := repository.New(pool)
	handler := NewCatalogHandler(zap.NewNop(), repo, events.NewNotifier(""))

	tenant := "33333333-3333-3333-3333-333333333333"
	creator := uuid.New()

	_, err := repo.CreateTool(ctx, tenant, repository.CreateToolParams{
		Name:         "catalog-tool",
		DisplayName:  "Catalog Tool",
		Description:  "Used for catalog tests",
		Category:     nil,
		Tags:         []string{"alpha"},
		Version:      "1.2.3",
		Status:       "published",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
		Definition:   json.RawMessage(`{"kind":"http"}`),
		Allowlist:    json.RawMessage(`{"hosts":["api.example.com"],"protocols":["https"]}`),
		Metadata:     json.RawMessage(`{"payload_bytes_limit":5120}`),
		TimeoutSecs:  30,
		MaxRetries:   2,
		PayloadLimit: 5120,
		IsPublic:     true,
		CreatedBy:    creator,
	})
	if err != nil {
		t.Fatalf("seed tool: %v", err)
	}

	// First call should return 200 with an ETag and emit a catalog read metric.
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("claims", &authtypes.Claims{TenantID: tenant, Role: "admin", UserID: creator.String()})
		c.Set("requestID", "req-1")
	})
	router.GET("/catalog/resolved", handler.Resolved)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/catalog/resolved?limit=10&offset=0", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on first call, got %d", w.Code)
	}
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header on first response")
	}
	tools, err := repo.ListTools(ctx, tenant)
	if err != nil {
		t.Fatalf("list tools for etag assertion: %v", err)
	}
	expectedETag := computeETag(tools)
	if etag != expectedETag {
		t.Fatalf("etag mismatch: got %s want %s", etag, expectedETag)
	}
	if got := testutil.ToFloat64(metrics.CatalogReads.WithLabelValues(tenant)); got != 1 {
		t.Fatalf("expected catalog read metric to be 1, got %v", got)
	}

	// Second call with matching If-None-Match should return 304 without incrementing metrics.
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/catalog/resolved?limit=10&offset=0", nil)
	req2.Header.Set("If-None-Match", expectedETag)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Fatalf("expected 304 on cache hit, got %d; response etag=%s body=%s", w2.Code, w2.Header().Get("ETag"), w2.Body.String())
	}
	if got := testutil.ToFloat64(metrics.CatalogReads.WithLabelValues(tenant)); got != 1 {
		t.Fatalf("expected catalog read metric to remain 1 after 304, got %v", got)
	}
}

// Ensure the tools list handler emits metrics and honors tenant scoping using the same Postgres fixture.
func TestToolsListEmitsMetrics(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	pool := setupHandlerTestDB(t)
	repo := repository.New(pool)
	handler := NewToolsHandler(zap.NewNop(), repo, events.NewNotifier(""))

	tenant := "44444444-4444-4444-4444-444444444444"
	creator := uuid.New()

	_, err := repo.CreateTool(ctx, tenant, repository.CreateToolParams{
		Name:         "list-tool",
		DisplayName:  "List Tool",
		Description:  "Used for list metrics",
		Category:     nil,
		Tags:         []string{"beta"},
		Version:      "0.0.1",
		Status:       "published",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
		Definition:   json.RawMessage(`{"kind":"http"}`),
		Allowlist:    json.RawMessage(`{"hosts":["list.example.com"],"protocols":["https"]}`),
		Metadata:     json.RawMessage(`{"payload_bytes_limit":2048}`),
		TimeoutSecs:  20,
		MaxRetries:   1,
		PayloadLimit: 2048,
		IsPublic:     false,
		CreatedBy:    creator,
	})
	if err != nil {
		t.Fatalf("seed tool: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/tools", nil)
	c.Set("claims", &authtypes.Claims{TenantID: tenant, Role: "admin", UserID: creator.String()})
	c.Set("requestID", "req-3")
	c.Params = gin.Params{{Key: "tenant_id", Value: tenant}}

	handler.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d", w.Code)
	}
	if got := testutil.ToFloat64(metrics.ToolsReads.WithLabelValues(tenant)); got != 1 {
		t.Fatalf("expected tools read metric to be 1, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.Errors.WithLabelValues(tenant, c.FullPath(), "500")); got != 0 {
		t.Fatalf("expected no error metrics, got %v", got)
	}
}

// Below are minimal Postgres helpers duplicated from repository tests to exercise handlers against a real DB.
const handlerDefaultTestDBURL = "postgres://test:test@localhost:55432/test?sslmode=disable"

func setupHandlerTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	dbURL := os.Getenv("TOOLS_MANAGER_TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = handlerDefaultTestDBURL
	}

	if err := handlerPingDB(ctx, dbURL); err != nil {
		dbURL = startHandlerPostgresContainer(t, ctx)
	}

	ensureHandlerRoleAndMigrate(t, ctx, dbURL)

	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		t.Fatalf("parse pool config: %v", err)
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET ROLE application")
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	t.Cleanup(func() { pool.Close() })
	return pool
}

func ensureHandlerRoleAndMigrate(t *testing.T, ctx context.Context, dbURL string) {
	t.Helper()

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'application') THEN
        CREATE ROLE application;
    END IF;
    GRANT application TO CURRENT_USER;
END $$;
`)
	if err != nil {
		t.Fatalf("ensure application role: %v", err)
	}

	_, err = conn.Exec(ctx, `
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'service_account') THEN
        CREATE ROLE service_account;
    END IF;
END $$;
`)
	if err != nil {
		t.Fatalf("ensure service_account role: %v", err)
	}

	downFiles := []string{
		"000003_quota_rules.down.sql",
		"000002_policy_rules.down.sql",
		"000001_create_catalog.down.sql",
	}
	for _, file := range downFiles {
		sql := handlerReadSQL(t, file)
		if _, err := conn.Exec(ctx, sql); err != nil {
			t.Fatalf("apply down migration %s: %v", file, err)
		}
	}

	upFiles := []string{
		"000001_create_catalog.up.sql",
		"000002_policy_rules.up.sql",
		"000003_quota_rules.up.sql",
	}
	for _, file := range upFiles {
		sql := handlerReadSQL(t, file)
		if _, err := conn.Exec(ctx, sql); err != nil {
			t.Fatalf("apply up migration %s: %v", file, err)
		}
	}

	_, err = conn.Exec(ctx, `
GRANT ALL PRIVILEGES ON SCHEMA public TO application;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO application;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO application;
`)
	if err != nil {
		t.Fatalf("grant privileges to application role: %v", err)
	}
}

func handlerPingDB(ctx context.Context, dbURL string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	return conn.Ping(ctx)
}

func startHandlerPostgresContainer(t *testing.T, ctx context.Context) string {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("container mapped port: %v", err)
	}

	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	return "postgres://test:test@" + host + ":" + port.Port() + "/test?sslmode=disable"
}

func handlerReadSQL(t *testing.T, fileName string) string {
	t.Helper()

	path := filepath.Join("..", "..", "migrations", fileName)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", fileName, err)
	}
	return string(content)
}
