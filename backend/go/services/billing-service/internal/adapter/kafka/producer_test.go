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

func TestEnsureTenantHeadersPreservesExisting(t *testing.T) {
	ctx := authmw.WithTenantID(context.Background(), "tenant-123")
	headers := []sarama.RecordHeader{{Key: []byte(authmw.TenantIDHeader), Value: []byte("existing")}}

	got := ensureTenantHeaders(ctx, headers)

	if len(got) != 1 {
		t.Fatalf("expected 1 header, got %d", len(got))
	}
	if string(got[0].Value) != "existing" {
		t.Fatalf("expected to preserve existing header value, got %s", string(got[0].Value))
	}

	// Ensure input was not mutated
	if string(headers[0].Value) != "existing" {
		t.Fatalf("input headers mutated, got %s", string(headers[0].Value))
	}
}

func TestEnsureTenantHeadersFillsEmptyValue(t *testing.T) {
	ctx := authmw.WithTenantID(context.Background(), "tenant-123")
	headers := []sarama.RecordHeader{{Key: []byte(authmw.TenantIDHeader), Value: []byte("")}}

	got := ensureTenantHeaders(ctx, headers)

	if string(got[0].Value) != "tenant-123" {
		t.Fatalf("expected header value to be filled, got %s", string(got[0].Value))
	}

	if len(headers[0].Value) != 0 {
		t.Fatalf("input header mutated, got %s", string(headers[0].Value))
	}
}
