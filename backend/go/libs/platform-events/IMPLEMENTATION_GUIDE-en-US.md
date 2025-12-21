# Implementation Guide - Platform Events

> Step-by-step guide to integrate the platform-events library into your microservices.

## Index

1. [Initial Setup](#initial-setup)
2. [Publisher - Publishing Events](#publisher---publishing-events)
3. [Consumer - Consuming Events](#consumer---consuming-events)
4. [Patterns and Best Practices](#patterns-and-best-practices)
5. [Troubleshooting](#troubleshooting)
6. [Headers Reference](#headers-reference)
7. [Tooling/System Payloads](#toolingsystem-payloads)
8. [Integration Checklist](#integration-checklist)
9. [Next Steps](#next-steps)

---

## Initial Setup

### 1. Add Dependency

```bash
d cd backend/go/services/your-service
go get github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events
```

### 2. Configure Environment Variables

Add to `.env.example` and `.env`:

```env
# Kafka Configuration
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=your-service-group
SERVICE_NAME=your-service
ENVIRONMENT=development
DEBUG=true
```

### 3. Kafka via Docker Compose

Add to your `docker-compose.yml`:

```yaml
services:
  kafka:
    image: bitnami/kafka:3.6
    ports:
      - "9092:9092"
    environment:
      - KAFKA_CFG_NODE_ID=0
      - KAFKA_CFG_PROCESS_ROLES=controller,broker
      - KAFKA_CFG_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@kafka:9093
      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER
    volumes:
      - kafka_data:/bitnami/kafka

volumes:
  kafka_data:
```

---

## Publisher - Publishing Events

### Step 1: Create a Global Publisher

```go
// internal/infrastructure/events/publisher.go
package events

import (
    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/config"
    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/publisher"
)

var globalPublisher *publisher.Publisher

func InitPublisher() error {
    cfg := config.LoadFromEnv()

    pub, err := publisher.New(cfg)
    if err != nil {
        return err
    }

    globalPublisher = pub
    return nil
}

func GetPublisher() *publisher.Publisher {
    return globalPublisher
}

func ClosePublisher() error {
    if globalPublisher != nil {
        return globalPublisher.Close()
    }
    return nil
}
```

### Step 2: Initialize in `main`

```go
// cmd/server/main.go
package main

import (
    "your-service/internal/infrastructure/events"
)

func main() {
    if err := events.InitPublisher(); err != nil {
        log.Fatalf("Failed to init publisher: %v", err)
    }
    defer events.ClosePublisher()

    // Rest of the app...
}
```

### Step 3: Publish from Use Cases

```go
// internal/usecase/user/create_user.go
package user

import (
    "context"

    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/events"
    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/topics"
    "your-service/internal/infrastructure/events"
)

type CreateUserUseCase struct {
    userRepo UserRepository
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, input CreateUserInput) error {
    if err := uc.validate(input); err != nil {
        return err
    }

    user := &User{ /* ... */ }
    if err := uc.userRepo.Create(ctx, user); err != nil {
        return err
    }

    evt := events.NewEvent(
        topics.UserCreated,
        "auth-gateway",
        events.UserCreatedEvent{
            UserID:    user.ID,
            TenantID:  user.TenantID,
            Email:     user.Email,
            Name:      user.Name,
            Role:      user.Role,
            CreatedAt: user.CreatedAt,
        },
    ).WithTenantID(user.TenantID).WithUserID(user.ID)

    if pub := events.GetPublisher(); pub != nil {
        if err := pub.Publish(ctx, topics.UserCreated, evt); err != nil {
            log.Printf("Failed to publish event: %v", err)
        }
    }

    return nil
}
```

---

## Consumer - Consuming Events

### Step 1: Create a Global Consumer

```go
// internal/infrastructure/events/consumer.go
package events

import (
    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/config"
    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/consumer"
    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/topics"
)

var globalConsumer *consumer.Consumer

func InitConsumer(handlers map[string]types.EventHandler) error {
    cfg := config.LoadFromEnv()

    topicsToConsume := []string{
        topics.UserCreated,
        topics.TenantCreated,
        // Add the topics your service consumes
    }

    cons, err := consumer.New(cfg, topicsToConsume)
    if err != nil {
        return err
    }

    for topic, handler := range handlers {
        cons.Subscribe(topic, handler)
    }

    globalConsumer = cons
    return nil
}

func StartConsumer() error {
    if globalConsumer != nil {
        return globalConsumer.Start()
    }
    return nil
}

func CloseConsumer() error {
    if globalConsumer != nil {
        return globalConsumer.Close()
    }
    return nil
}
```

### Step 2: Implement Handlers

```go
// internal/infrastructure/events/handlers/user_created_handler.go
package handlers

import (
    "log"

    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/types"
    "your-service/internal/usecase/billing"
)

type UserCreatedHandler struct {
    createTrialUseCase *billing.CreateTrialSubscriptionUseCase
}

func NewUserCreatedHandler(createTrialUC *billing.CreateTrialSubscriptionUseCase) *UserCreatedHandler {
    return &UserCreatedHandler{createTrialUseCase: createTrialUC}
}

func (h *UserCreatedHandler) Handle(event *types.Event) error {
    log.Printf("Handling UserCreated event: %s", event.ID)

    var payload events.UserCreatedEvent
    if err := json.Unmarshal(event.Data.([]byte), &payload); err != nil {
        return fmt.Errorf("failed to parse event data: %w", err)
    }

    return h.createTrialUseCase.Execute(context.Background(), billing.CreateTrialInput{
        UserID:   payload.UserID,
        TenantID: payload.TenantID,
    })
}
```

### Step 3: Register Handlers in `main`

```go
// cmd/server/main.go
package main

import (
    "your-service/internal/infrastructure/events"
    "your-service/internal/infrastructure/events/handlers"

    "github.com/rafaelluisdacostacoelho/serphona/backend/go/libs/platform-events/topics"
)

func main() {
    createTrialUC := billing.NewCreateTrialSubscriptionUseCase(...)
    userCreatedHandler := handlers.NewUserCreatedHandler(createTrialUC)

    eventHandlers := map[string]types.EventHandler{
        topics.UserCreated: userCreatedHandler.Handle,
    }

    if err := events.InitConsumer(eventHandlers); err != nil {
        log.Fatalf("Failed to init consumer: %v", err)
    }
    defer events.CloseConsumer()

    if err := events.StartConsumer(); err != nil {
        log.Fatalf("Failed to start consumer: %v", err)
    }

    sigterm := make(chan os.Signal, 1)
    signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
    <-sigterm
}
```

---

## Patterns and Best Practices

### 1. Event-First Design
Publish events after state changes:

```go
if len(changes) > 0 {
    evt := events.NewEvent(
        topics.TenantUpdated,
        "tenant-manager",
        events.TenantUpdatedEvent{TenantID: tenant.ID, Changes: changes, UpdatedAt: time.Now()},
    )
    pub.Publish(ctx, topics.TenantUpdated, evt)
}
```

### 2. Idempotency
Ensure handlers are idempotent (store processed IDs, guard with checks).

### 3. Transactional Outbox Pattern
Guarantee DB + event consistency:

```go
// inside a use case
outboxEvent := &OutboxEvent{EventID: generateID(), Topic: topics.OrderCreated, Payload: orderEventData, CreatedAt: time.Now()}
if err := uc.outboxRepo.CreateWithTx(ctx, tx, outboxEvent); err != nil { return err }
```

Workers should read pending outbox rows, publish, then mark as published.

### 4. Dead Letter Queue
Send repeatedly failing events to a DLQ topic.

### 5. Circuit Breaker
Wrap handlers that call external dependencies with a circuit breaker; send to retry/DLQ when open.

---

## Monitoring (logs, metrics)

- **Logs**: set `DEBUG=true` to get publisher/consumer logs. Prefer wiring your service logger (e.g., zap via platform-observability) and wrap publishes/handlers with contextual fields like tenant/user/trace.
- **Metrics**: export kafka-go stats to Prometheus (example below) and scrape with your service registry (attach the same labels you use elsewhere: `service`, `env`, `tenant` when relevant).

```go
// metrics/registry.go
var (
    pubWrites = promauto.NewGauge(prometheus.GaugeOpts{Name: "platform_events_publisher_writes_total", Help: "Total writes reported by kafka writer"})
    pubErrors = promauto.NewGauge(prometheus.GaugeOpts{Name: "platform_events_publisher_errors_total", Help: "Total write errors"})
    consLag   = promauto.NewGauge(prometheus.GaugeOpts{Name: "platform_events_consumer_lag", Help: "Consumer group lag (approx)"})
)

func StartMetrics(pub *publisher.Publisher, cons *consumer.Consumer) {
    go func() {
        ticker := time.NewTicker(15 * time.Second)
        defer ticker.Stop()
        for range ticker.C {
            w := pub.Stats()
            pubWrites.Set(float64(w.Writes))
            pubErrors.Set(float64(w.Errors))

            if cons != nil {
                r := cons.Stats()
                consLag.Set(float64(r.Lag))
            }
        }
    }()
}
```

- Expose the Prometheus handler in your HTTP server (e.g., `/metrics`) and add alerts for high `pubErrors` or growing `consLag`.

## Document published/consumed events

Add a short table to your service README with the topics and payloads you publish/consume so other teams can integrate quickly.

```markdown
### Events

| Direction | Topic | Payload | Notes |
| --- | --- | --- | --- |
| Publish | auth.user.created | events.UserCreatedEvent | Emitted after user creation; carries tenant_id/user_id |
| Consume | tenant.created | events.TenantCreatedEvent | Triggers billing bootstrap |
```

Keep the table in sync with your use cases and link back to [TOPICS-en-US.md](./TOPICS-en-US.md) when you reuse shared payloads.

---

## Troubleshooting

### Events are not published
1. Check Kafka container:
```bash
docker compose ps kafka
docker compose logs kafka
```
2. Validate config:
```go
cfg := config.LoadFromEnv()
if err := cfg.Validate(); err != nil { log.Fatalf("Invalid config: %v", err) }
```
3. Enable debug:
```env
DEBUG=true
```

### Events are not consumed
1. Check topic exists:
```bash
docker compose exec -T kafka kafka-topics --list --bootstrap-server localhost:9092
```
2. Inspect consumer group:
```bash
docker compose exec -T kafka kafka-consumer-groups --describe --group your-service-group --bootstrap-server localhost:9092
```
3. Confirm handler registration:
```go
log.Printf("Registered handlers: %+v", eventHandlers)
```

### Consumer lag
- Increase concurrency: `KAFKA_CONSUMER_CONCURRENCY=10`
- Optimize handlers (async work, batching, avoid slow calls)

### Duplicate events
- Implement idempotency in handlers
- Use transactions where possible
- Store processed event IDs

---

## Headers Reference

| Header | Required | Description |
| --- | --- | --- |
| event_type | Yes | Event type (aligns with topic) |
| source | Yes | Service that published the event |
| version | Yes | Event schema version (default 1.0) |
| tenant_id | Optional | Tenant that owns the event |
| user_id | Optional | User that triggered the event |
| trace_id | Optional | Trace ID for distributed tracing |
| span_id | Optional | Span ID for distributed tracing |

## Tooling/System Payloads

### Tooling
- `tool.registered` → `ToolRegisteredEvent` (tool_id, tenant_id, name, version, registered_at, registered_by?, metadata?)
- `tool.invoked` → `ToolInvokedEvent` (tool_id, tenant_id, action, invoked_at, correlation_id?, payload?)
- `tool.completed` → `ToolCompletedEvent` (tool_id, tenant_id, action, result, duration_ms?, completed_at, correlation_id?)
- `tool.failed` → `ToolFailedEvent` (tool_id, tenant_id, action, error, duration_ms?, failed_at, context?, correlation_id?)

### System
- `system.health.check` → `SystemHealthCheckEvent` (service, status, checked_at, details?)
- `system.error` → `SystemErrorEvent` (service, error, severity?, occurred_at, trace_id?, span_id?, labels?)
- `system.alert` → `SystemAlertEvent` (alert_id, severity, service, message, created_at, labels?)
- `system.configuration.updated` → `ConfigurationUpdatedEvent` (service, updated_by?, updated_at, changes?)

---

## Next Steps

1. Monitor performance: add Prometheus metrics.
2. Add DLQ for repeatedly failing events.
3. Add tracing with platform-observability.
4. Optimize: batching and compression.
5. Scale: increase partitions/consumers.

---

**Questions?** See the [README-en-US](./README-en-US.md) or open an issue.
