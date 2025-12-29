package registry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestMemoryRegistryDescribeNotFound(t *testing.T) {
	r := NewMemoryRegistry()
	if _, err := r.DescribeTool(context.Background(), "t1", "missing"); err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestMemoryRegistryDescribeWithETagNotFound(t *testing.T) {
	r := NewMemoryRegistry()
	if _, _, _, err := r.DescribeToolWithETag(context.Background(), "t1", "missing", ""); err == nil {
		t.Fatalf("expected not found error from describe with etag")
	}
}

func TestMemoryRegistryListToolsWithETagNotModified(t *testing.T) {
	r := NewMemoryRegistry()
	tool := protocol.Tool{TenantID: "t1", Name: "echo", Version: "1", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`), ETag: "e1", UpdatedAt: time.Now().UTC()}
	_ = r.Upsert(tool)

	etag, tools, notModified, err := r.ListToolsWithETag(context.Background(), "t1", "")
	if err != nil || etag == "" || notModified || len(tools) != 1 {
		t.Fatalf("unexpected first call result")
	}

	etag2, tools2, notModified2, err := r.ListToolsWithETag(context.Background(), "t1", etag)
	if err != nil {
		t.Fatalf("second call err: %v", err)
	}
	if !notModified2 || etag2 != etag || tools2 != nil {
		t.Fatalf("expected not modified hit")
	}
}

func TestMemoryRegistryListToolsEmptyTenant(t *testing.T) {
	r := NewMemoryRegistry()
	etag, tools, notModified, err := r.ListToolsWithETag(context.Background(), "missing", "any")
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if etag != "" || notModified {
		t.Fatalf("expected empty etag and notModified false, got etag=%s notMod=%v", etag, notModified)
	}
	if len(tools) != 0 {
		t.Fatalf("expected empty tools, got %v", tools)
	}
}

func TestMemoryRegistryUpsertValidationError(t *testing.T) {
	r := NewMemoryRegistry()
	if err := r.Upsert(protocol.Tool{TenantID: "t1", Version: "1.0.0"}); err == nil {
		t.Fatalf("expected validation error for missing name")
	}
}

func TestMemoryRegistryDescribeToolMissingWithinTenant(t *testing.T) {
	r := NewMemoryRegistry()
	tool := protocol.Tool{TenantID: "t1", Name: "echo", Version: "1", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`)}
	if err := r.Upsert(tool); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if _, err := r.DescribeTool(context.Background(), "t1", "other"); err == nil {
		t.Fatalf("expected error for missing tool in tenant")
	}
}

func TestMemoryRegistryListToolsWithETagError(t *testing.T) {
	original := memoryListFunc
	defer func() { memoryListFunc = original }()

	memoryListFunc = func(*MemoryRegistry, string) ([]protocol.Tool, error) {
		return nil, errors.New("list fail")
	}

	r := NewMemoryRegistry()
	if _, _, _, err := r.ListToolsWithETag(context.Background(), "t1", ""); err == nil {
		t.Fatalf("expected error from list hook")
	}
}
