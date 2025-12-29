package response

import (
	"context"
	"encoding/json"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestErrorEnvelopeIncludesTraceFromContext(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	env := Error(ctx, "CODE", "msg", nil)
	b, _ := json.Marshal(env)
	var out ErrorEnvelope
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Error.TraceID == "" {
		t.Fatalf("expected trace_id from context")
	}
}

func TestWithTraceFromContextHandlesNil(t *testing.T) {
	env := Error(nil, "CODE", "msg", nil)
	if env.Error.TraceID != "" {
		t.Fatalf("expected empty trace id when context nil")
	}
}

func TestWithTraceFromContextIgnoresInvalidSpan(t *testing.T) {
	ctx := context.Background()
	env := Error(ctx, "CODE", "msg", nil, WithTraceFromContext(ctx))
	if env.Error.TraceID != "" {
		t.Fatalf("expected empty trace id for invalid span, got %s", env.Error.TraceID)
	}
}

func TestWithTraceFromContextNilCtx(t *testing.T) {
	meta := &Meta{}
	WithTraceFromContext(nil)(meta)
	if meta.TraceID != "" {
		t.Fatalf("expected no trace id for nil context")
	}
}

func TestWithTraceFromContextNilSpan(t *testing.T) {
	ctx := trace.ContextWithSpan(context.Background(), nil)
	meta := &Meta{}
	WithTraceFromContext(ctx)(meta)
	if meta.TraceID != "" {
		t.Fatalf("expected no trace id when span is nil")
	}
}

func TestWithTraceFromContextValidSpan(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	tracer := tp.Tracer("trace-valid")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	meta := &Meta{}
	WithTraceFromContext(ctx)(meta)

	if meta.TraceID == "" {
		t.Fatalf("expected trace id to be set from valid span")
	}
}
