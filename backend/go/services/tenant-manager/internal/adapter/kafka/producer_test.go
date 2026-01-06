package kafka

import (
	"context"
	"fmt"
	"testing"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	authmw "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-auth/middleware"
)

func TestSendMessageTenantHeaderInjection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		ctx          context.Context
		expectHeader bool
		tenantID     string
	}{
		{
			name:         "with tenant in context",
			ctx:          authmw.WithTenantID(context.Background(), "tenant-123"),
			expectHeader: true,
			tenantID:     "tenant-123",
		},
		{
			name:         "without tenant in context",
			ctx:          context.Background(),
			expectHeader: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := sarama.NewConfig()
			mock := mocks.NewSyncProducer(t, cfg)
			producer := &Producer{producer: mock}

			mock.ExpectSendMessageWithMessageCheckerFunctionAndSucceed(func(msg *sarama.ProducerMessage) error {
				for _, h := range msg.Headers {
					if string(h.Key) == authmw.TenantIDHeader {
						if !tt.expectHeader {
							return fmt.Errorf("unexpected tenant header present: %s", string(h.Value))
						}
						if string(h.Value) != tt.tenantID {
							return fmt.Errorf("unexpected tenant header value: %s", string(h.Value))
						}
						return nil
					}
				}

				if tt.expectHeader {
					return fmt.Errorf("tenant header missing")
				}
				return nil
			})

			if err := producer.SendMessage(tt.ctx, "topic", []byte("k"), []byte("v")); err != nil {
				t.Fatalf("expected success, got %v", err)
			}
		})
	}
}

func TestEnsureTenantHeaders(t *testing.T) {
	t.Parallel()

	ctxWithTenant := authmw.WithTenantID(context.Background(), "tenant-123")

	tests := []struct {
		name           string
		ctx            context.Context
		headers        []sarama.RecordHeader
		expectHeaders  []sarama.RecordHeader
		expectAppended bool
	}{
		{
			name:    "adds header when missing",
			ctx:     ctxWithTenant,
			headers: nil,
			expectHeaders: []sarama.RecordHeader{{
				Key:   []byte(authmw.TenantIDHeader),
				Value: []byte("tenant-123"),
			}},
			expectAppended: true,
		},
		{
			name: "preserves existing tenant header",
			ctx:  ctxWithTenant,
			headers: []sarama.RecordHeader{{
				Key:   []byte(authmw.TenantIDHeader),
				Value: []byte("existing"),
			}},
			expectHeaders: []sarama.RecordHeader{{
				Key:   []byte(authmw.TenantIDHeader),
				Value: []byte("existing"),
			}},
		},
		{
			name: "fills empty tenant header",
			ctx:  ctxWithTenant,
			headers: []sarama.RecordHeader{{
				Key: []byte(authmw.TenantIDHeader),
			}},
			expectHeaders: []sarama.RecordHeader{{
				Key:   []byte(authmw.TenantIDHeader),
				Value: []byte("tenant-123"),
			}},
		},
		{
			name: "no tenant in context",
			ctx:  context.Background(),
			headers: []sarama.RecordHeader{{
				Key:   []byte("other"),
				Value: []byte("v"),
			}},
			expectHeaders: []sarama.RecordHeader{{
				Key:   []byte("other"),
				Value: []byte("v"),
			}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orig := cloneHeaders(tt.headers)
			got := ensureTenantHeaders(tt.ctx, tt.headers)

			if len(got) != len(tt.expectHeaders) {
				t.Fatalf("header count mismatch: got %d want %d", len(got), len(tt.expectHeaders))
			}

			for i := range got {
				if string(got[i].Key) != string(tt.expectHeaders[i].Key) || string(got[i].Value) != string(tt.expectHeaders[i].Value) {
					t.Fatalf("header %d mismatch: got %v want %v", i, got[i], tt.expectHeaders[i])
				}
			}

			if len(tt.headers) > 0 {
				for i := range tt.headers {
					if string(tt.headers[i].Key) != string(orig[i].Key) || string(tt.headers[i].Value) != string(orig[i].Value) {
						t.Fatalf("input headers mutated: got %v want %v", tt.headers[i], orig[i])
					}
				}
			}
		})
	}
}

func TestEnsureTenantHeaders_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		ctx            context.Context
		headers        []sarama.RecordHeader
		expectHeaders  []sarama.RecordHeader
		expectAppended bool
	}{
		{
			name: "with tenant in context",
			ctx:  authmw.WithTenantID(context.Background(), "tenant-123"),
			headers: []sarama.RecordHeader{
				{Key: []byte("existing-header"), Value: []byte("value")},
			},
			expectHeaders: []sarama.RecordHeader{
				{Key: []byte("existing-header"), Value: []byte("value")},
				{Key: []byte(authmw.TenantIDHeader), Value: []byte("tenant-123")},
			},
			expectAppended: true,
		},
		{
			name: "without tenant in context",
			ctx:  context.Background(),
			headers: []sarama.RecordHeader{
				{Key: []byte("existing-header"), Value: []byte("value")},
			},
			expectHeaders: []sarama.RecordHeader{
				{Key: []byte("existing-header"), Value: []byte("value")},
			},
			expectAppended: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ensureTenantHeaders(tt.ctx, tt.headers)

			if len(got) != len(tt.expectHeaders) {
				t.Fatalf("expected %d headers, got %d", len(tt.expectHeaders), len(got))
			}

			for i, header := range got {
				if string(header.Key) != string(tt.expectHeaders[i].Key) || string(header.Value) != string(tt.expectHeaders[i].Value) {
					t.Errorf("header mismatch at index %d: expected %+v, got %+v", i, tt.expectHeaders[i], header)
				}
			}

			if tt.expectAppended {
				found := false
				for _, header := range got {
					if string(header.Key) == authmw.TenantIDHeader {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected tenant header to be appended, but it was not")
				}
			}
		})
	}
}
