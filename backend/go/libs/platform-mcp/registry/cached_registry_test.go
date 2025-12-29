package registry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type stubLoader struct {
	list          map[string][]protocol.Tool
	describe      map[string]protocol.Tool
	listCalls     int
	describeCalls int
	listErr       error
}

func (s *stubLoader) ListTools(_ context.Context, tenantID string) ([]protocol.Tool, error) {
	s.listCalls++
	if s.listErr != nil {
		return nil, s.listErr
	}
	tools := s.list[tenantID]
	cpy := make([]protocol.Tool, len(tools))
	copy(cpy, tools)
	return cpy, nil
}

func (s *stubLoader) DescribeTool(_ context.Context, tenantID, name string) (protocol.Tool, error) {
	s.describeCalls++
	key := tenantID + ":" + strings.ToLower(name)
	tool, ok := s.describe[key]
	if !ok {
		return protocol.Tool{}, errors.New("not found")
	}
	return tool, nil
}

func TestCachedRegistry_ListToolsWithETagAndTTL(t *testing.T) {
	ctx := context.Background()
	baseTime := time.Now().UTC()
	current := baseTime

	loader := &stubLoader{
		list: map[string][]protocol.Tool{
			"t1": {{Name: "Echo", TenantID: "t1", ETag: "e1", UpdatedAt: baseTime}},
		},
	}

	reg := NewCachedRegistry(loader, time.Minute)
	reg.now = func() time.Time { return current }

	etag, tools, notModified, err := reg.ListToolsWithETag(ctx, "t1", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if notModified || etag != "e1" || len(tools) != 1 {
		t.Fatalf("unexpected list result: etag=%s notMod=%v tools=%d", etag, notModified, len(tools))
	}
	if loader.listCalls != 1 {
		t.Fatalf("expected one list call, got %d", loader.listCalls)
	}

	etag, tools, notModified, err = reg.ListToolsWithETag(ctx, "t1", "e1")
	if err != nil {
		t.Fatalf("cached list: %v", err)
	}
	if !notModified || tools != nil || etag != "e1" {
		t.Fatalf("expected not modified from cache, got etag=%s notMod=%v tools=%v", etag, notModified, tools)
	}
	if loader.listCalls != 1 {
		t.Fatalf("expected no extra list call, got %d", loader.listCalls)
	}

	current = current.Add(2 * time.Minute)
	loader.list["t1"] = []protocol.Tool{{Name: "Echo", TenantID: "t1", ETag: "e2", UpdatedAt: current}}

	etag, tools, notModified, err = reg.ListToolsWithETag(ctx, "t1", "e1")
	if err != nil {
		t.Fatalf("refreshed list: %v", err)
	}
	if notModified || etag != "e2" || len(tools) != 1 {
		t.Fatalf("expected refreshed tools, got etag=%s notMod=%v tools=%d", etag, notModified, len(tools))
	}
	if loader.listCalls != 2 {
		t.Fatalf("expected refresh list call, got %d", loader.listCalls)
	}
}

func TestCachedRegistry_DescribeUsesCacheAndIfNoneMatch(t *testing.T) {
	ctx := context.Background()
	baseTime := time.Now().UTC()
	current := baseTime

	loader := &stubLoader{
		list: map[string][]protocol.Tool{
			"t1": {{Name: "Echo", TenantID: "t1", ETag: "e1", UpdatedAt: baseTime}},
		},
		describe: map[string]protocol.Tool{},
	}
	reg := NewCachedRegistry(loader, time.Minute)
	reg.now = func() time.Time { return current }

	etag, tool, notModified, err := reg.DescribeToolWithETag(ctx, "t1", "Echo", "")
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	if notModified || etag != "e1" || tool.Name != "Echo" {
		t.Fatalf("unexpected describe result: etag=%s notMod=%v tool=%+v", etag, notModified, tool)
	}
	if loader.listCalls != 1 || loader.describeCalls != 0 {
		t.Fatalf("expected list=1 describe=0, got list=%d describe=%d", loader.listCalls, loader.describeCalls)
	}

	etag, tool, notModified, err = reg.DescribeToolWithETag(ctx, "t1", "Echo", "e1")
	if err != nil {
		t.Fatalf("describe cached: %v", err)
	}
	if !notModified || etag != "e1" || tool.Name != "" {
		t.Fatalf("expected not modified with empty tool, got etag=%s notMod=%v tool=%+v", etag, notModified, tool)
	}
	if loader.listCalls != 1 || loader.describeCalls != 0 {
		t.Fatalf("expected no extra loader calls, got list=%d describe=%d", loader.listCalls, loader.describeCalls)
	}
}

func TestCachedRegistry_FallbackDescribeWhenNotCached(t *testing.T) {
	ctx := context.Background()
	baseTime := time.Now().UTC()
	loader := &stubLoader{
		list: map[string][]protocol.Tool{
			"t1": {},
		},
		describe: map[string]protocol.Tool{
			"t1:echo": {Name: "Echo", TenantID: "t1", ETag: "e1", UpdatedAt: baseTime},
		},
	}
	reg := NewCachedRegistry(loader, time.Minute)

	etag, tool, notModified, err := reg.DescribeToolWithETag(ctx, "t1", "Echo", "")
	if err != nil {
		t.Fatalf("describe fallback: %v", err)
	}
	if notModified || etag != "e1" || tool.Name != "Echo" {
		t.Fatalf("unexpected fallback result: etag=%s notMod=%v tool=%+v", etag, notModified, tool)
	}
	if loader.listCalls != 1 || loader.describeCalls != 1 {
		t.Fatalf("expected loader list=1 describe=1, got list=%d describe=%d", loader.listCalls, loader.describeCalls)
	}

	// Should now be cached.
	etag, tool, notModified, err = reg.DescribeToolWithETag(ctx, "t1", "Echo", "e1")
	if err != nil {
		t.Fatalf("describe cached fallback: %v", err)
	}
	if !notModified || etag != "e1" {
		t.Fatalf("expected not modified after caching, got etag=%s notMod=%v", etag, notModified)
	}
	if loader.listCalls != 1 || loader.describeCalls != 1 {
		t.Fatalf("expected no new loader calls, got list=%d describe=%d", loader.listCalls, loader.describeCalls)
	}
}

func TestNewCachedRegistryAppliesDefaultTTL(t *testing.T) {
	reg := NewCachedRegistry(&stubLoader{}, 0)
	if reg.ttl <= 0 {
		t.Fatalf("expected default ttl to be set")
	}
}

func TestCachedRegistryWrappers(t *testing.T) {
	ctx := context.Background()
	loader := &stubLoader{list: map[string][]protocol.Tool{"t1": {{Name: "Echo", TenantID: "t1", ETag: "e1", UpdatedAt: time.Now().UTC()}}}}
	reg := NewCachedRegistry(loader, time.Minute)

	tools, err := reg.ListTools(ctx, "t1")
	if err != nil || len(tools) != 1 {
		t.Fatalf("list wrapper: err=%v tools=%v", err, tools)
	}

	tool, err := reg.DescribeTool(ctx, "t1", "echo")
	if err != nil || tool.Name != "Echo" {
		t.Fatalf("describe wrapper failed: %v %+v", err, tool)
	}
}

func TestCachedRegistryListDescribeErrorPropagation(t *testing.T) {
	ctx := context.Background()
	loader := &stubLoader{listErr: errors.New("list fail")}
	reg := NewCachedRegistry(loader, time.Minute)

	if _, _, _, err := reg.ListToolsWithETag(ctx, "t1", ""); err == nil {
		t.Fatalf("expected list error to propagate")
	}

	if _, _, _, err := reg.DescribeToolWithETag(ctx, "t1", "echo", ""); err == nil {
		t.Fatalf("expected describe to fail when ensureEntry errors")
	}
}

func TestCachedRegistryDescribeLoaderError(t *testing.T) {
	ctx := context.Background()
	loader := &stubLoader{list: map[string][]protocol.Tool{"t1": {}}, describe: map[string]protocol.Tool{}}
	reg := NewCachedRegistry(loader, time.Minute)

	if _, _, _, err := reg.DescribeToolWithETag(ctx, "t1", "missing", ""); err == nil {
		t.Fatalf("expected describe loader error")
	}
	if loader.listCalls != 1 || loader.describeCalls != 1 {
		t.Fatalf("unexpected loader calls: list=%d describe=%d", loader.listCalls, loader.describeCalls)
	}
}
