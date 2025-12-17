package events

import (
	"testing"
	"time"

	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

func TestSystemErrorEventBind(t *testing.T) {
	payload := SystemErrorEvent{
		Service:    "analytics-query-service",
		Error:      "timeout contacting ClickHouse",
		Severity:   "error",
		OccurredAt: time.Now().UTC().Truncate(time.Second),
		TraceID:    "trace-123",
		SpanID:     "span-456",
		Labels: map[string]string{
			"tenant_id": "tenant-xyz",
		},
	}

	evt := NewEvent(topics.SystemError, "analytics-query-service", payload)
	decoded, err := types.Bind[SystemErrorEvent](evt)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.Service != payload.Service {
		t.Fatalf("Service mismatch: got %s want %s", decoded.Service, payload.Service)
	}
	if decoded.Error != payload.Error {
		t.Fatalf("Error mismatch: got %s want %s", decoded.Error, payload.Error)
	}
	if decoded.Severity != payload.Severity {
		t.Fatalf("Severity mismatch: got %s want %s", decoded.Severity, payload.Severity)
	}
	if !decoded.OccurredAt.Equal(payload.OccurredAt) {
		t.Fatalf("OccurredAt mismatch: got %s want %s", decoded.OccurredAt, payload.OccurredAt)
	}
	if decoded.TraceID != payload.TraceID {
		t.Fatalf("TraceID mismatch: got %s want %s", decoded.TraceID, payload.TraceID)
	}
	if decoded.SpanID != payload.SpanID {
		t.Fatalf("SpanID mismatch: got %s want %s", decoded.SpanID, payload.SpanID)
	}
	if len(decoded.Labels) != len(payload.Labels) || decoded.Labels["tenant_id"] != payload.Labels["tenant_id"] {
		t.Fatalf("Labels mismatch: got %+v want %+v", decoded.Labels, payload.Labels)
	}
}
