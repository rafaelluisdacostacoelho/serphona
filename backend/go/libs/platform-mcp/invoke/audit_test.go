package invoke

import (
	"context"
	"errors"
	"testing"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type memoryAuditSink struct {
	records []AuditRecord
}

func (m *memoryAuditSink) Write(rec AuditRecord) error {
	m.records = append(m.records, rec)
	return nil
}

func TestAuditObserverRecordsOutcomeAndError(t *testing.T) {
	sink := &memoryAuditSink{}
	obs := NewAuditObserver(sink)
	req := protocol.InvocationRequest{TenantID: "t1", SessionID: "s1", RequestID: "r1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventError, Error: &protocol.InvocationError{Code: "invalid_request", Message: "bad"}}, nil, 120*time.Millisecond)

	if len(sink.records) != 1 {
		t.Fatalf("expected one record, got %d", len(sink.records))
	}
	rec := sink.records[0]
	if rec.Outcome != "error" || rec.ErrorCode != "invalid_request" || rec.RequestID != "r1" || rec.SessionID != "s1" {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestAuditObserverSamplerSkips(t *testing.T) {
	sink := &memoryAuditSink{}
	obs := NewAuditObserver(sink, WithAuditSampler(func(_ context.Context, _ AuditRecord) bool { return false }))
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventResult}, nil, 0)

	if len(sink.records) != 0 {
		t.Fatalf("expected sampler to drop record")
	}
}

func TestAuditObserverRouterOverridesSink(t *testing.T) {
	defaultSink := &memoryAuditSink{}
	altSink := &memoryAuditSink{}
	obs := NewAuditObserver(defaultSink, WithAuditRouter(func(_ context.Context, _ AuditRecord) AuditSink { return altSink }))
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventResult}, nil, 0)

	if len(defaultSink.records) != 0 {
		t.Fatalf("expected default sink unused")
	}
	if len(altSink.records) != 1 {
		t.Fatalf("expected routed sink to receive record")
	}
}

func TestAuditObserverAddsSpanEvent(t *testing.T) {
	ctx := context.Background()
	exp := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	tracer := provider.Tracer("test")
	ctx, span := tracer.Start(ctx, "invocation")

	sink := &memoryAuditSink{}
	obs := NewAuditObserver(sink, WithAuditSpanEvents(true))
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(ctx, req, protocol.InvocationEvent{Type: protocol.EventResult}, nil, 0)
	span.End()
	_ = provider.ForceFlush(context.Background())

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected one span, got %d", len(spans))
	}
	events := spans[0].Events
	if len(events) == 0 {
		t.Fatalf("expected an audit event recorded")
	}
	found := false
	for _, e := range events {
		if e.Name == "audit" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("audit event not found on span")
	}
}

func TestAuditObserverHandlesInvokeError(t *testing.T) {
	sink := &memoryAuditSink{}
	obs := NewAuditObserver(sink)
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventResult}, errors.New("invoke boom"), 0)

	if len(sink.records) != 1 || sink.records[0].Outcome != "error" || sink.records[0].ErrorMessage == "" {
		t.Fatalf("expected invoke error recorded, got %+v", sink.records)
	}
}

func TestAuditObserverNoSinkIsNoop(t *testing.T) {
	obs := NewAuditObserver(nil)
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}
	obs.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventResult}, nil, 0)
}

func TestAuditObserverRecordsProgress(t *testing.T) {
	sink := &memoryAuditSink{}
	obs := NewAuditObserver(sink)
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventProgress, Progress: &protocol.Progress{Stage: "step"}}, nil, 0)

	if len(sink.records) != 1 || sink.records[0].Progress != "step" {
		t.Fatalf("expected progress recorded, got %+v", sink.records)
	}
}
