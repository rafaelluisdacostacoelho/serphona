package registry

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestMemoryRegistry(t *testing.T) {
	r := NewMemoryRegistry()
	tool := protocol.Tool{
		Name:         "echo",
		Version:      "1.0.0",
		TenantID:     "t1",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
	}
	if err := r.Upsert(tool); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	list, err := r.ListTools(context.Background(), "t1")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(list))
	}

	_, err = r.DescribeTool(context.Background(), "t1", "echo")
	if err != nil {
		t.Fatalf("describe failed: %v", err)
	}
}

func TestMemoryRegistryETag(t *testing.T) {
	r := NewMemoryRegistry()
	tool := protocol.Tool{
		Name:         "echo",
		Version:      "1.0.0",
		TenantID:     "t1",
		InputSchema:  json.RawMessage(`{"type":"object"}`),
		OutputSchema: json.RawMessage(`{"type":"object"}`),
	}
	if err := r.Upsert(tool); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	etag, _, notModified, err := r.DescribeToolWithETag(context.Background(), "t1", "echo", "")
	if err != nil {
		t.Fatalf("describe etag failed: %v", err)
	}
	if etag == "" {
		t.Fatalf("expected etag")
	}
	if notModified {
		t.Fatalf("should not be notModified on first call")
	}

	_, _, notModified, err = r.DescribeToolWithETag(context.Background(), "t1", "echo", etag)
	if err != nil {
		t.Fatalf("describe etag failed: %v", err)
	}
	if !notModified {
		t.Fatalf("expected notModified when etag matches")
	}
}
