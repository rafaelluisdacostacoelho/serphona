package events

import (
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/topics"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/types"
)

func TestInteractionLoggedEventBind(t *testing.T) {
	payload := InteractionLoggedEvent{
		InteractionID: "int-1",
		TenantID:      "tenant-123",
		AgentID:       "agent-1",
		Channel:       "voice",
		LoggedAt:      time.Now().UTC().Truncate(time.Second),
		Metadata: map[string]interface{}{
			"conversation_id": "conv-1",
		},
	}

	evt := NewEvent(topics.InteractionLogged, "analytics-query-service", payload)
	decoded, err := types.Bind[InteractionLoggedEvent](evt)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.InteractionID != payload.InteractionID || decoded.TenantID != payload.TenantID || decoded.AgentID != payload.AgentID || decoded.Channel != payload.Channel {
		t.Fatalf("Decoded payload mismatch: %+v", decoded)
	}
	if !decoded.LoggedAt.Equal(payload.LoggedAt) {
		t.Fatalf("LoggedAt mismatch: got %s want %s", decoded.LoggedAt, payload.LoggedAt)
	}
	if decoded.Metadata["conversation_id"] != payload.Metadata["conversation_id"] {
		t.Fatalf("Metadata mismatch: %+v", decoded.Metadata)
	}
}

func TestMetricRecordedEventBind(t *testing.T) {
	payload := MetricRecordedEvent{
		Metric:     "latency_ms",
		Value:      123.4,
		Unit:       "ms",
		Dimensions: map[string]string{"path": "/healthz"},
		CapturedAt: time.Now().UTC().Truncate(time.Second),
	}

	evt := NewEvent(topics.MetricRecorded, "analytics-query-service", payload)
	decoded, err := types.Bind[MetricRecordedEvent](evt)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	if decoded.Metric != payload.Metric || decoded.Value != payload.Value || decoded.Unit != payload.Unit {
		t.Fatalf("Decoded payload mismatch: %+v", decoded)
	}
	if decoded.Dimensions["path"] != payload.Dimensions["path"] {
		t.Fatalf("Dimensions mismatch: %+v", decoded.Dimensions)
	}
	if !decoded.CapturedAt.Equal(payload.CapturedAt) {
		t.Fatalf("CapturedAt mismatch: got %s want %s", decoded.CapturedAt, payload.CapturedAt)
	}
}
