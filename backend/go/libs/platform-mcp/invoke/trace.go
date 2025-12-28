package invoke

import (
	"context"
	"sync"
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
	spans  sync.Map // key -> trace.Span
}

// NewTracingObserver builds an OTEL-based observer.
func NewTracingObserver(name string) *TracingObserver {
	if name == "" {
		name = "platform-mcp"
	}
	return &TracingObserver{tracer: otel.Tracer(name)}
}

func (t *TracingObserver) OnInvocationEvent(ctx context.Context, req protocol.InvocationRequest, evt protocol.InvocationEvent, invokeErr error, _ time.Duration) {
	key := spanKey(req)
	span := t.loadOrStart(ctx, key, req)

	if invokeErr != nil {
		span.RecordError(invokeErr)
		span.SetStatus(codes.Error, invokeErr.Error())
		span.End()
		t.spans.Delete(key)
		return
	}

	if evt.Progress != nil {
		span.AddEvent("progress", trace.WithAttributes(attribute.String("stage", evt.Progress.Stage)))
	}

	if evt.Error != nil {
		span.RecordError(mcperrors.New(evt.Error.Code, evt.Error.Message))
		span.SetStatus(codes.Error, evt.Error.Message)
		span.End()
		t.spans.Delete(key)
		return
	}

	if evt.Type == protocol.EventResult {
		span.SetStatus(codes.Ok, "")
		span.End()
		t.spans.Delete(key)
	}
}

func (t *TracingObserver) loadOrStart(ctx context.Context, key string, req protocol.InvocationRequest) trace.Span {
	if val, ok := t.spans.Load(key); ok {
		if sp, ok := val.(trace.Span); ok {
			return sp
		}
	}
	ctx, span := t.tracer.Start(ctx, "mcp.invocation", trace.WithSpanKind(trace.SpanKindInternal))
	attrs := []attribute.KeyValue{
		attribute.String("tenant_id", req.TenantID),
		attribute.String("tool", req.Tool.Name),
		attribute.String("request_id", req.RequestID),
		attribute.String("session_id", req.SessionID),
	}
	span.SetAttributes(attrs...)
	t.spans.Store(key, span)
	return span
}

func spanKey(req protocol.InvocationRequest) string {
	if req.RequestID != "" {
		return req.RequestID
	}
	return req.TenantID + "|" + req.SessionID + "|" + req.Tool.Name
}
