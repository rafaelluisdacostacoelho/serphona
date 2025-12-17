# Platform Events Library

> 📬 Shared event/messaging library for asynchronous communication between Serphona microservices.

[🇧🇷 Versão em Português](./README-pt-BR.md)

## Purpose

This library provides a robust event publication/subscription system based on Apache Kafka to enable asynchronous and decoupled communication between Serphona microservices.

## Key Features

- ✅ Event Publisher
- ✅ Event Consumer  
- ✅ Standardized event types
- ✅ Pre-defined topics
- ✅ Centralized configuration
- ✅ Automatic retry
- ✅ Batch processing
- ✅ Event filters
- ✅ Trace context propagation

## Quick Start

### Publishing Events

```go
package main

import (
    "context"
    "github.com/serphona/serphona/backend/go/libs/platform-events/config"
    "github.com/serphona/serphona/backend/go/libs/platform-events/events"
    "github.com/serphona/serphona/backend/go/libs/platform-events/publisher"
    "github.com/serphona/serphona/backend/go/libs/platform-events/topics"
)

func main() {
    cfg := config.LoadFromEnv()
    pub, _ := publisher.New(cfg)
    defer pub.Close()
    
    event := events.NewEvent(
        topics.UserCreated,
        "auth-gateway",
        events.UserCreatedEvent{
            UserID:   "user-123",
            TenantID: "tenant-456",
            Email:    "user@example.com",
        },
    )
    
    pub.Publish(context.Background(), topics.UserCreated, event)
}
```

### Consuming Events

```go
package main

import (
    "log"
    "github.com/serphona/serphona/backend/go/libs/platform-events/config"
    "github.com/serphona/serphona/backend/go/libs/platform-events/consumer"
    "github.com/serphona/serphona/backend/go/libs/platform-events/topics"
    "github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

func main() {
    cfg := config.LoadFromEnv()
    cons, _ := consumer.New(cfg, []string{topics.UserCreated})
    defer cons.Close()
    
    cons.Subscribe(topics.UserCreated, func(event *types.Event) error {
        log.Printf("User created: %+v", event)
        return nil
    })
    
    cons.Start()
}
```

## Documentation

- [Portuguese README](./README-pt-BR.md) - Complete documentation in Portuguese
- [Implementation Guide (PT-BR)](./IMPLEMENTATION_GUIDE-pt-BR.md) - Step-by-step integration guide
- [Examples](./examples/) - Complete usage examples
- [Topics and Payloads](./TOPICS.md) - Topic-to-payload map and pending contracts

## Headers and Metadata

The publisher sets headers for `event_type`, `source`, `version` and, when present, `tenant_id`, `user_id`, `trace_id`, `span_id`. The consumer hydrates these into `types.Event` and stores any extra headers in `Metadata`.

## Typed Payload Helper

Use `types.Bind[T]` to decode `event.Data` into a strong type:

```go
var payload events.UserCreatedEvent
payload, err := types.Bind[events.UserCreatedEvent](event)
```

## Available Topics

### Auth Events
- `auth.user.created`, `auth.user.updated`, `auth.user.deleted`
- `auth.user.logged_in`, `auth.user.logged_out`
- `auth.password.changed`, `auth.password.reset`

### Tenant Events
- `tenant.created`, `tenant.updated`, `tenant.deleted`
- `tenant.suspended`, `tenant.activated`
- `tenant.member.added`, `tenant.member.removed`

### Billing Events
- `billing.subscription.created`, `billing.payment.succeeded`
- `billing.credits.purchased`, `billing.credits.consumed`

### Agent Events
- `agent.created`, `agent.conversation.started`
- `agent.message.sent`, `agent.message.received`

### Analytics Events
- `analytics.interaction.logged`, `analytics.metric.recorded`

### Tool Events
- `tool.registered`, `tool.invoked`, `tool.completed`

### System Events
- `system.health.check`, `system.error`, `system.alert`

## Configuration

### Environment Variables

```env
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=my-service-group
SERVICE_NAME=my-service
ENVIRONMENT=development
DEBUG=true
```

## Architecture

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Service   │─────▶│    Kafka    │◀─────│   Service   │
│     A       │      │   (Topics)  │      │      B      │
│ (Publisher) │      └─────────────┘      │ (Consumer)  │
└─────────────┘                           └─────────────┘
```

## Related Documentation

- [Architecture Guide](../../../docs/architecture/LIBS_VS_SERVICES.md)
- [Auth Gateway](../../services/auth-gateway/README.md)
- [Tenant Manager](../../services/tenant-manager/README.md)

---

**Version**: 1.0.0  
**License**: Proprietary
