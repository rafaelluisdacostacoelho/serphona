package events

import (
	"reflect"
	"testing"
	"time"

	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

func TestToolInvokedEventBind(t *testing.T) {
	payload := ToolInvokedEvent{
		ToolID:        "tool-123",
		TenantID:      "tenant-xyz",
		Action:        "sync_contacts",
		InvokedAt:     time.Now().UTC().Truncate(time.Second),
		CorrelationID: "corr-1",
		Payload: map[string]interface{}{
			"job":   "contacts-sync",
			"batch": float64(3),
		},
	}

	evt := NewEvent(topics.ToolInvoked, "tools-gateway", payload)
	decoded, err := types.Bind[ToolInvokedEvent](evt)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.ToolID != payload.ToolID {
		t.Fatalf("ToolID mismatch: got %s want %s", decoded.ToolID, payload.ToolID)
	}
	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if decoded.Action != payload.Action {
		t.Fatalf("Action mismatch: got %s want %s", decoded.Action, payload.Action)
	}
	if !decoded.InvokedAt.Equal(payload.InvokedAt) {
		t.Fatalf("InvokedAt mismatch: got %s want %s", decoded.InvokedAt, payload.InvokedAt)
	}
	if decoded.CorrelationID != payload.CorrelationID {
		t.Fatalf("CorrelationID mismatch: got %s want %s", decoded.CorrelationID, payload.CorrelationID)
	}
	if !reflect.DeepEqual(decoded.Payload, payload.Payload) {
		t.Fatalf("Payload mismatch: got %+v want %+v", decoded.Payload, payload.Payload)
	}
}

func TestToolCompletedEventBind(t *testing.T) {
	payload := ToolCompletedEvent{
		ToolID:        "tool-123",
		TenantID:      "tenant-xyz",
		Action:        "sync_contacts",
		Result:        map[string]interface{}{"synced": float64(42)},
		DurationMs:    1500,
		CompletedAt:   time.Now().UTC().Truncate(time.Second),
		CorrelationID: "corr-1",
	}

	evt := NewEvent(topics.ToolCompleted, "tools-gateway", payload)
	decoded, err := types.Bind[ToolCompletedEvent](evt)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.ToolID != payload.ToolID {
		t.Fatalf("ToolID mismatch: got %s want %s", decoded.ToolID, payload.ToolID)
	}
	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if decoded.Action != payload.Action {
		t.Fatalf("Action mismatch: got %s want %s", decoded.Action, payload.Action)
	}
	if decoded.DurationMs != payload.DurationMs {
		t.Fatalf("DurationMs mismatch: got %d want %d", decoded.DurationMs, payload.DurationMs)
	}
	if !decoded.CompletedAt.Equal(payload.CompletedAt) {
		t.Fatalf("CompletedAt mismatch: got %s want %s", decoded.CompletedAt, payload.CompletedAt)
	}
	if decoded.CorrelationID != payload.CorrelationID {
		t.Fatalf("CorrelationID mismatch: got %s want %s", decoded.CorrelationID, payload.CorrelationID)
	}
	if !reflect.DeepEqual(decoded.Result, payload.Result) {
		t.Fatalf("Result mismatch: got %+v want %+v", decoded.Result, payload.Result)
	}
}

func TestToolFailedEventBind(t *testing.T) {
	payload := ToolFailedEvent{
		ToolID:        "tool-123",
		TenantID:      "tenant-xyz",
		Action:        "sync_contacts",
		Error:         "timeout",
		DurationMs:    1500,
		FailedAt:      time.Now().UTC().Truncate(time.Second),
		Context:       "job=contacts-sync",
		CorrelationID: "corr-1",
	}

	evt := NewEvent(topics.ToolFailed, "tools-gateway", payload)
	decoded, err := types.Bind[ToolFailedEvent](evt)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.ToolID != payload.ToolID {
		t.Fatalf("ToolID mismatch: got %s want %s", decoded.ToolID, payload.ToolID)
	}
	if decoded.TenantID != payload.TenantID {
		t.Fatalf("TenantID mismatch: got %s want %s", decoded.TenantID, payload.TenantID)
	}
	if decoded.Action != payload.Action {
		t.Fatalf("Action mismatch: got %s want %s", decoded.Action, payload.Action)
	}
	if decoded.Error != payload.Error {
		t.Fatalf("Error mismatch: got %s want %s", decoded.Error, payload.Error)
	}
	if decoded.DurationMs != payload.DurationMs {
		t.Fatalf("DurationMs mismatch: got %d want %d", decoded.DurationMs, payload.DurationMs)
	}
	if !decoded.FailedAt.Equal(payload.FailedAt) {
		t.Fatalf("FailedAt mismatch: got %s want %s", decoded.FailedAt, payload.FailedAt)
	}
	if decoded.Context != payload.Context {
		t.Fatalf("Context mismatch: got %s want %s", decoded.Context, payload.Context)
	}
	if decoded.CorrelationID != payload.CorrelationID {
		t.Fatalf("CorrelationID mismatch: got %s want %s", decoded.CorrelationID, payload.CorrelationID)
	}
}
