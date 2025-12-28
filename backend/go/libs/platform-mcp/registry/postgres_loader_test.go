package registry

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestPostgresLoaderList(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	rows := mock.NewRows([]string{"name", "version", "display_name", "summary", "description", "input_schema", "output_schema", "scopes", "tags", "allow_hosts", "idempotency_key", "max_body_bytes", "max_duration_ms", "deprecated", "etag", "updated_at"}).
		AddRow("echo", "1.0.0", "Echo", "", "", json.RawMessage(`{"type":"object"}`), json.RawMessage(`{"type":"object"}`), []string{"tool:run"}, []string{"util"}, []string{"api.example.com"}, "", int64(1024), int64(5000), false, "etag1", time.Now())

	mock.ExpectQuery("FROM tools").WithArgs("t1").WillReturnRows(rows)

	loader := NewPostgresLoader(mock)
	tools, err := loader.ListTools(context.Background(), "t1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tools) != 1 || tools[0].TenantID != "t1" {
		t.Fatalf("unexpected tools: %+v", tools)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPostgresLoaderDescribe(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	row := mock.NewRows([]string{"name", "version", "display_name", "summary", "description", "input_schema", "output_schema", "scopes", "tags", "allow_hosts", "idempotency_key", "max_body_bytes", "max_duration_ms", "deprecated", "etag", "updated_at"}).
		AddRow("echo", "1.0.0", "Echo", "", "", json.RawMessage(`{"type":"object"}`), json.RawMessage(`{"type":"object"}`), []string{"tool:run"}, []string{"util"}, []string{"api.example.com"}, "", int64(1024), int64(5000), false, "etag1", time.Now())

	mock.ExpectQuery("FROM tools").WithArgs("t1", "echo").WillReturnRows(row)

	loader := NewPostgresLoader(mock)
	tool, err := loader.DescribeTool(context.Background(), "t1", "echo")
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	if tool.Name != "echo" || tool.TenantID != "t1" {
		t.Fatalf("unexpected tool: %+v", tool)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
