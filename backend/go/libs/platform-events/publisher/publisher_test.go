package publisher

import (
	"context"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/events"
	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
)

// fakeWriter wraps kafka.Writer to inspect the last message written without hitting a broker.
type fakeWriter struct {
	last kafka.Message
}

func (f *fakeWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	if len(msgs) > 0 {
		f.last = msgs[0]
	}
	return nil
}

func (f *fakeWriter) Stats() kafka.WriterStats { return kafka.WriterStats{} }
func (f *fakeWriter) Close() error             { return nil }

// stub dialer path by replacing DialLeader usage with writer.WriteMessages in test via helper.
func TestPublishSetsRequiredAndOptionalHeaders(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ServiceName = "svc-test"
	cfg.ClientID = "client-test"
	cfg.GroupID = "group-test"

	pub, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}

	// Swap writer for fake to avoid network
	fw := &fakeWriter{}
	pub.writer = fw
	pub.ready[topics.UserCreated] = true // skip ensureTopic network path

	ctx := context.Background()
	evt := events.NewEvent(topics.UserCreated, "auth-gateway", events.UserCreatedEvent{UserID: "u1"}).
		WithTenantID("tenant-1").WithUserID("user-1").WithTrace("trace-1", "span-1")

	if err := pub.Publish(ctx, topics.UserCreated, evt); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}

	headers := map[string]string{}
	for _, h := range fw.last.Headers {
		headers[h.Key] = string(h.Value)
	}

	// Required
	for _, key := range []string{"event_type", "source", "version"} {
		if headers[key] == "" {
			t.Fatalf("missing required header %s", key)
		}
	}

	// Optional
	if headers["tenant_id"] != "tenant-1" {
		t.Fatalf("tenant_id header mismatch: %s", headers["tenant_id"])
	}
	if headers["user_id"] != "user-1" {
		t.Fatalf("user_id header mismatch: %s", headers["user_id"])
	}
	if headers["trace_id"] != "trace-1" || headers["span_id"] != "span-1" {
		t.Fatalf("trace headers mismatch: trace_id=%s span_id=%s", headers["trace_id"], headers["span_id"])
	}
}

func TestPublishOmitsOptionalHeadersWhenEmpty(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ServiceName = "svc-test"
	cfg.ClientID = "client-test"
	cfg.GroupID = "group-test"

	pub, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	fw := &fakeWriter{}
	pub.writer = fw
	pub.ready[topics.UserCreated] = true

	evt := events.NewEvent(topics.UserCreated, "auth-gateway", events.UserCreatedEvent{UserID: "u1"})
	if err := pub.Publish(context.Background(), topics.UserCreated, evt); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}

	headers := map[string]string{}
	for _, h := range fw.last.Headers {
		headers[h.Key] = string(h.Value)
	}

	if headers["tenant_id"] != "" || headers["user_id"] != "" || headers["trace_id"] != "" || headers["span_id"] != "" {
		t.Fatalf("optional headers should be empty when not set: %+v", headers)
	}
}

func TestStatsReturnsWriterStats(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ServiceName = "svc-test"
	cfg.ClientID = "client-test"
	cfg.GroupID = "group-test"

	pub, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	fw := &fakeWriter{}
	pub.writer = fw
	pub.ready[topics.UserCreated] = true

	stats := pub.Stats()
	if stats.Messages != 0 {
		t.Fatalf("expected zeroed stats, got %v", stats.Messages)
	}
}
