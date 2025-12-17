//go:build integration

package integration

import (
	"testing"

	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/consumer"
	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
)

// TestIntegrationConsumer validates consumer wiring without fetching messages.
// Requires KAFKA_BROKERS to be set; otherwise the test is skipped.
func TestIntegrationConsumer(t *testing.T) {
	cfg := config.LoadFromEnv()
	if len(cfg.Brokers) == 0 {
		t.Skip("KAFKA_BROKERS not set; skipping integration consumer test")
	}

	cfg.ServiceName = "integration-consumer"
	cfg.ClientID = "integration-consumer"
	cfg.GroupID = "integration-consumer-group"

	topicsToConsume := []string{
		topics.UserCreated,
		topics.TenantCreated,
		topics.AgentCreated,
	}

	cons, err := consumer.New(cfg, topicsToConsume)
	if err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}
	defer cons.Close()

	// We intentionally avoid calling Start() here because it requires live topics.
}
