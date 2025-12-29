package registry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// AllowListProvider returns the set of tool names allowed for a tenant.
type AllowListProvider interface {
	AllowedTools(ctx context.Context, tenantID string) (map[string]struct{}, error)
}

// AllowListRegistry wraps a Registry enforcing per-tenant allow-lists.
type AllowListRegistry struct {
	base  Registry
	allow AllowListProvider
}

// NewAllowListRegistry builds a registry that filters tools by allow-list.
func NewAllowListRegistry(base Registry, allow AllowListProvider) *AllowListRegistry {
	return &AllowListRegistry{base: base, allow: allow}
}

// ListTools returns only tools present in the tenant allow-list.
func (r *AllowListRegistry) ListTools(ctx context.Context, tenantID string) ([]protocol.Tool, error) {
	_, tools, _, err := r.ListToolsWithETag(ctx, tenantID, "")
	return tools, err
}

// DescribeTool returns a tool if allowed for the tenant.
func (r *AllowListRegistry) DescribeTool(ctx context.Context, tenantID, name string) (protocol.Tool, error) {
	_, tool, _, err := r.DescribeToolWithETag(ctx, tenantID, name, "")
	return tool, err
}

// ListToolsWithETag filters the base registry list and recomputes ETag after filtering.
func (r *AllowListRegistry) ListToolsWithETag(ctx context.Context, tenantID, ifNoneMatch string) (string, []protocol.Tool, bool, error) {
	allowSet, err := r.allow.AllowedTools(ctx, tenantID)
	if err != nil {
		return "", nil, false, err
	}
	if len(allowSet) == 0 {
		return "", nil, false, fmt.Errorf("allow-list empty for tenant %s", tenantID)
	}

	baseETag, tools, notModified, err := r.base.ListToolsWithETag(ctx, tenantID, "")
	if err != nil {
		return "", nil, false, err
	}
	_ = baseETag // ignored because we recompute after filtering

	filtered := filterAllowed(tools, allowSet)
	etag := listETag(filtered)
	if etag != "" && ifNoneMatch != "" && etag == ifNoneMatch {
		return etag, nil, true, nil
	}
	return etag, filtered, notModified && len(filtered) == 0, nil
}

// DescribeToolWithETag fetches a tool if it is allow-listed.
func (r *AllowListRegistry) DescribeToolWithETag(ctx context.Context, tenantID, name, ifNoneMatch string) (string, protocol.Tool, bool, error) {
	allowSet, err := r.allow.AllowedTools(ctx, tenantID)
	if err != nil {
		return "", protocol.Tool{}, false, err
	}
	if len(allowSet) == 0 {
		return "", protocol.Tool{}, false, fmt.Errorf("allow-list empty for tenant %s", tenantID)
	}

	if !isAllowed(name, allowSet) {
		return "", protocol.Tool{}, false, errors.New("tool not allowed")
	}

	etag, tool, notModified, err := r.base.DescribeToolWithETag(ctx, tenantID, name, ifNoneMatch)
	if err != nil {
		return "", protocol.Tool{}, false, err
	}
	return etag, tool, notModified, nil
}

func filterAllowed(tools []protocol.Tool, allowSet map[string]struct{}) []protocol.Tool {
	out := make([]protocol.Tool, 0, len(tools))
	for _, t := range tools {
		if isAllowed(t.Name, allowSet) {
			out = append(out, t)
		}
	}
	return out
}

func isAllowed(name string, allowSet map[string]struct{}) bool {
	_, ok := allowSet[strings.ToLower(name)]
	return ok
}

// StaticAllowList is a simple AllowListProvider backed by an in-memory map.
type StaticAllowList map[string][]string // tenant -> tool names

// AllowedTools returns the allow-list set for a tenant.
func (s StaticAllowList) AllowedTools(_ context.Context, tenantID string) (map[string]struct{}, error) {
	names := s[tenantID]
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[strings.ToLower(n)] = struct{}{}
	}
	return set, nil
}
