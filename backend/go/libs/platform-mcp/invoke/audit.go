package invoke

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

// AuditRecord captures a redacted audit event.
type AuditRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	TenantID     string    `json:"tenant_id"`
	Tool         string    `json:"tool"`
	Outcome      string    `json:"outcome"`
	DurationMs   int64     `json:"duration_ms"`
	RequestID    string    `json:"request_id,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	ErrorCode    string    `json:"error_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Progress     string    `json:"progress,omitempty"`
}

// AuditSink is where audit records are written.
type AuditSink interface {
	Write(record AuditRecord) error
}

// AuditSampler decides whether to emit a record.
type AuditSampler func(ctx context.Context, rec AuditRecord) bool

// AuditRouter optionally routes records to a sink.
type AuditRouter func(ctx context.Context, rec AuditRecord) AuditSink

// AuditObserver emits audit records via an AuditSink.
type AuditObserver struct {
	sink          AuditSink
	sampler       AuditSampler
	router        AuditRouter
	spanEventEmit bool
}

// NewAuditObserver builds an audit observer.
func NewAuditObserver(sink AuditSink, opts ...AuditOption) *AuditObserver {
	obs := &AuditObserver{sink: sink}
	for _, opt := range opts {
		opt(obs)
	}
	return obs
}

// AuditOption customizes AuditObserver behavior.
type AuditOption func(*AuditObserver)

// WithAuditSampler sets a sampler.
func WithAuditSampler(s AuditSampler) AuditOption {
	return func(o *AuditObserver) { o.sampler = s }
}

// WithAuditRouter sets a router to override the sink.
func WithAuditRouter(r AuditRouter) AuditOption {
	return func(o *AuditObserver) { o.router = r }
}

// WithAuditSpanEvents toggles adding audit events to the active span.
func WithAuditSpanEvents(enabled bool) AuditOption {
	return func(o *AuditObserver) { o.spanEventEmit = enabled }
}

// OnInvocationEvent converts events/errors into audit records.
func (a *AuditObserver) OnInvocationEvent(ctx context.Context, req protocol.InvocationRequest, evt protocol.InvocationEvent, invokeErr error, elapsed time.Duration) {
	if a == nil || a.sink == nil {
		return
	}
	outcome := classifyOutcome(evt, invokeErr)
	rec := AuditRecord{
		Timestamp:  time.Now().UTC(),
		TenantID:   req.TenantID,
		Tool:       req.Tool.Name,
		Outcome:    outcome,
		DurationMs: elapsed.Milliseconds(),
		RequestID:  req.RequestID,
		SessionID:  req.SessionID,
	}
	if evt.Progress != nil {
		rec.Progress = evt.Progress.Stage
	}
	if invokeErr != nil {
		rec.ErrorMessage = invokeErr.Error()
	} else if evt.Error != nil {
		rec.ErrorCode = string(evt.Error.Code)
		rec.ErrorMessage = evt.Error.Message
	}
	if a.sampler != nil && !a.sampler(ctx, rec) {
		return
	}
	sink := a.sink
	if a.router != nil {
		if routed := a.router(ctx, rec); routed != nil {
			sink = routed
		}
	}
	if sink != nil {
		_ = sink.Write(rec)
	}
	if a.spanEventEmit {
		span := trace.SpanFromContext(ctx)
		if span != nil && span.SpanContext().IsValid() {
			span.AddEvent("audit", trace.WithAttributes(
				attribute.String("tenant_id", rec.TenantID),
				attribute.String("tool", rec.Tool),
				attribute.String("outcome", rec.Outcome),
				attribute.String("request_id", rec.RequestID),
				attribute.String("session_id", rec.SessionID),
				attribute.String("error_code", rec.ErrorCode),
			))
		}
	}
}
