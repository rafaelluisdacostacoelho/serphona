package response

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type successBody struct {
	Data map[string]string `json:"data"`
	Meta Meta              `json:"meta"`
}

type errorBody struct {
	Error ErrorPayload `json:"error"`
}

func TestWriteSuccessIncludesMeta(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test")

	ctx := middleware.WithRequestID(context.Background(), "req-123")
	ctx, span := tracer.Start(ctx, "span")
	traceID := span.SpanContext().TraceID().String()
	span.End()

	w := httptest.NewRecorder()

	WriteSuccess(ctx, w, http.StatusCreated, map[string]string{"ok": "yes"}, WithPagination(Pagination{Page: 1, PageSize: 20, Total: 40, TotalPages: 2}))

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	var body successBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body.Data["ok"] != "yes" {
		t.Fatalf("unexpected data: %#v", body.Data)
	}
	if body.Meta.RequestID != "req-123" {
		t.Fatalf("expected request_id req-123, got %q", body.Meta.RequestID)
	}
	if body.Meta.TraceID != traceID {
		t.Fatalf("expected trace_id %s, got %q", traceID, body.Meta.TraceID)
	}
	if body.Meta.Pagination == nil || body.Meta.Pagination.Page != 1 || body.Meta.Pagination.Total != 40 {
		t.Fatalf("unexpected pagination: %#v", body.Meta.Pagination)
	}
}

func TestWriteErrorIncludesIDs(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test")

	ctx := middleware.WithRequestID(context.Background(), "req-999")
	ctx, span := tracer.Start(ctx, "span")
	traceID := span.SpanContext().TraceID().String()
	span.End()

	w := httptest.NewRecorder()

	WriteError(ctx, w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid input", map[string]string{"field": "reason"})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var body errorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}

	if body.Error.Code != "VALIDATION_ERROR" || body.Error.Message != "invalid input" {
		t.Fatalf("unexpected error payload: %#v", body.Error)
	}
	if body.Error.RequestID != "req-999" {
		t.Fatalf("expected request_id req-999, got %q", body.Error.RequestID)
	}
	if body.Error.TraceID != traceID {
		t.Fatalf("expected trace_id %s, got %q", traceID, body.Error.TraceID)
	}
}

func TestWriteJSONNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusNoContent, map[string]string{"ignored": "value"})

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("expected empty body for no content, got %q", w.Body.String())
	}
}

func TestWriteJSONEncodingError(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]float64{"bad": math.Inf(1)})

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on encode failure, got %d", w.Code)
	}
}
