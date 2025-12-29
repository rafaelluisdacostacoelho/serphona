package response

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// Pagination represents pagination metadata for list responses.
type Pagination struct {
	Page       int `json:"page,omitempty"`
	PageSize   int `json:"page_size,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// Meta carries optional correlation and pagination data.
type Meta struct {
	TraceID    string      `json:"trace_id,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// SuccessEnvelope is the standard success shape.
type SuccessEnvelope struct {
	Data interface{} `json:"data"`
	Meta *Meta       `json:"meta,omitempty"`
}

// ErrorBody holds error details.
type ErrorBody struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// ErrorEnvelope is the standard error shape.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// Success builds a success envelope with optional meta options.
func Success(ctx context.Context, data interface{}, opts ...Option) SuccessEnvelope {
	meta := buildMeta(ctx, opts...)
	if meta.TraceID == "" && meta.RequestID == "" && meta.Pagination == nil {
		return SuccessEnvelope{Data: data}
	}
	return SuccessEnvelope{Data: data, Meta: meta}
}

// Error builds an error envelope with optional correlation options.
func Error(ctx context.Context, code, message string, details interface{}, opts ...Option) ErrorEnvelope {
	meta := buildMeta(ctx, opts...)
	body := ErrorBody{Code: code, Message: message, Details: details, TraceID: meta.TraceID, RequestID: meta.RequestID}
	return ErrorEnvelope{Error: body}
}

// Option customizes envelope metadata.
type Option func(*Meta)

// WithRequestID sets the request_id meta field.
func WithRequestID(id string) Option {
	return func(m *Meta) { m.RequestID = id }
}

// WithTraceID sets the trace_id meta field.
func WithTraceID(id string) Option {
	return func(m *Meta) { m.TraceID = id }
}

// WithTraceFromContext extracts a trace_id from the active span if present.
func WithTraceFromContext(ctx context.Context) Option {
	return func(m *Meta) {
		if ctx == nil {
			return
		}
		span := trace.SpanFromContext(ctx)
		sc := span.SpanContext()
		if sc.IsValid() {
			m.TraceID = sc.TraceID().String()
		}
	}
}

// WithPagination sets pagination metadata.
func WithPagination(p Pagination) Option {
	return func(m *Meta) {
		// Copy to avoid caller mutation.
		pg := p
		m.Pagination = &pg
	}
}

func buildMeta(ctx context.Context, opts ...Option) *Meta {
	meta := &Meta{}
	// Always try to extract trace_id when ctx provided.
	if ctx != nil {
		WithTraceFromContext(ctx)(meta)
	}
	for _, opt := range opts {
		if opt != nil {
			opt(meta)
		}
	}
	if meta.TraceID == "" && meta.RequestID == "" && meta.Pagination == nil {
		return meta
	}
	return meta
}
