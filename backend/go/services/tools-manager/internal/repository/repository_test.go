package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultTestDBURL = "postgres://test:test@localhost:55432/test?sslmode=disable"

func TestCreateAndListRespectsRLS(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)

	repo := New(pool)

	tenantA := "11111111-1111-1111-1111-111111111111"
	tenantB := "22222222-2222-2222-2222-222222222222"
	creator := uuid.New()

	_, err := repo.CreateTool(ctx, tenantA, CreateToolParams{
		Name:         "tool-a",
		DisplayName:  "Tool A",
		Description:  "Tenant A tool",
		Category:     strPtr("category-a"),
		Tags:         []string{"alpha"},
		Version:      "1.0.0",
		Status:       "published",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
		Definition:   json.RawMessage(`{"kind":"http"}`),
		Allowlist:    json.RawMessage(`{"hosts":["api.tenant-a.local"],"protocols":["https"]}`),
		Metadata:     json.RawMessage(`{"payload_bytes_limit":10240}`),
		TimeoutSecs:  30,
		MaxRetries:   2,
		PayloadLimit: 10240,
		IsPublic:     true,
		CreatedBy:    creator,
	})
	if err != nil {
		t.Fatalf("create tool for tenant A failed: %v", err)
	}

	_, err = repo.CreateTool(ctx, tenantB, CreateToolParams{
		Name:         "tool-b",
		DisplayName:  "Tool B",
		Description:  "Tenant B tool",
		Category:     strPtr("category-b"),
		Tags:         []string{"beta"},
		Version:      "2.0.0",
		Status:       "draft",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
		Definition:   json.RawMessage(`{"kind":"http"}`),
		Allowlist:    json.RawMessage(`{"hosts":["api.tenant-b.local"],"protocols":["http"]}`),
		Metadata:     json.RawMessage(`{"payload_bytes_limit":20480}`),
		TimeoutSecs:  45,
		MaxRetries:   1,
		PayloadLimit: 20480,
		IsPublic:     false,
		CreatedBy:    creator,
	})
	if err != nil {
		t.Fatalf("create tool for tenant B failed: %v", err)
	}

	listA, err := repo.ListTools(ctx, tenantA)
	if err != nil {
		t.Fatalf("list tools for tenant A failed: %v", err)
	}
	if len(listA) != 1 || listA[0].Name != "tool-a" {
		t.Fatalf("tenant A should see only tool-a, got: %+v", listA)
	}
	if listA[0].Version == nil || listA[0].Version.Version != "1.0.0" {
		t.Fatalf("tenant A expected version metadata, got: %+v", listA[0].Version)
	}

	listB, err := repo.ListTools(ctx, tenantB)
	if err != nil {
		t.Fatalf("list tools for tenant B failed: %v", err)
	}
	if len(listB) != 1 || listB[0].Name != "tool-b" {
		t.Fatalf("tenant B should see only tool-b, got: %+v", listB)
	}
}

func TestCreateToolConflict(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)

	repo := New(pool)
	tenant := "11111111-1111-1111-1111-111111111111"
	creator := uuid.New()

	params := CreateToolParams{
		Name:         "duplicate-tool",
		DisplayName:  "Duplicate Tool",
		Description:  "First insert",
		Category:     nil,
		Tags:         []string{"dup"},
		Version:      "0.1.0",
		Status:       "draft",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
		Definition:   json.RawMessage(`{"kind":"http"}`),
		Allowlist:    json.RawMessage(`{"hosts":["dup.local"],"protocols":["https"]}`),
		Metadata:     json.RawMessage(`{"payload_bytes_limit":1024}`),
		TimeoutSecs:  20,
		MaxRetries:   1,
		PayloadLimit: 1024,
		IsPublic:     false,
		CreatedBy:    creator,
	}

	if _, err := repo.CreateTool(ctx, tenant, params); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	if _, err := repo.CreateTool(ctx, tenant, params); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate name, got: %v", err)
	}

	tools, err := repo.ListTools(ctx, tenant)
	if err != nil {
		t.Fatalf("list after conflict failed: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected single tool after conflict, got %d", len(tools))
	}
}

func TestPolicyRulesRespectRLSAndOrdering(t *testing.T) {
	ctx := context.Background()
	pool := setupTestDB(t)

	repo := New(pool)
	tenantA := "11111111-1111-1111-1111-111111111111"
	tenantB := "22222222-2222-2222-2222-222222222222"
	creator := uuid.New()

	// tenant A: two rules, higher weight deny should win
	_, err := repo.CreatePolicyRule(ctx, tenantA, CreatePolicyRuleParams{
		TenantID:  uuidPtr(tenantA),
		Effect:    "allow",
		Scopes:    []string{"write"},
		Roles:     []string{"admin"},
		Weight:    1,
		CreatedBy: creator,
	})
	if err != nil {
		t.Fatalf("create allow rule tenant A: %v", err)
	}
	_, err = repo.CreatePolicyRule(ctx, tenantA, CreatePolicyRuleParams{
		TenantID:  uuidPtr(tenantA),
		Effect:    "deny",
		Scopes:    []string{"write"},
		Roles:     []string{"admin"},
		Weight:    5,
		CreatedBy: creator,
	})
	if err != nil {
		t.Fatalf("create deny rule tenant A: %v", err)
	}

	// tenant B: single allow
	_, err = repo.CreatePolicyRule(ctx, tenantB, CreatePolicyRuleParams{
		TenantID:  uuidPtr(tenantB),
		Effect:    "allow",
		Scopes:    []string{"read"},
		Roles:     []string{"user"},
		Weight:    3,
		CreatedBy: creator,
	})
	if err != nil {
		t.Fatalf("create rule tenant B: %v", err)
	}

	rulesA, err := repo.ListPolicyRules(ctx, tenantA)
	if err != nil {
		t.Fatalf("list rules tenant A: %v", err)
	}
	if len(rulesA) != 2 {
		t.Fatalf("expected 2 rules for tenant A, got %d", len(rulesA))
	}
	if rulesA[0].Effect != "deny" || rulesA[0].Weight != 5 {
		t.Fatalf("expected deny rule first by weight, got %+v", rulesA[0])
	}

	rulesB, err := repo.ListPolicyRules(ctx, tenantB)
	if err != nil {
		t.Fatalf("list rules tenant B: %v", err)
	}
	if len(rulesB) != 1 || rulesB[0].Effect != "allow" {
		t.Fatalf("tenant B should see its allow rule only, got %+v", rulesB)
	}
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	dbURL := os.Getenv("TOOLS_MANAGER_TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultTestDBURL
	}

	ensureRoleAndMigrate(t, ctx, dbURL)

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

func ensureRoleAndMigrate(t *testing.T, ctx context.Context, dbURL string) {
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

	downSQL := readSQL(t, "000001_create_catalog.down.sql")
	if _, err := conn.Exec(ctx, downSQL); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}

	upSQL := readSQL(t, "000001_create_catalog.up.sql")
	if _, err := conn.Exec(ctx, upSQL); err != nil {
		t.Fatalf("apply up migration: %v", err)
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

func readSQL(t *testing.T, fileName string) string {
	t.Helper()

	path := filepath.Join("..", "..", "migrations", fileName)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", fileName, err)
	}
	return string(content)
}

func strPtr(s string) *string {
	return &s
}

func uuidPtr(s string) *uuid.UUID {
	id, _ := uuid.Parse(s)
	return &id
}
