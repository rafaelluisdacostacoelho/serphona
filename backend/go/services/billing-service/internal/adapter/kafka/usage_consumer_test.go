package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/google/uuid"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
	walletapp "github.com/rafaelluisdacostacoelho/serphona/backend/go/services/billing-service/internal/application/wallet"
)

func TestUsageHandlerProcessesEvent(t *testing.T) {
	t.Parallel()

	walletSvc := &mockWalletService{}
	handler := &usageHandler{wallet: walletSvc}

	evt := walletapp.UsageReported{
		TenantID:    uuid.New(),
		Period:      "2025-12",
		OccurredAt:  time.Now().UTC().Format(time.RFC3339),
		Source:      "tenant-manager",
		Calls:       1,
		Minutes:     2,
		APIRequests: 3,
		RequestID:   "req-1",
	}

	payload, _ := json.Marshal(evt)
	msg := &sarama.ConsumerMessage{Value: payload, Offset: 10}

	status, err := handler.processMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("processMessage error: %v", err)
	}
	if status != "success" {
		t.Fatalf("unexpected status: %s", status)
	}

	if len(walletSvc.calls) != 1 {
		t.Fatalf("expected 1 debit call, got %d", len(walletSvc.calls))
	}
	if walletSvc.calls[0].RequestID != evt.RequestID {
		t.Fatalf("request_id mismatch")
	}
}

type mockWalletService struct {
	calls []walletapp.UsageReported
}

func (m *mockWalletService) DebitUsage(_ context.Context, evt walletapp.UsageReported) error {
	m.calls = append(m.calls, evt)
	return nil
}

func TestSendToDLQAddsTenantHeader(t *testing.T) {
	t.Parallel()

	cfg := sarama.NewConfig()
	producer := mocks.NewSyncProducer(t, cfg)
	tenantID := uuid.NewString()

	producer.ExpectSendMessageWithMessageCheckerFunctionAndSucceed(func(msg *sarama.ProducerMessage) error {
		found := false
		for _, h := range msg.Headers {
			if string(h.Key) == authmw.TenantIDHeader {
				found = true
				if string(h.Value) != tenantID {
					return errors.New("unexpected tenant header value")
				}
			}
		}
		if !found {
			return errors.New("tenant header missing")
		}
		return nil
	})

	h := &usageHandler{producer: producer, dlqTopic: "dlq"}
	payload := map[string]string{"tenant_id": tenantID, "request_id": "r-1", "trace_id": "tr-1"}
	raw, _ := json.Marshal(payload)
	msg := &sarama.ConsumerMessage{Topic: "usage.reported", Partition: 0, Offset: 1, Value: raw}

	if err := h.sendToDLQ(msg, errors.New("fail"), "error"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}
