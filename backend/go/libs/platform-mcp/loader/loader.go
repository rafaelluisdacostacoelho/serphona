package loader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

var filepathAbs = filepath.Abs

// LoadFromFile loads tools for a tenant from a JSON file. Accepts either an array of Tool
// or an object {"tools": [...]}. TenantID is enforced/overridden on all entries.
func LoadFromFile(ctx context.Context, path string, tenantID string) ([]protocol.Tool, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}
	safePath, err := resolveToolsPath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(safePath) // #nosec G304 path validated in resolveToolsPath
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Try array first
	var arr []protocol.Tool
	if err := json.Unmarshal(data, &arr); err == nil {
		return normalizeAndValidate(ctx, arr, tenantID)
	}

	// Try wrapper object
	var wrapper struct {
		Tools []protocol.Tool `json:"tools"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal tools: %w", err)
	}
	return normalizeAndValidate(ctx, wrapper.Tools, tenantID)
}

func normalizeAndValidate(_ context.Context, tools []protocol.Tool, tenantID string) ([]protocol.Tool, error) {
	out := make([]protocol.Tool, 0, len(tools))
	for _, t := range tools {
		t.TenantID = tenantID
		if err := protocol.ValidateTool(t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func resolveToolsPath(rawPath string) (string, error) {
	if rawPath == "" {
		return "", errors.New("path is required")
	}

	cleaned := filepath.Clean(rawPath)
	root := os.Getenv("MCP_TOOLS_ROOT")
	if root != "" {
		rootAbs, err := filepathAbs(filepath.Clean(root))
		if err != nil {
			return "", fmt.Errorf("resolve MCP_TOOLS_ROOT: %w", err)
		}

		candidate := cleaned
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(rootAbs, candidate)
		}
		candidate, err = filepathAbs(candidate)
		if err != nil {
			return "", fmt.Errorf("resolve path: %w", err)
		}

		if !strings.HasPrefix(candidate, rootAbs+string(filepath.Separator)) && candidate != rootAbs {
			return "", fmt.Errorf("path %q escapes configured root %q", candidate, rootAbs)
		}

		return candidate, ensureRegularFile(candidate)
	}

	if strings.Contains(cleaned, ".."+string(filepath.Separator)) || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == ".." {
		return "", fmt.Errorf("path traversal not allowed: %q", rawPath)
	}

	absPath, err := filepathAbs(cleaned)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	return absPath, ensureRegularFile(absPath)
}

func ensureRegularFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %q: %w", path, err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("path %q is not a regular file", path)
	}

	return nil
}
