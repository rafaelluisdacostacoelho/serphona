package response

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	"go.opentelemetry.io/otel/trace"
)

// Meta carries correlation IDs and pagination data for responses.
type Meta struct {
	TraceID    string      `json:"trace_id,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Pagination describes pagination fields for list responses.
type Pagination struct {
	Page       int `json:"page,omitempty"`
	PageSize   int `json:"page_size,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// successEnvelope is the canonical success payload.
type successEnvelope struct {
	Data interface{} `json:"data"`
	Meta *Meta       `json:"meta,omitempty"`
}

// ErrorPayload is the canonical error body.
type ErrorPayload struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// ErrorEnvelope is the canonical error payload wrapper.
type ErrorEnvelope struct {
	Error ErrorPayload `json:"error"`
}

// Option allows customizing response metadata.
type Option func(*metaOptions)

type metaOptions struct {
	pagination *Pagination
}

// WithPagination attaches pagination metadata to the response.
func WithPagination(p Pagination) Option {
	return func(opts *metaOptions) {
		opts.pagination = &p
	}
}

// WriteSuccess writes a success envelope with optional metadata.
func WriteSuccess(ctx context.Context, w http.ResponseWriter, status int, data interface{}, opts ...Option) {
	meta := buildMeta(ctx, opts...)
	payload := successEnvelope{Data: data, Meta: meta}
	writeJSON(w, status, payload)
}

// WriteError writes an error envelope and preserves correlation IDs.
func WriteError(ctx context.Context, w http.ResponseWriter, status int, code, message string, details interface{}) {
	meta := buildMeta(ctx)
	payload := ErrorEnvelope{Error: ErrorPayload{Code: code, Message: message, Details: details}}
	if meta != nil {
		payload.Error.TraceID = meta.TraceID
		payload.Error.RequestID = meta.RequestID
	}
	writeJSON(w, status, payload)
}

func buildMeta(ctx context.Context, opts ...Option) *Meta {
	m := &Meta{}

	if ctx != nil {
		if reqID, err := middleware.RequestIDFromContext(ctx); err == nil && reqID != "" {
			m.RequestID = reqID
		}
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			m.TraceID = sc.TraceID().String()
		}
	}

	cfg := metaOptions{}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.pagination != nil {
		m.Pagination = cfg.pagination
	}

	if m.TraceID == "" && m.RequestID == "" && m.Pagination == nil {
		return nil
	}
	return m
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
