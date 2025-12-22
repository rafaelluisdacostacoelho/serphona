# Platform Observability Library

> 📊 Complete observability library for tracking interactions, metrics, and logs in Serphona.

## 🎯 Objective

Collect and track **all service interactions** including:
- ✅ Complete conversation flow
- ✅ Agent choices and decisions
- ✅ Customer statements and responses
- ✅ Performance and quality metrics
- ✅ Complete context for analytics

## 🏗️ Open Source Stack

```
┌────────────────────────────────────────────────────────────┐
│                    OBSERVABILITY STACK                     │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ OpenTelemetry│  │  Prometheus  │  │     Loki     │      │
│  │   (Traces)   │  │  (Metrics)   │  │    (Logs)    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│         │                  │                  │            │
│         └──────────────────┼──────────────────┘            │
│                            ▼                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Grafana (Visualization)                 │  │
│  └──────────────────────────────────────────────────────┘  │
│                            │                               │
│                            ▼                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         ClickHouse (Analytics Storage)               │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

### Components:

| Tool | Purpose | Port |
|------|---------|------|
| **OpenTelemetry** | Distributed tracing | - |
| **Prometheus** | Metrics and alerts | 9090 |
| **Loki** | Log aggregation | 3100 |
| **Tempo** | Traces backend | 3200 |
| **Grafana** | Dashboards and visualization | 3000 |
| **ClickHouse** | Analytics storage | 8123 |

## 📦 Library Structure

```
platform-observability/
├── tracing/
│   ├── tracer.go           # OpenTelemetry tracer setup
│   ├── span.go             # Span helpers
│   └── conversation.go     # Conversation flow tracking
├── metrics/
│   ├── prometheus.go       # Prometheus metrics
│   ├── conversation.go     # Conversation metrics
│   └── performance.go      # Performance metrics
├── logging/
│   ├── logger.go           # Zap logger setup
│   ├── middleware.go       # HTTP logging middleware
│   └── conversation.go     # Conversation event logging
├── middleware/
│   ├── http.go             # HTTP instrumentation
│   ├── grpc.go             # gRPC instrumentation
│   └── conversation.go     # Conversation tracking
├── exporter/
│   ├── clickhouse.go       # ClickHouse exporter
│   └── batch.go            # Batch processing
├── types/
│   ├── conversation.go     # Conversation events
│   ├── interaction.go      # Interaction events
│   └── metrics.go          # Metric types
└── config/
    └── config.go           # Configuration
```

## 🚀 Quick Start

### 1. Installation

```bash
go get github.com/serphona/backend/go/libs/platform-observability
```

### 2. Configuration

```go
package main

import (
    "github.com/serphona/backend/go/libs/platform-observability/config"
    "github.com/serphona/backend/go/libs/platform-observability/tracing"
    "github.com/serphona/backend/go/libs/platform-observability/metrics"
    "github.com/serphona/backend/go/libs/platform-observability/logging"
)

func main() {
    // Configure observability
    cfg := config.Config{
        ServiceName:    "agent-orchestrator",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        
        // Tracing
        TracingEnabled:  true,
        TracingEndpoint: "http://tempo:4317",
        
        // Metrics
        MetricsEnabled: true,
        MetricsPort:    9090,
        
        // Logging
        LoggingEnabled:  true,
        LogLevel:        "info",
        LokiEndpoint:    "http://loki:3100",
        
        // ClickHouse
        ClickHouseEnabled:  true,
        ClickHouseEndpoint: "http://clickhouse:8123",
    }
    
    // Initialize tracing
    tracer, err := tracing.New(cfg)
    if err != nil {
        panic(err)
    }
    defer tracer.Shutdown()
    
    // Initialize metrics
    metrics.Init(cfg)
    
    // Initialize logging
    logger := logging.New(cfg)
    defer logger.Sync()
    
    // Your application...
}
```

### 3. Conversation Tracking

```go
import (
    obs "github.com/serphona/backend/go/libs/platform-observability"
    "github.com/serphona/backend/go/libs/platform-observability/types"
)

// Start conversation
conversationID := obs.StartConversation(ctx, types.ConversationStart{
    TenantID:     "tenant-123",
    AgentID:      "agent-456",
    CustomerID:   "customer-789",
    Channel:      "voice",
    Language:     "en-US",
    StartTime:    time.Now(),
})

