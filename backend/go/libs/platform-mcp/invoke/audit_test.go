package invoke

import (
	"context"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-mcp/protocol"
)

type memoryAuditSink struct {
	records []AuditRecord
}

func (m *memoryAuditSink) Write(rec AuditRecord) error {
	m.records = append(m.records, rec)
	return nil
}

func TestAuditObserverRecordsOutcomeAndError(t *testing.T) {
	sink := &memoryAuditSink{}
	obs := NewAuditObserver(sink)
	req := protocol.InvocationRequest{TenantID: "t1", SessionID: "s1", RequestID: "r1", Tool: protocol.ToolRef{Name: "echo"}}

	obs.OnInvocationEvent(context.Background(), req, protocol.InvocationEvent{Type: protocol.EventError, Error: &protocol.InvocationError{Code: "invalid_request", Message: "bad"}}, nil, 120*time.Millisecond)

	if len(sink.records) != 1 {
		t.Fatalf("expected one record, got %d", len(sink.records))
	}
	rec := sink.records[0]
	if rec.Outcome != "error" || rec.ErrorCode != "invalid_request" || rec.RequestID != "r1" || rec.SessionID != "s1" {
		t.Fatalf("unexpected record: %+v", rec)
	}
}
