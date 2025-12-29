package loader

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestLoadFromFileArray(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "tools-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	tools := []protocol.Tool{{Name: "echo", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`)}}
	if err := json.NewEncoder(tmp).Encode(tools); err != nil {
		t.Fatalf("encode: %v", err)
	}
	_ = tmp.Close()

	loaded, err := LoadFromFile(context.Background(), tmp.Name(), "t1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded) != 1 || loaded[0].TenantID != "t1" {
		t.Fatalf("expected tenant applied, got %+v", loaded)
	}
}

func TestLoadFromFileWrapper(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "tools-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	wrapper := struct {
		Tools []protocol.Tool `json:"tools"`
	}{Tools: []protocol.Tool{{Name: "math.add", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`)}}}
	if err := json.NewEncoder(tmp).Encode(wrapper); err != nil {
		t.Fatalf("encode: %v", err)
	}
	_ = tmp.Close()

	loaded, err := LoadFromFile(context.Background(), tmp.Name(), "tenant-x")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded[0].TenantID != "tenant-x" {
		t.Fatalf("tenant not applied: %+v", loaded[0])
	}
}

func TestLoadFromFileMissingTenant(t *testing.T) {
	_, err := LoadFromFile(context.Background(), "does-not-exist.json", "")
	if err == nil {
		t.Fatalf("expected error for missing tenant_id")
	}
}

func TestLoadFromFileRespectsRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MCP_TOOLS_ROOT", root)

	filePath := filepath.Join(root, "tools.json")
	tools := []protocol.Tool{{Name: "echo", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`)}}
	if err := os.WriteFile(filePath, mustJSON(tools), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	loaded, err := LoadFromFile(context.Background(), "tools.json", "tenant-root")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := loaded[0].TenantID; got != "tenant-root" {
		t.Fatalf("tenant not applied: %s", got)
	}
}

func TestLoadFromFileBlocksEscape(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MCP_TOOLS_ROOT", root)

	if _, err := LoadFromFile(context.Background(), filepath.Join("..", "tools.json"), "tenant-x"); err == nil {
		t.Fatalf("expected error for path escaping MCP_TOOLS_ROOT")
	}
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
