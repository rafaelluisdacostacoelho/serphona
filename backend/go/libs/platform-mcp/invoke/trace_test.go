package invoke

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

func TestTracingObserverRecordsStatus(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	obs := &TracingObserver{tracer: provider.Tracer("test")}

	ctx := context.Background()
	req := protocol.InvocationRequest{TenantID: "t1", RequestID: "r1", SessionID: "s1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(ctx, req, protocol.InvocationEvent{Type: protocol.EventResult}, nil, 0)

	provider.ForceFlush(ctx)
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected one span, got %d", len(spans))
	}
	if spans[0].Status.Code != codes.Ok {
		t.Fatalf("expected OK status, got %v", spans[0].Status.Code)
	}
}

func TestTracingObserverKeepsSpanAcrossProgress(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	obs := &TracingObserver{tracer: provider.Tracer("test")}

	ctx := context.Background()
	req := protocol.InvocationRequest{TenantID: "t1", RequestID: "r1", SessionID: "s1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(ctx, req, protocol.InvocationEvent{Type: protocol.EventProgress, Progress: &protocol.Progress{Stage: "chunk1"}}, nil, 0)
	obs.OnInvocationEvent(ctx, req, protocol.InvocationEvent{Type: protocol.EventProgress, Progress: &protocol.Progress{Stage: "chunk2"}}, nil, 0)
	obs.OnInvocationEvent(ctx, req, protocol.InvocationEvent{Type: protocol.EventResult}, nil, 0)

	_ = provider.ForceFlush(context.Background())
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected one span, got %d", len(spans))
	}
	if spans[0].Status.Code != codes.Ok {
		t.Fatalf("expected OK status, got %v", spans[0].Status.Code)
	}
	if len(spans[0].Events) != 2 {
		t.Fatalf("expected two progress events, got %d", len(spans[0].Events))
	}
	if spans[0].EndTime.Before(spans[0].StartTime) {
		t.Fatalf("span should be ended after events")
	}
}

func TestTracingObserverClosesOnError(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	obs := &TracingObserver{tracer: provider.Tracer("test")}

	ctx := context.Background()
	req := protocol.InvocationRequest{TenantID: "t1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(ctx, req, protocol.InvocationEvent{Type: protocol.EventProgress, Progress: &protocol.Progress{Stage: "chunk1"}}, nil, 0)
	obs.OnInvocationEvent(ctx, req, protocol.InvocationEvent{Type: protocol.EventError, Error: &protocol.InvocationError{Code: "boom", Message: "fail"}}, nil, time.Second)

	_ = provider.ForceFlush(context.Background())
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected one span, got %d", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Fatalf("expected error status, got %v", spans[0].Status.Code)
	}
}
