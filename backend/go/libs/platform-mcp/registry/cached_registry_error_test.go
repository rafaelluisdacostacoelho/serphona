package registry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type failingLoader struct{}

func (f failingLoader) ListTools(context.Context, string) ([]protocol.Tool, error) {
	return nil, errors.New("boom")
}
func (f failingLoader) DescribeTool(context.Context, string, string) (protocol.Tool, error) {
	return protocol.Tool{}, errors.New("boom")
}

func TestCachedRegistryPropagatesLoaderError(t *testing.T) {
	reg := NewCachedRegistry(failingLoader{}, time.Second)
	if _, _, _, err := reg.ListToolsWithETag(context.Background(), "t1", ""); err == nil {
		t.Fatalf("expected error from loader")
	}
}
