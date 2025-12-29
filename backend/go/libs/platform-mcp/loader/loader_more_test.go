package loader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromFileBlocksTraversalWithoutRoot(t *testing.T) {
	_, err := LoadFromFile(context.Background(), filepath.Join("..", "tools.json"), "t1")
	if err == nil {
		t.Fatalf("expected traversal error without MCP_TOOLS_ROOT")
	}
}

func TestLoadFromFileRejectsNonRegularFile(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "fifo")
	if err := os.Mkdir(socketPath, 0o755); err != nil {
		t.Fatalf("setup dir: %v", err)
	}

	if _, err := LoadFromFile(context.Background(), socketPath, "t1"); err == nil {
		t.Fatalf("expected error for non-regular file")
	}
}

func TestLoadFromFileWithInvalidToolFailsValidation(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "tools.json")
	if err := os.WriteFile(filePath, []byte(`[{"name":"","version":"1.0.0"}]`), 0o600); err != nil {
		t.Fatalf("write tools: %v", err)
	}

	if _, err := LoadFromFile(context.Background(), filePath, "t1"); err == nil {
		t.Fatalf("expected validation error for tool without name")
	}
}

func TestLoadFromFileMissingPathErrors(t *testing.T) {
	if _, err := LoadFromFile(context.Background(), "missing.json", "t1"); err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestResolveToolsPathRootExactFile(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "tools.json")
	if err := os.WriteFile(filePath, []byte(`[]`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("MCP_TOOLS_ROOT", root)

	if _, err := LoadFromFile(context.Background(), root, "tenant"); err == nil {
		t.Fatalf("expected error when path is directory not file")
	}
}

func TestLoadFromFileAbsoluteWithinRoot(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "tools.json")
	if err := os.WriteFile(filePath, []byte(`[{"name":"echo","version":"1.0.0","input_schema":{},"output_schema":{}}]`), 0o600); err != nil {
		t.Fatalf("write tools: %v", err)
	}
	t.Setenv("MCP_TOOLS_ROOT", root)

	tools, err := LoadFromFile(context.Background(), filePath, "tenant-abs")
	if err != nil {
		t.Fatalf("load absolute: %v", err)
	}
	if len(tools) != 1 || tools[0].TenantID != "tenant-abs" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestLoadFromFileRelativeWithinRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tools.json"), []byte(`[{"name":"echo","version":"1.0.0","input_schema":{},"output_schema":{}}]`), 0o600); err != nil {
		t.Fatalf("write tools: %v", err)
	}
	t.Setenv("MCP_TOOLS_ROOT", root)

	tools, err := LoadFromFile(context.Background(), "tools.json", "tenant-rel")
	if err != nil {
		t.Fatalf("load relative: %v", err)
	}
	if len(tools) != 1 || tools[0].TenantID != "tenant-rel" {
		t.Fatalf("unexpected tools: %+v", tools)
	}
}

func TestResolveToolsPathRootAbsError(t *testing.T) {
	originalAbs := filepathAbs
	defer func() { filepathAbs = originalAbs }()

	filepathAbs = func(string) (string, error) { return "", errors.New("abs fail root") }
	t.Setenv("MCP_TOOLS_ROOT", t.TempDir())

	if _, err := resolveToolsPath("tools.json"); err == nil || !strings.Contains(err.Error(), "resolve MCP_TOOLS_ROOT") {
		t.Fatalf("expected root abs error, got %v", err)
	}
}

func TestResolveToolsPathCandidateAbsError(t *testing.T) {
	originalAbs := filepathAbs
	defer func() { filepathAbs = originalAbs }()

	calls := 0
	filepathAbs = func(path string) (string, error) {
		calls++
		if calls == 1 {
			return path, nil // rootAbs resolution succeeds
		}
		return "", errors.New("abs fail candidate")
	}
	t.Setenv("MCP_TOOLS_ROOT", t.TempDir())

	if _, err := resolveToolsPath("tools.json"); err == nil || !strings.Contains(err.Error(), "resolve path") {
		t.Fatalf("expected candidate abs error, got %v", err)
	}
}

func TestResolveToolsPathAbsErrorWithoutRoot(t *testing.T) {
	originalAbs := filepathAbs
	defer func() { filepathAbs = originalAbs }()

	filepathAbs = func(string) (string, error) { return "", errors.New("abs fail") }

	if _, err := resolveToolsPath("tools.json"); err == nil || !strings.Contains(err.Error(), "resolve path") {
		t.Fatalf("expected abs error, got %v", err)
	}
}

func TestLoadFromFileEmptyPath(t *testing.T) {
	if _, err := LoadFromFile(context.Background(), "", "t1"); err == nil {
		t.Fatalf("expected error for empty path")
	}
}

func TestLoadFromFileAbsoluteOutsideRootBlocked(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MCP_TOOLS_ROOT", root)

	outDir := filepath.Join(root, "..", "outside")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	outFile := filepath.Join(outDir, "tools.json")
	if err := os.WriteFile(outFile, []byte(`[{"name":"echo","version":"1.0.0","input_schema":{},"output_schema":{}}]`), 0o600); err != nil {
		t.Fatalf("write tools: %v", err)
	}

	if _, err := LoadFromFile(context.Background(), outFile, "tenant-x"); err == nil {
		t.Fatalf("expected error for absolute path outside MCP_TOOLS_ROOT")
	}
}

func TestLoadFromFileInvalidJSONErrors(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "tools.json")
	if err := os.WriteFile(filePath, []byte(`not-json`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := LoadFromFile(context.Background(), filePath, "t1"); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestLoadFromFileUnreadableFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "tools.json")
	if err := os.WriteFile(filePath, []byte(`[{}]`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(filePath, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	if _, err := LoadFromFile(context.Background(), filePath, "t1"); err == nil {
		t.Fatalf("expected read error for unreadable file")
	}
}

func TestLoadFromFileWrapperValidationError(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "tools.json")
	content := []byte(`{"tools":[{"name":"","version":"1.0.0"}]}`)
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := LoadFromFile(context.Background(), filePath, "t1"); err == nil {
		t.Fatalf("expected validation error for wrapper tool")
	}
}

func TestResolveToolsPathDotsOnly(t *testing.T) {
	if _, err := LoadFromFile(context.Background(), "..", "t1"); err == nil {
		t.Fatalf("expected traversal error for parent directory path")
	}
}
