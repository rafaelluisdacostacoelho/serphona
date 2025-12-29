package registry

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestAllowListRegistry_ListFiltersTools(t *testing.T) {
	base := NewMemoryRegistry()
	toolAllowed := protocol.Tool{TenantID: "t1", Name: "echo", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`), ETag: "e1", UpdatedAt: time.Now().UTC()}
	toolBlocked := protocol.Tool{TenantID: "t1", Name: "math.add", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`), ETag: "e2", UpdatedAt: time.Now().UTC()}
	if err := base.Upsert(toolAllowed); err != nil {
		t.Fatalf("upsert allowed: %v", err)
	}
	if err := base.Upsert(toolBlocked); err != nil {
		t.Fatalf("upsert blocked: %v", err)
	}

	allow := StaticAllowList{"t1": {"echo"}}
	reg := NewAllowListRegistry(base, allow)

	etag, tools, notModified, err := reg.ListToolsWithETag(context.Background(), "t1", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if notModified {
		t.Fatalf("unexpected notModified")
	}
	if etag == "" {
		t.Fatalf("expected etag")
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("expected only allowed tool, got %+v", tools)
	}

	_, _, notModified, err = reg.ListToolsWithETag(context.Background(), "t1", etag)
	if err != nil {
		t.Fatalf("list if-none-match: %v", err)
	}
	if !notModified {
		t.Fatalf("expected notModified when etag matches filtered list")
	}
}

func TestAllowListRegistry_DescribeDeniesUnlisted(t *testing.T) {
	base := NewMemoryRegistry()
	tool := protocol.Tool{TenantID: "t1", Name: "echo", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`)}
	if err := base.Upsert(tool); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	allow := StaticAllowList{"t1": {"echo"}}
	reg := NewAllowListRegistry(base, allow)

	if _, _, _, err := reg.DescribeToolWithETag(context.Background(), "t1", "math.add", ""); err == nil {
		t.Fatalf("expected denial for unlisted tool")
	}

	_, toolOut, notModified, err := reg.DescribeToolWithETag(context.Background(), "t1", "echo", "")
	if err != nil {
		t.Fatalf("describe allowed: %v", err)
	}
	if notModified || toolOut.Name != "echo" {
		t.Fatalf("unexpected describe result: notMod=%v tool=%+v", notModified, toolOut)
	}
}

func TestAllowListRegistry_EmptyAllowListDenies(t *testing.T) {
	base := NewMemoryRegistry()
	tool := protocol.Tool{TenantID: "t1", Name: "echo", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`)}
	if err := base.Upsert(tool); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	allow := StaticAllowList{"t1": {}}
	reg := NewAllowListRegistry(base, allow)

	if _, _, _, err := reg.ListToolsWithETag(context.Background(), "t1", ""); err == nil {
		t.Fatalf("expected error for empty allow-list")
	}
}

func TestAllowListRegistryDescribeNotModified(t *testing.T) {
	base := NewMemoryRegistry()
	tool := protocol.Tool{TenantID: "t1", Name: "echo", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`), UpdatedAt: time.Now().UTC(), ETag: "e1"}
	if err := base.Upsert(tool); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	reg := NewAllowListRegistry(base, StaticAllowList{"t1": {"echo"}})

	etag, toolOut, notModified, err := reg.DescribeToolWithETag(context.Background(), "t1", "echo", "e1")
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	if !notModified || etag != "e1" || toolOut.Name != "" {
		t.Fatalf("expected not modified with empty tool, got etag=%s tool=%+v notMod=%v", etag, toolOut, notModified)
	}
}

func TestAllowListRegistryListFilteredEmpty(t *testing.T) {
	base := NewMemoryRegistry()
	blocked := protocol.Tool{TenantID: "t1", Name: "blocked", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`), UpdatedAt: time.Now().UTC(), ETag: "e1"}
	if err := base.Upsert(blocked); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	reg := NewAllowListRegistry(base, StaticAllowList{"t1": {"other"}})

	etag, tools, notModified, err := reg.ListToolsWithETag(context.Background(), "t1", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if notModified || etag != "" || len(tools) != 0 {
		t.Fatalf("expected filtered empty list, got etag=%s notMod=%v tools=%v", etag, notModified, tools)
	}
}

