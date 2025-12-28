package loader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// LoadFromFile loads tools for a tenant from a JSON file. Accepts either an array of Tool
// or an object {"tools": [...]}. TenantID is enforced/overridden on all entries.
func LoadFromFile(ctx context.Context, path string, tenantID string) ([]protocol.Tool, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id is required")
	}
	data, err := os.ReadFile(path)
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
