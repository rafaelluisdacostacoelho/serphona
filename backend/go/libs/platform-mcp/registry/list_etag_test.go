package registry

import (
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestListETagChoosesLatest(t *testing.T) {
	now := time.Now().UTC()
	tools := []protocol.Tool{
		{Name: "a", TenantID: "t1", ETag: "e1", UpdatedAt: now.Add(-time.Minute)},
		{Name: "b", TenantID: "t1", ETag: "e2", UpdatedAt: now},
	}
	if etag := listETag(tools); etag != "e2" {
		t.Fatalf("expected latest etag e2, got %s", etag)
	}

	if etag := listETag(nil); etag != "" {
		t.Fatalf("expected empty etag for empty list")
	}
}
