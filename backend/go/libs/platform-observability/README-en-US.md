# Platform Observability Library

Lightweight building blocks for Serphona services to emit traces, counters, and conversation events to external backends (OTLP, Prometheus, Kafka, Loki) with optional anomaly detection.

## What this library provides
- OTLP gRPC tracer provider with configurable sampling and TLS/bearer auth options.
- Prometheus counters for conversations, interactions, and decisions, plus a metrics HTTP server.
- HTTP and gRPC middleware wrappers based on `otelhttp` and `otelgrpc`.
- Conversation tracking helpers that keep in-memory state, emit spans/counters, and forward events to Kafka and/or Loki.
- Optional anomaly detection that raises alert events for surges in interaction/decision volume.

## Packages at a glance
```
platform-observability/
├── config/            // env-backed configuration loader
├── tracing/           // OTLP tracer provider setup and adaptive sampler
├── metrics/           // Prometheus counters and handler
├── middleware/        // HTTP and gRPC instrumentation wrappers
├── exporter/          // Kafka and Loki exporters
├── anomaly/           // simple z-score anomaly detector
├── types/             // conversation/interaction/decision structs and events
├── examples/          // conversation_tracking.go usage demo
└── observability.go   // Observer lifecycle and event processing
```

## Quick start
Install the module and initialize observability at service startup.

```bash
go get github.com/serphona/backend/go/libs/platform-observability
```

```go
package main

import (
    "context"
    "log"

    obs "github.com/serphona/backend/go/libs/platform-observability"
    "github.com/serphona/backend/go/libs/platform-observability/config"
    "github.com/serphona/backend/go/libs/platform-observability/middleware"
)

func main() {
    cfg := config.LoadFromEnv()
    cfg.ServiceName = "agent-orchestrator"
    cfg.Environment = "production"
    cfg.TracingEndpoint = "tempo:4317" // required when tracing is enabled

    observer, err := obs.Init(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer observer.Shutdown(context.Background())

    // HTTP server with tracing
    handler := middleware.HTTP(http.DefaultServeMux, cfg.ServiceName)
    go http.ListenAndServe(":8080", handler)

    // gRPC server with tracing
    _ = grpc.NewServer(
        grpc.UnaryInterceptor(middleware.GRPCUnary()),
        grpc.StreamInterceptor(middleware.GRPCStream()),
    )
}
```

## Track conversations and decisions

```go
ctx := context.Background()

id := obs.StartConversation(ctx, types.ConversationStart{
    TenantID: "tenant-123",
    AgentID:  "agent-456",
    Channel:  "voice",
    Language: "en-US",
})

obs.TrackInteraction(ctx, id, types.Interaction{
    Speaker: "agent",
    Content: "Hello, how can I help you?",
    Sentiment: "neutral",
})

obs.TrackDecision(ctx, id, types.Decision{
    DecisionType: "transfer",
    Option:       "technical_support",
})

obs.EndConversation(ctx, id, types.ConversationEnd{
    Resolution: "transferred",
    Rating:     5,
})
```

Each call appends to the in-memory conversation record, emits a span (if tracing is on), bumps Prometheus counters, and forwards events to Kafka/Loki when configured.

## Metrics and exporters
- Prometheus counters exposed at `MetricsPath` (default `/metrics`) on `MetricsPort` (default `9090`):
  - `obs_conversation_events_total{event,tenant}`
  - `obs_interactions_total{speaker,tenant}`
  - `obs_decisions_total{type,tenant}`
- Tracing uses OTLP gRPC; resource attributes include `service.name`, `service.version`, and `deployment.environment`.
- Kafka exporter sends JSON events to a single topic; TLS/SASL PLAIN are supported but no retries/backpressure.
- Loki exporter pushes JSON events with optional `X-Scope-OrgID`; no retries/backoff.

## Configuration (env-first)
| Variable | Purpose | Default |
| --- | --- | --- |
| `SERVICE_NAME`, `SERVICE_VERSION`, `ENVIRONMENT` | Resource attributes for traces and logs | `unknown`, `1.0.0`, `development` |
| `TRACING_ENABLED` | Enable OTLP tracing | `true` |
| `TRACING_ENDPOINT` | OTLP gRPC target (host:port) | `tempo:4317` |
| `TRACING_SAMPLER`, `TRACING_SAMPLER_STRATEGY` | Ratio and strategy (`ratio`, `parent_ratio`, `always_on`, `always_off`, `adaptive`) | `1.0`, `parent_ratio` |
| `TRACING_INSECURE`, `TRACING_TLS_INSECURE` | Disable TLS or skip verification | `true`, `false` |
| `TRACING_TLS_CA_CERT`, `TRACING_TLS_CLIENT_CERT`, `TRACING_TLS_CLIENT_KEY` | Client/CA certificates | empty |
| `TRACING_BEARER_TOKEN` | Bearer token for OTLP requests | empty |
| `METRICS_ENABLED`, `METRICS_PORT`, `METRICS_PATH` | Expose Prometheus handler | `true`, `9090`, `/metrics` |
| `KAFKA_ENABLED`, `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_CLIENT_ID` | Kafka exporter settings | `false`, empty, `observability.events`, `platform-observability` |
| `KAFKA_SASL_USERNAME`, `KAFKA_SASL_PASSWORD`, `KAFKA_SASL_MECHANISM` | SASL PLAIN auth | empty |
| `KAFKA_TLS_ENABLED`, `KAFKA_TLS_INSECURE` | TLS options for Kafka | `false`, `false` |
| `LOKI_ENABLED`, `LOKI_ENDPOINT`, `LOKI_TENANT` | Loki push endpoint and tenant header | `false`, empty, empty |
| `CONVERSATION_TRACKING` | Toggle conversation state machine | `true` |
| `ANOMALY_DETECTION_ENABLED` and `ANOMALY_*` | Enable z-score anomaly alerts; window/bucket/z-score/cooldown knobs | `false`; `5m/1m/3.0/5/300s` |
| `ML_ALERTS_ENABLED` | Emit ML alert placeholder events from anomalies | `false` |

## Limitations to be aware of
- No HTTP/gRPC latency histograms or runtime/process metrics are emitted today.
- No ClickHouse exporter or persisted conversation storage; all state is in-memory per process.
- Kafka/Loki exporters do not implement retries or backpressure handling.
- Logging uses a default production zap logger; there is no logging package or redaction helpers exposed.
- Propagators are the OpenTelemetry defaults; there is no custom tenant/request ID enrichment in middleware.

## Testing
- Run unit tests: `go test ./...`
- Integration placeholder: `go test -tags=integration ./test/integration` (skips until implemented).
- Manual checks: ensure the metrics endpoint responds, traces reach your OTLP backend, and Kafka/Loki receive events when enabled.

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
- [Analytics Processor Service](../../../python/services/analytics-processor-service/README.md)
- [Observability Guide](../../../docs/architecture/OBSERVABILITY.md)

---

**Version**: 1.0.0  
**License**: Proprietary
