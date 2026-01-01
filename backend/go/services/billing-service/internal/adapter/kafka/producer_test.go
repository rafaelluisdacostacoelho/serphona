package kafka

import (
	"context"
	"fmt"
	"testing"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

func TestSendMessageSetsTenantHeader(t *testing.T) {
	cfg := sarama.NewConfig()
	mock := mocks.NewSyncProducer(t, cfg)
	producer := &Producer{producer: mock}

	mock.ExpectSendMessageWithMessageCheckerFunctionAndSucceed(func(msg *sarama.ProducerMessage) error {
		for _, h := range msg.Headers {
			if string(h.Key) == authmw.TenantIDHeader {
				if string(h.Value) != "tenant-123" {
					return fmt.Errorf("unexpected tenant header value: %s", string(h.Value))
				}
				return nil
			}
		}
		return fmt.Errorf("tenant header missing")
	})

	ctx := authmw.WithTenantID(context.Background(), "tenant-123")
	if err := producer.SendMessage(ctx, "topic", []byte("k"), []byte("v")); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestSendMessageWithoutTenantHeader(t *testing.T) {
	cfg := sarama.NewConfig()
	mock := mocks.NewSyncProducer(t, cfg)
	producer := &Producer{producer: mock}

	mock.ExpectSendMessageWithMessageCheckerFunctionAndSucceed(func(msg *sarama.ProducerMessage) error {
		for _, h := range msg.Headers {
			if string(h.Key) == authmw.TenantIDHeader {
				return fmt.Errorf("expected no tenant header, found %s", string(h.Value))
			}
		}
		return nil
	})

	if err := producer.SendMessage(context.Background(), "topic", []byte("k"), []byte("v")); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}
