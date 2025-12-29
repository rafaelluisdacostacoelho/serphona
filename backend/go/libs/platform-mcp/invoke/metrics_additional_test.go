package invoke

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// noSinkObserver verifies we don't panic when sink is nil.
func TestMetricsObserverNilSink(t *testing.T) {
	var obs *MetricsObserver
	obs = NewMetricsObserver(nil)
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}
	obs.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventResult}, nil, time.Millisecond)
}

func TestMultiObserverFanout(t *testing.T) {
	sink := &stubSink{}
	obs1 := NewMetricsObserver(sink)
	obs2 := NewMetricsObserver(sink)
	multi := NewMultiObserver(obs1, obs2)

	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}
	multi.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventResult}, nil, 0)

	if len(sink.counters) != 2 {
		t.Fatalf("expected fanout to both observers, got %d", len(sink.counters))
	}
}

func TestClassifyOutcomeProgress(t *testing.T) {
	if out := classifyOutcome(protocol.InvocationEvent{Type: protocol.EventProgress}, nil); out != "progress" {
		t.Fatalf("expected progress, got %s", out)
	}
	if out := classifyOutcome(protocol.InvocationEvent{}, errors.New("boom")); out != "error" {
		t.Fatalf("expected error when invokeErr set, got %s", out)
	}
}
