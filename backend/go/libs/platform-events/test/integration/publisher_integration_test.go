//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/events"
	"github.com/serphona/serphona/backend/go/libs/platform-events/publisher"
	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

// TestIntegrationPublisher publishes a small set of events against a Kafka broker.
// Requires KAFKA_BROKERS to be set and reachable; otherwise the test is skipped.
func TestIntegrationPublisher(t *testing.T) {
	cfg := config.LoadFromEnv()
	if len(cfg.Brokers) == 0 {
		t.Skip("KAFKA_BROKERS not set; skipping integration publisher test")
	}

	broker := cfg.Brokers[0]
	ensureTopicAvailable(t, broker, topics.UserCreated)
	ensureTopicAvailable(t, broker, topics.AgentCreated)

	cfg.ServiceName = "integration-publisher"
	cfg.ClientID = "integration-publisher"
	cfg.Debug = os.Getenv("DEBUG") == "true"

	pub, err := publisher.New(cfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	defer pub.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userEvent := events.NewEvent(
		topics.UserCreated,
		"integration-test",
		events.UserCreatedEvent{
			UserID:    "user-123",
			TenantID:  "tenant-456",
			Email:     "user@example.com",
			Name:      "John Doe",
			Role:      "member",
			CreatedAt: time.Now().UTC(),
		},
	).WithTenantID("tenant-456").WithUserID("user-123")

	publishWithRetry(t, pub, ctx, topics.UserCreated, userEvent)

	batchEvents := []*types.Event{
		events.NewEvent(topics.AgentCreated, "integration-test", events.AgentCreatedEvent{
			AgentID:   "agent-1",
			TenantID:  "tenant-789",
			Name:      "Customer Support Agent",
			Channel:   "voice",
			Model:     "gpt-4",
			CreatedBy: "user-123",
			CreatedAt: time.Now().UTC(),
		}),
		events.NewEvent(topics.AgentCreated, "integration-test", events.AgentCreatedEvent{
			AgentID:   "agent-2",
			TenantID:  "tenant-789",
			Name:      "Sales Agent",
			Channel:   "chat",
			Model:     "gpt-4",
			CreatedBy: "user-123",
			CreatedAt: time.Now().UTC(),
		}),
	}

	publishBatchWithRetry(t, pub, ctx, topics.AgentCreated, batchEvents)
}

func ensureTopicAvailable(t *testing.T, broker, topic string) {
	t.Helper()

	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		t.Fatalf("failed to dial kafka broker %s: %v", broker, err)
	}
	defer conn.Close()

	deadline := time.Now().Add(10 * time.Second)
	if err := conn.SetDeadline(deadline); err != nil {
		t.Fatalf("failed to set deadline for broker %s: %v", broker, err)
	}

	if err := conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}); err != nil && !strings.Contains(err.Error(), "Topic with this name already exists") {
		t.Fatalf("failed to create topic %s: %v", topic, err)
	}

	partitions, err := conn.ReadPartitions(topic)
	if err != nil {
		t.Fatalf("failed to read partitions for topic %s: %v", topic, err)
	}

	if len(partitions) == 0 {
		t.Fatalf("no partitions available for topic %s", topic)
	}

	waitForLeader(t, broker, topic)
}

func waitForLeader(t *testing.T, broker, topic string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for attempt := 1; time.Now().Before(deadline); attempt++ {
		conn, err := kafka.DialLeader(context.Background(), "tcp", broker, topic, 0)
		if err == nil {
			conn.Close()
			return
		}

		t.Logf("[warn] topic %s leader not ready (attempt %d): %v", topic, attempt, err)
		time.Sleep(300 * time.Millisecond)
	}

	t.Fatalf("leader for topic %s not ready after retries", topic)
}

func publishWithRetry(t *testing.T, pub *publisher.Publisher, ctx context.Context, topic string, event *types.Event) {
	t.Helper()

	for attempt := 1; attempt <= 5; attempt++ {
		if err := pub.Publish(ctx, topic, event); err != nil {
			if strings.Contains(err.Error(), "Unknown Topic") && attempt < 5 {
				t.Logf("[warn] publish attempt %d failed with unknown topic; retrying", attempt)
				time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
				continue
			}
			t.Fatalf("failed to publish event after %d attempts: %v", attempt, err)
		}

		return
	}
}

func publishBatchWithRetry(t *testing.T, pub *publisher.Publisher, ctx context.Context, topic string, events []*types.Event) {
	t.Helper()

	for attempt := 1; attempt <= 5; attempt++ {
		if err := pub.PublishBatch(ctx, topic, events); err != nil {
			if strings.Contains(err.Error(), "Unknown Topic") && attempt < 5 {
				t.Logf("[warn] batch publish attempt %d failed with unknown topic; retrying", attempt)
				time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
				continue
			}
			t.Fatalf("failed to publish batch after %d attempts: %v", attempt, err)
		}

		return
	}
}
