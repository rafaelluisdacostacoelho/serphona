//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/config"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/events"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/publisher"
	"github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/topics"
)

// TestIntegrationToolingAndSystem publishes a minimal set of tooling/system events
// to ensure schemas serialize and Kafka accepts them when brokers are available.
func TestIntegrationToolingAndSystem(t *testing.T) {
	cfg := config.LoadFromEnv()
	if len(cfg.Brokers) == 0 {
		t.Skip("KAFKA_BROKERS not set; skipping")
	}

	cfg.ServiceName = "integration-tooling-system"
	cfg.ClientID = "integration-tooling-system"

	pub, err := publisher.New(cfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	defer pub.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tooling events
	toolRegistered := events.NewEvent(
		topics.ToolRegistered,
		"integration-test",
		events.ToolRegisteredEvent{
			ToolID:       "tool-123",
			TenantID:     "tenant-xyz",
			Name:         "crm-plugin",
			Version:      "1.2.3",
			RegisteredAt: time.Now().UTC(),
			RegisteredBy: "user-abc",
			Metadata: map[string]string{
				"vendor": "acme",
			},
		},
	).WithTenantID("tenant-xyz").WithUserID("user-abc")

	toolFailed := events.NewEvent(
		topics.ToolFailed,
		"integration-test",
		events.ToolFailedEvent{
			ToolID:     "tool-123",
			TenantID:   "tenant-xyz",
			Action:     "sync_contacts",
			Error:      "timeout",
			DurationMs: 1500,
			FailedAt:   time.Now().UTC(),
			Context:    "job=contacts-sync",
		},
	).WithTenantID("tenant-xyz")

	// System events
	systemHealth := events.NewEvent(
		topics.SystemHealthCheck,
		"integration-test",
		events.SystemHealthCheckEvent{
			Service:   "billing-service",
			Status:    "ready",
			CheckedAt: time.Now().UTC(),
			Details: map[string]string{
				"db": "ok",
			},
		},
	)

	systemAlert := events.NewEvent(
		topics.SystemAlert,
		"integration-test",
		events.SystemAlertEvent{
			AlertID:   "alert-1",
			Severity:  "critical",
			Service:   "analytics-query-service",
			Message:   "Kafka lag above threshold",
			CreatedAt: time.Now().UTC(),
			Labels: map[string]string{
				"tenant_id": "tenant-xyz",
			},
		},
	)

	configUpdated := events.NewEvent(
		topics.ConfigurationUpdated,
		"integration-test",
		events.ConfigurationUpdatedEvent{
			Service:   "tenant-manager",
			UpdatedBy: "user-ops",
			UpdatedAt: time.Now().UTC(),
			Changes: map[string]string{
				"feature_x": "enabled",
			},
		},
	)

	if err := pub.Publish(ctx, topics.ToolRegistered, toolRegistered); err != nil {
		t.Fatalf("failed to publish tool.registered: %v", err)
	}

	if err := pub.Publish(ctx, topics.ToolFailed, toolFailed); err != nil {
		t.Fatalf("failed to publish tool.failed: %v", err)
	}

	if err := pub.Publish(ctx, topics.SystemHealthCheck, systemHealth); err != nil {
		t.Fatalf("failed to publish system.health.check: %v", err)
	}

	if err := pub.Publish(ctx, topics.SystemAlert, systemAlert); err != nil {
		t.Fatalf("failed to publish system.alert: %v", err)
	}

	if err := pub.Publish(ctx, topics.ConfigurationUpdated, configUpdated); err != nil {
		t.Fatalf("failed to publish system.configuration.updated: %v", err)
	}
}
