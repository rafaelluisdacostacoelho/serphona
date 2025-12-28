package invoke

import (
	"context"
	"time"

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

// AuditObserver emits audit records via an AuditSink.
type AuditObserver struct {
	sink AuditSink
}

// NewAuditObserver builds an audit observer.
func NewAuditObserver(sink AuditSink) *AuditObserver {
	return &AuditObserver{sink: sink}
}

// OnInvocationEvent converts events/errors into audit records.
func (a *AuditObserver) OnInvocationEvent(_ context.Context, req protocol.InvocationRequest, evt protocol.InvocationEvent, invokeErr error, elapsed time.Duration) {
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
	_ = a.sink.Write(rec)
}
