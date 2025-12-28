package invoke

import (
	"context"
	"testing"

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
