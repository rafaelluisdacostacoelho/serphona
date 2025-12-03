package main

import (
	"context"
	"log"
	"time"

	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/events"
	"github.com/serphona/serphona/backend/go/libs/platform-events/publisher"
	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

func main() {
	// Criar configuração
	cfg := config.LoadFromEnv()
	cfg.ServiceName = "example-publisher"
	cfg.Debug = true

	// Criar publisher
	pub, err := publisher.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	// Criar evento de usuário criado
	userEvent := events.NewEvent(
		topics.UserCreated,
		"auth-gateway",
		events.UserCreatedEvent{
			UserID:    "user-123",
			TenantID:  "tenant-456",
			Email:     "user@example.com",
			Name:      "John Doe",
			Role:      "member",
			CreatedAt: time.Now(),
		},
	).WithTenantID("tenant-456").
		WithUserID("user-123")

	// Publicar evento
	ctx := context.Background()
	if err := pub.Publish(ctx, topics.UserCreated, userEvent); err != nil {
		log.Fatalf("Failed to publish event: %v", err)
	}

	log.Println("Event published successfully!")

	// Criar evento de tenant criado
	tenantEvent := events.NewEvent(
		topics.TenantCreated,
		"tenant-manager",
		events.TenantCreatedEvent{
			TenantID:  "tenant-789",
			Name:      "Acme Corp",
			Plan:      "professional",
			OwnerID:   "user-123",
			CreatedAt: time.Now(),
		},
	).WithTenantID("tenant-789").
		WithUserID("user-123")

	// Publicar evento
	if err := pub.Publish(ctx, topics.TenantCreated, tenantEvent); err != nil {
		log.Fatalf("Failed to publish tenant event: %v", err)
	}

	log.Println("Tenant event published successfully!")

	// Exemplo de batch publishing
	batchEvents := []*types.Event{
		events.NewEvent(topics.AgentCreated, "agent-orchestrator", events.AgentCreatedEvent{
			AgentID:   "agent-1",
			TenantID:  "tenant-789",
			Name:      "Customer Support Agent",
			Type:      "voice",
			Model:     "gpt-4",
			CreatedBy: "user-123",
			CreatedAt: time.Now(),
		}),
		events.NewEvent(topics.AgentCreated, "agent-orchestrator", events.AgentCreatedEvent{
			AgentID:   "agent-2",
			TenantID:  "tenant-789",
			Name:      "Sales Agent",
			Type:      "chat",
			Model:     "gpt-4",
			CreatedBy: "user-123",
			CreatedAt: time.Now(),
		}),
	}

	if err := pub.PublishBatch(ctx, topics.AgentCreated, batchEvents); err != nil {
		log.Fatalf("Failed to publish batch: %v", err)
	}

	log.Println("Batch published successfully!")

	// Exibir estatísticas
	stats := pub.Stats()
	log.Printf("Publisher stats: Messages=%d, Bytes=%d", stats.Messages, stats.Bytes)
}