// Track interaction
obs.TrackInteraction(ctx, conversationID, types.Interaction{
    Type:       "agent_message",
    Speaker:    "agent",
    Content:    "Hello, how can I help you?",
    Timestamp:  time.Now(),
    Metadata: map[string]string{
        "sentiment": "neutral",
        "intent":    "greeting",
    },
})

// Track customer response
obs.TrackInteraction(ctx, conversationID, types.Interaction{
    Type:       "customer_message",
    Speaker:    "customer",
    Content:    "I need help with my account",
    Timestamp:  time.Now(),
    Metadata: map[string]string{
        "sentiment": "neutral",
        "intent":    "account_support",
    },
})

// Track agent decision
obs.TrackDecision(ctx, conversationID, types.Decision{
    DecisionType: "transfer",
    Option:       "technical_support",
    Reason:       "Customer needs technical assistance",
    Timestamp:    time.Now(),
})

// End conversation
obs.EndConversation(ctx, conversationID, types.ConversationEnd{
    EndTime:    time.Now(),
    Resolution: "transferred",
    Rating:     5,
    Tags:       []string{"account", "technical"},
})
```

## 📊 Collected Metrics

### Conversations

```go
// Total conversations
conversation_total{tenant_id, agent_id, channel} counter

// Conversation duration
conversation_duration_seconds{tenant_id, agent_id, resolution} histogram

// Interactions per conversation
conversation_interactions_total{tenant_id, speaker_type} histogram

// Resolution rate
conversation_resolution_rate{tenant_id, agent_id, resolution_type} gauge

// Customer satisfaction
conversation_customer_rating{tenant_id, agent_id} histogram
```

### Performance

```go
// Agent response time
agent_response_time_seconds{agent_id, tenant_id} histogram

// API latency
http_request_duration_seconds{method, path, status} histogram

// Error rate
error_rate{service, error_type} counter
```

### Quality

```go
// Average sentiment
conversation_sentiment_score{tenant_id, agent_id} gauge

// Assertiveness
agent_assertiveness_score{agent_id} gauge

// Compliance
conversation_compliance_score{tenant_id, policy} gauge
```

## 📝 Event Logging

### Log Structure

```json
{
  "timestamp": "2025-12-01T02:00:00Z",
  "level": "info",
  "service": "agent-orchestrator",
  "tenant_id": "tenant-123",
  "conversation_id": "conv-456",
  "event_type": "interaction",
  "speaker": "agent",
  "content": "How can I help you?",
  "metadata": {
    "sentiment": "positive",
    "intent": "greeting",
    "confidence": 0.95
  },
  "trace_id": "abc123",
  "span_id": "def456"
}
```

### Tracked Events

- ✅ **conversation.started** - Conversation start
- ✅ **interaction.agent** - Agent statement
- ✅ **interaction.customer** - Customer statement
- ✅ **decision.made** - Decision made
- ✅ **transfer.initiated** - Transfer initiated
- ✅ **conversation.ended** - Conversation end
- ✅ **error.occurred** - Error detected
- ✅ **compliance.violation** - Policy violation

## 🔍 Distributed Tracing

### Trace Example

```
Conversation Flow (conv-123)
│
├─ span: conversation.start (100ms)
│  │
│  ├─ span: agent.greeting (50ms)
│  │  └─ event: agent_message
│  │
│  ├─ span: customer.response (2s)
│  │  └─ event: customer_message
│  │
│  ├─ span: intent.detection (150ms)
│  │  └─ event: intent_classified
│  │
│  ├─ span: agent.response (100ms)
│  │  └─ event: agent_message
│  │
│  └─ span: conversation.end (50ms)
     └─ event: conversation_ended