func TestAllowListRegistryListFilteredEmptyNotModified(t *testing.T) {
	base := NewMemoryRegistry()
	blocked := protocol.Tool{TenantID: "t1", Name: "blocked", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`), UpdatedAt: time.Now().UTC(), ETag: "e1"}
	if err := base.Upsert(blocked); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	reg := NewAllowListRegistry(base, StaticAllowList{"t1": {"other"}})

	baseETag, _, _, err := base.ListToolsWithETag(context.Background(), "t1", "")
	if err != nil {
		t.Fatalf("base list: %v", err)
	}
	etag, tools, notModified, err := reg.ListToolsWithETag(context.Background(), "t1", baseETag)
	if err != nil {
		t.Fatalf("list with if-none-match: %v", err)
	}
	if notModified || etag != "" || len(tools) != 0 {
		t.Fatalf("expected filtered empty list without notModified hit, got etag=%s notMod=%v tools=%v", etag, notModified, tools)
	}
}

func TestAllowListRegistryWrappers(t *testing.T) {
	base := NewMemoryRegistry()
	tool := protocol.Tool{TenantID: "t1", Name: "echo", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`), UpdatedAt: time.Now().UTC(), ETag: "e1"}
	if err := base.Upsert(tool); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	reg := NewAllowListRegistry(base, StaticAllowList{"t1": {"echo"}})

	tools, err := reg.ListTools(context.Background(), "t1")
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("expected allowed tool via wrapper, got %+v", tools)
	}

	toolOut, err := reg.DescribeTool(context.Background(), "t1", "echo")
	if err != nil || toolOut.Name != "echo" {
		t.Fatalf("describe via wrapper failed: %v %+v", err, toolOut)
	}
}

type errorAllowList struct{}

func (errorAllowList) AllowedTools(context.Context, string) (map[string]struct{}, error) {
	return nil, errors.New("allowlist failed")
}

type errorRegistry struct {
	listErr     error
	describeErr error
}

func (e errorRegistry) ListTools(context.Context, string) ([]protocol.Tool, error) {
	return nil, e.listErr
}
func (e errorRegistry) DescribeTool(context.Context, string, string) (protocol.Tool, error) {
	return protocol.Tool{}, e.describeErr
}
func (e errorRegistry) ListToolsWithETag(context.Context, string, string) (string, []protocol.Tool, bool, error) {
	return "", nil, false, e.listErr
}
func (e errorRegistry) DescribeToolWithETag(context.Context, string, string, string) (string, protocol.Tool, bool, error) {
	return "", protocol.Tool{}, false, e.describeErr
}

func TestAllowListRegistryDescribeAllowProviderError(t *testing.T) {
	base := NewMemoryRegistry()
	reg := NewAllowListRegistry(base, errorAllowList{})

	if _, _, _, err := reg.DescribeToolWithETag(context.Background(), "t1", "echo", ""); err == nil {
		t.Fatalf("expected error from allow provider")
	}
}

func TestAllowListRegistryDescribeEmptyAllowList(t *testing.T) {
	base := NewMemoryRegistry()
	tool := protocol.Tool{TenantID: "t1", Name: "echo", Version: "1.0.0", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`)}
	if err := base.Upsert(tool); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	reg := NewAllowListRegistry(base, StaticAllowList{"t1": {}})

	if _, _, _, err := reg.DescribeToolWithETag(context.Background(), "t1", "echo", ""); err == nil {
		t.Fatalf("expected error for empty allow-list on describe")
	}
}

func TestAllowListRegistryListProviderError(t *testing.T) {
	base := NewMemoryRegistry()
	reg := NewAllowListRegistry(base, errorAllowList{})

	if _, _, _, err := reg.ListToolsWithETag(context.Background(), "t1", ""); err == nil {
		t.Fatalf("expected error from allow provider on list")
	}
}

func TestAllowListRegistryBaseListError(t *testing.T) {
	base := errorRegistry{listErr: errors.New("list fail")}
	reg := NewAllowListRegistry(base, StaticAllowList{"t1": {"echo"}})

	if _, _, _, err := reg.ListToolsWithETag(context.Background(), "t1", ""); err == nil {
		t.Fatalf("expected base list error")
	}
}

func TestAllowListRegistryBaseDescribeError(t *testing.T) {
	base := errorRegistry{describeErr: errors.New("describe fail")}
	reg := NewAllowListRegistry(base, StaticAllowList{"t1": {"echo"}})

	if _, _, _, err := reg.DescribeToolWithETag(context.Background(), "t1", "echo", ""); err == nil {
		t.Fatalf("expected base describe error")
	}
}
