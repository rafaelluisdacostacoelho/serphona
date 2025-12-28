package response

import (
	"context"
	"encoding/json"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSuccessEnvelopeWithMeta(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	env := Success(ctx, map[string]string{"foo": "bar"}, WithRequestID("r1"))

	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	data, ok := out["data"].(map[string]interface{})
	if !ok || data["foo"] != "bar" {
		t.Fatalf("unexpected data: %#v", out["data"])
	}
	meta, ok := out["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("meta missing: %#v", out)
	}
	if meta["request_id"] != "r1" {
		t.Fatalf("expected request_id r1, got %v", meta["request_id"])
	}
	if meta["trace_id"] == "" {
		t.Fatalf("expected trace_id from span")
	}
}

func TestSuccessEnvelopeOmitsEmptyMeta(t *testing.T) {
	env := Success(context.Background(), "ok")
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if string(b) != `{"data":"ok"}` {
		t.Fatalf("unexpected json: %s", string(b))
	}
}

func TestErrorEnvelopeShape(t *testing.T) {
	env := Error(context.Background(), "VALIDATION_ERROR", "bad", map[string]string{"f": "x"}, WithRequestID("req-123"))
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	errBody, ok := out["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("error body missing: %#v", out)
	}
	if errBody["code"] != "VALIDATION_ERROR" || errBody["message"] != "bad" {
		t.Fatalf("unexpected error body: %#v", errBody)
	}
	if errBody["details"].(map[string]interface{})["f"] != "x" {
		t.Fatalf("unexpected details: %#v", errBody)
	}
	if errBody["request_id"] != "req-123" {
		t.Fatalf("expected request_id, got %#v", errBody["request_id"])
	}
}

func TestPaginationIncluded(t *testing.T) {
	env := Success(context.Background(), []int{1, 2}, WithPagination(Pagination{Page: 1, PageSize: 2, Total: 10, TotalPages: 5}))
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	meta := out["meta"].(map[string]interface{})
	if meta["pagination"] == nil {
		t.Fatalf("pagination missing in meta")
	}
}
