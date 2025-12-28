package invoke

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	mcperrors "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/errors"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// TracingObserver emits spans around invocation events.
type TracingObserver struct {
	tracer trace.Tracer
}

// NewTracingObserver builds an OTEL-based observer.
func NewTracingObserver(name string) *TracingObserver {
	if name == "" {
		name = "platform-mcp"
	}
	return &TracingObserver{tracer: otel.Tracer(name)}
}

func (t *TracingObserver) OnInvocationEvent(ctx context.Context, req protocol.InvocationRequest, evt protocol.InvocationEvent, invokeErr error, _ time.Duration) {
	ctx, span := t.tracer.Start(ctx, "mcp.invocation", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	attrs := []attribute.KeyValue{
		attribute.String("tenant_id", req.TenantID),
		attribute.String("tool", req.Tool.Name),
		attribute.String("request_id", req.RequestID),
		attribute.String("session_id", req.SessionID),
	}
	span.SetAttributes(attrs...)

	if invokeErr != nil {
		span.RecordError(invokeErr)
		span.SetStatus(codes.Error, invokeErr.Error())
		return
	}
	if evt.Error != nil {
		span.RecordError(mcperrors.New(evt.Error.Code, evt.Error.Message))
		span.SetStatus(codes.Error, evt.Error.Message)
	} else if evt.Type == protocol.EventResult {
		span.SetStatus(codes.Ok, "")
	}
}
