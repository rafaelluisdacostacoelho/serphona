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
- [Examples](./examples/) - Usage snippets (payload shaping)
- [Topics and Payloads](./TOPICS-en-US.md) - Topic-to-payload map

## Headers and Metadata

The publisher sets headers for `event_type`, `source`, `version` and, when present, `tenant_id`, `user_id`, `trace_id`, `span_id`. The consumer hydrates these into `types.Event` and stores any extra headers in `Metadata`.

| Header | Required | Description |
| --- | --- | --- |
| event_type | Yes | Event type (matches topic constant) |
| source | Yes | Origin service publishing the event |
| version | Yes | Event schema version (defaults to 1.0) |
| tenant_id | Optional | Tenant that owns the event |
| user_id | Optional | User who triggered the event |
| trace_id | Optional | Trace context for distributed tracing |
| span_id | Optional | Span context for distributed tracing |

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
- `billing.subscription.created`, `billing.subscription.updated`, `billing.subscription.cancelled`
- `billing.payment.succeeded`, `billing.payment.failed`
- `billing.credits.purchased`, `billing.credits.consumed`, `billing.invoice.generated`

### Agent Events
- `agent.created`, `agent.updated`, `agent.deleted`
- `agent.deployed`, `agent.started`, `agent.stopped`
- `agent.conversation.started`, `agent.conversation.ended`
- `agent.message.sent`, `agent.message.received`

### Analytics Events
- `analytics.interaction.logged`, `analytics.metric.recorded`
- `analytics.report.generated`, `analytics.data.exported`

### Tool Events
- `tool.registered`, `tool.invoked`, `tool.completed`, `tool.failed`

### System Events
- `system.health.check`, `system.error`, `system.alert`, `system.configuration.updated`

## Tooling & System payloads

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

## Tooling/System examples

```go
// Tool failed example
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

// Tool invoked example
toolInvoked := events.NewEvent(
    topics.ToolInvoked,
    "integration-test",
    events.ToolInvokedEvent{
        ToolID:        "tool-123",
        TenantID:      "tenant-xyz",
        Action:        "sync_contacts",
        InvokedAt:     time.Now().UTC(),
        CorrelationID: "corr-1",
        Payload: map[string]interface{}{
            "job": "contacts-sync",
        },
    },
).WithTenantID("tenant-xyz")

// System alert example
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

// System error example
systemError := events.NewEvent(
    topics.SystemError,
    "integration-test",
    events.SystemErrorEvent{
        Service:    "analytics-query-service",
        Error:      "timeout contacting ClickHouse",
        Severity:   "error",
        OccurredAt: time.Now().UTC(),
        TraceID:    "trace-123",
        Labels: map[string]string{
            "tenant_id": "tenant-xyz",
        },
    },
)

## Event versioning

- Default version is `1.0` on all events.
- Additive, backward-compatible changes (novos campos opcionais) keep the same version.
- Breaking changes require bumping the version and handling compatibility on consumers.
- Prefer adding optional fields instead of changing or removing existing ones.
```

## Configuration

### Environment Variables

```env
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=my-service-group
SERVICE_NAME=my-service
ENVIRONMENT=development
DEBUG=true
```

## Testing & Coverage

- Unit tests: run `make test` (or `go test ./...`) from `backend/go/libs/platform-events`.
- Coverage report: run `make test-coverage` to generate `coverage.out` and `coverage.html`; view the HTML locally or use `go tool cover -func=coverage.out` for a summary.
- Integration tests: run `make test-integration` (supports `TEST_ARGS="--with-compose"` to start the docker-compose Kafka stack and forward extra `go test` flags).

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
