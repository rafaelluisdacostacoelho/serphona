package invoke

import (
	"context"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type stubSink struct {
	counters []map[string]string
	histos   []struct {
		labels map[string]string
		value  float64
	}
}

func (s *stubSink) IncCounter(_ string, labels map[string]string) {
	s.counters = append(s.counters, labels)
}

func (s *stubSink) ObserveHistogram(_ string, value float64, labels map[string]string) {
	s.histos = append(s.histos, struct {
		labels map[string]string
		value  float64
	}{labels: labels, value: value})
}

func TestMetricsObserverRecordsOutcome(t *testing.T) {
	sink := &stubSink{}
	obs := NewMetricsObserver(sink)
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventResult}, nil, 150*time.Millisecond)

	if len(sink.counters) != 1 || len(sink.histos) != 1 {
		t.Fatalf("expected metrics: %+v %+v", sink.counters, sink.histos)
	}
	if sink.counters[0]["outcome"] != "ok" || sink.counters[0]["tenant"] != "t1" || sink.counters[0]["tool"] != "echo" {
		t.Fatalf("unexpected labels: %+v", sink.counters[0])
	}
}

func TestMetricsObserverClassifiesCancel(t *testing.T) {
	sink := &stubSink{}
	obs := NewMetricsObserver(sink)
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventError, Error: &protocol.InvocationError{Code: "cancelled"}}, nil, 0)

	if sink.counters[0]["outcome"] != "cancelled" {
		t.Fatalf("expected cancelled, got %+v", sink.counters[0])
	}
}

func TestMetricsObserverEnricherAddsLabels(t *testing.T) {
	sink := &stubSink{}
	obs := NewMetricsObserver(sink, func(ctx context.Context, req protocol.InvocationRequest, _ protocol.InvocationEvent, _ error) map[string]string {
		v, _ := ctx.Value(cacheHitKey{}).(string)
		return map[string]string{"cache_hit": v, "tenant": req.TenantID + "-override"}
	})
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}

	ctx := WithCacheHit(context.Background(), "true")
	obs.OnInvocationEvent(ctx, req, protocol.InvocationEvent{Type: protocol.EventResult}, nil, 0)

	if sink.counters[0]["cache_hit"] != "true" {
		t.Fatalf("expected cache_hit label")
	}
	if sink.counters[0]["tenant"] != "t1-override" {
		t.Fatalf("enricher should override tenant")
	}
}
