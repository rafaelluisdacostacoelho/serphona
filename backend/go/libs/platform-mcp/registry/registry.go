package registry

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// Registry exposes tool lookups for a tenant.
type Registry interface {
	ListTools(ctx context.Context, tenantID string) ([]protocol.Tool, error)
	DescribeTool(ctx context.Context, tenantID, name string) (protocol.Tool, error)
	// ETag-aware helpers
	ListToolsWithETag(ctx context.Context, tenantID, ifNoneMatch string) (etag string, tools []protocol.Tool, notModified bool, err error)
	DescribeToolWithETag(ctx context.Context, tenantID, name, ifNoneMatch string) (etag string, tool protocol.Tool, notModified bool, err error)
}

// MemoryRegistry is a simple multi-tenant in-memory registry with ETag-like versioning.
type MemoryRegistry struct {
	mu sync.RWMutex

	data map[string]map[string]protocol.Tool // tenant -> tool name -> Tool
}

// NewMemoryRegistry creates a new empty registry.
func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{data: make(map[string]map[string]protocol.Tool)}
}

// Upsert inserts or updates a tool for a tenant.
func (r *MemoryRegistry) Upsert(t protocol.Tool) error {
	if err := protocol.ValidateTool(t); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	byTenant, ok := r.data[t.TenantID]
	if !ok {
		byTenant = make(map[string]protocol.Tool)
		r.data[t.TenantID] = byTenant
	}

	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = time.Now().UTC()
	}
	if t.ETag == "" {
		t.ETag = t.UpdatedAt.Format(time.RFC3339Nano)
	}
	byTenant[strings.ToLower(t.Name)] = t
	return nil
}

// ListTools lists all tools for a tenant.
func (r *MemoryRegistry) ListTools(_ context.Context, tenantID string) ([]protocol.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	byTenant, ok := r.data[tenantID]
	if !ok {
		return []protocol.Tool{}, nil
	}

	out := make([]protocol.Tool, 0, len(byTenant))
	for _, t := range byTenant {
		out = append(out, t)
	}
	return out, nil
}

// DescribeTool fetches a tool by tenant and name.
func (r *MemoryRegistry) DescribeTool(_ context.Context, tenantID, name string) (protocol.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	byTenant, ok := r.data[tenantID]
	if !ok {
		return protocol.Tool{}, errors.New("tool not found")
	}
	t, ok := byTenant[strings.ToLower(name)]
	if !ok {
		return protocol.Tool{}, errors.New("tool not found")
	}
	return t, nil
}

// ListToolsWithETag returns tools and an aggregate ETag; notModified signals If-None-Match hit.
func (r *MemoryRegistry) ListToolsWithETag(ctx context.Context, tenantID, ifNoneMatch string) (string, []protocol.Tool, bool, error) {
	tools, err := r.ListTools(ctx, tenantID)
	if err != nil {
		return "", nil, false, err
	}
	etag := listETag(tools)
	if etag != "" && ifNoneMatch != "" && ifNoneMatch == etag {
		return etag, nil, true, nil
	}
	return etag, tools, false, nil
}

// DescribeToolWithETag returns a tool and its ETag; notModified signals If-None-Match hit.
func (r *MemoryRegistry) DescribeToolWithETag(ctx context.Context, tenantID, name, ifNoneMatch string) (string, protocol.Tool, bool, error) {
	tool, err := r.DescribeTool(ctx, tenantID, name)
	if err != nil {
		return "", protocol.Tool{}, false, err
	}
	etag := tool.ETag
	if etag != "" && ifNoneMatch != "" && ifNoneMatch == etag {
		return etag, protocol.Tool{}, true, nil
	}
	return etag, tool, false, nil
}

func listETag(tools []protocol.Tool) string {
	if len(tools) == 0 {
		return ""
	}
	latest := tools[0].ETag
	latestTime := tools[0].UpdatedAt
	for _, t := range tools[1:] {
		if t.UpdatedAt.After(latestTime) {
			latest = t.ETag
			latestTime = t.UpdatedAt
		}
	}
	return latest
}