```

### Span Attributes

```go
span.SetAttributes(
    attribute.String("tenant_id", "tenant-123"),
    attribute.String("conversation_id", "conv-456"),
    attribute.String("agent_id", "agent-789"),
    attribute.String("customer_id", "cust-012"),
    attribute.String("channel", "voice"),
    attribute.String("language", "en-US"),
    attribute.Int("interaction_count", 15),
    attribute.Float64("duration_seconds", 120.5),
    attribute.String("resolution", "solved"),
    attribute.Int("rating", 5),
)
```

## 🎨 Grafana Dashboards

### Conversations Dashboard

```yaml
- Active conversations in real-time
- Conversation rate per hour
- Average duration per channel
- Resolution distribution
- Top agents by volume
- Satisfaction rate
```

### Quality Dashboard

```yaml
- Average sentiment per tenant
- Compliance score
- Average response time
- Transfer rate
- Top detected intents
- Policy violations
```

### Performance Dashboard

```yaml
- Latency p50, p95, p99
- Error rate
- Request throughput
- Resource utilization
- SLA tracking
```

## 🔌 Analytics Integration

### ClickHouse Export

```go
// Configure exporter
exporter := clickhouse.NewExporter(clickhouse.Config{
    Endpoint: "http://clickhouse:8123",
    Database: "analytics",
    BatchSize: 1000,
    FlushInterval: 10 * time.Second,
})

// Export conversation
exporter.ExportConversation(conversation)

// Export aggregated metrics
exporter.ExportMetrics(metrics)
```

### ClickHouse Schema

```sql
-- Conversations table
CREATE TABLE conversations (
    conversation_id String,
    tenant_id String,
    agent_id String,
    customer_id String,
    channel String,
    start_time DateTime,
    end_time DateTime,
    duration_seconds Float64,
    interaction_count UInt32,
    resolution String,
    rating UInt8,
    tags Array(String),
    metadata Map(String, String)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(start_time)
ORDER BY (tenant_id, start_time);

-- Interactions table
CREATE TABLE interactions (
    interaction_id String,
    conversation_id String,
    tenant_id String,
    timestamp DateTime,
    speaker_type String,
    content String,
    sentiment String,
    intent String,
    confidence Float64,
    metadata Map(String, String)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (conversation_id, timestamp);
```

## 📦 Dependencies

```go
require (
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.21.0
    go.opentelemetry.io/otel/sdk v1.21.0
    github.com/prometheus/client_golang v1.17.0
    go.uber.org/zap v1.26.0
    github.com/ClickHouse/clickhouse-go/v2 v2.15.0
    github.com/grafana/loki-client-go v0.0.0-20230116142646-e7494d0ef70c
)
```

## 🧪 Testing

```bash
# Run tests
go test ./...

# Tests with coverage
go test -cover ./...

# Integration tests
go test -tags=integration ./...
```

## 📈 Grafana Dashboards

- Metrics overview at [grafana/observability-overview.json](backend/go/libs/platform-observability/grafana/observability-overview.json) (Prometheus datasource `DS_PROM`, `tenant` variable).
- Traces and logs at [grafana/traces-and-logs.json](backend/go/libs/platform-observability/grafana/traces-and-logs.json) (Tempo datasource `DS_TEMPO`, Loki `DS_LOKI`, filters `service` and `tenant`).

## 🛠️ `obsctl` CLI

Minimal CLI to query Tempo via TraceQL or fetch a specific trace.

```bash
# Search traces by service
go run ./cmd/obsctl --tempo-url http://tempo:3200 --tenant acme search --service agent-orchestrator --limit 20

# Search with custom TraceQL
go run ./cmd/obsctl --tempo-url http://tempo:3200 search --query '{ service.name = "agent-orchestrator" && duration > 500ms }'

# Get a trace
go run ./cmd/obsctl --tempo-url http://tempo:3200 --tenant acme get --trace-id <trace-id>
```

## 🚨 Anomaly detection and ML alerts

- Enable with `ANOMALY_DETECTION_ENABLED=true`; tune window (`ANOMALY_WINDOW_MINUTES`), bucket (`ANOMALY_BUCKET_MINUTES`), z-score threshold (`ANOMALY_ZSCORE_THRESHOLD`) and minimum events (`ANOMALY_MIN_COUNT`).
- Anomalies emit `anomaly.detected` (logger, Kafka, Loki) with score, mean, std.
- Set `ML_ALERTS_ENABLED=true` to emit `alert.ml` candidates referencing detected anomalies.

## 📚 Related Documentation

- [Analytics Query Service](../../services/analytics-query-service/README.md)
- [Analytics Processor Service](../../../python/analytics-processor-service/README.md)
- [Observability Guide](../../../docs/architecture/OBSERVABILITY.md)

---

**Version**: 1.0.0  
**License**: Proprietary
