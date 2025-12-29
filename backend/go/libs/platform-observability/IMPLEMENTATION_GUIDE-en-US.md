# Platform Observability — Implementation Guide

How to wire tracing, counters, and conversation events into a Go service using this library.

## Prerequisites
- OTLP backend reachable (Tempo/collector) at `TRACING_ENDPOINT`.
- Prometheus scraping the metrics port/path you expose.
- Optional: Kafka brokers and topic for event export; Loki endpoint for log/event push.
- Set `SERVICE_NAME`, `SERVICE_VERSION`, and `ENVIRONMENT` to tag spans/metrics consistently.

## Integration steps
- Add the dependency: `go get github.com/serphona/backend/go/libs/platform-observability`.
- Load configuration from environment:
  ```bash
  export SERVICE_NAME=agent-orchestrator
  export ENVIRONMENT=production
  export TRACING_ENDPOINT=tempo:4317
  export METRICS_ENABLED=true
  export METRICS_PORT=9090
  # Optional exporters
  export KAFKA_ENABLED=true
  export KAFKA_BROKERS=broker-1:9092,broker-2:9092
  export LOKI_ENABLED=true
  export LOKI_ENDPOINT=http://loki:3100
  ```
  ```go
  cfg := config.LoadFromEnv()
  cfg.ServiceVersion = "1.0.0"
  observer, err := observability.Init(cfg)
  if err != nil {
      log.Fatal(err)
  }
  defer observer.Shutdown(context.Background())
  ```
- Instrument servers:
  - HTTP: `handler := middleware.HTTP(mux, cfg.ServiceName); http.ListenAndServe(":8080", handler)`.
  - gRPC: pass `middleware.GRPCUnary()` and `middleware.GRPCStream()` into `grpc.NewServer` interceptors.
- Emit conversation events where business logic occurs:
  ```go
  id := observability.StartConversation(ctx, types.ConversationStart{TenantID: tenantID, AgentID: agentID, Channel: "voice"})
  observability.TrackInteraction(ctx, id, types.Interaction{Speaker: "agent", Content: "Hello"})
  observability.TrackDecision(ctx, id, types.Decision{DecisionType: "transfer", Option: "technical_support"})
  observability.EndConversation(ctx, id, types.ConversationEnd{Resolution: "transferred", Rating: 5})
  ```
- Enable exporters when needed:
  - Kafka: set `KAFKA_ENABLED=true`, `KAFKA_BROKERS`, `KAFKA_TOPIC` (defaults to `observability.events`), TLS/SASL via `KAFKA_TLS_*` and `KAFKA_SASL_*`.
  - Loki: set `LOKI_ENABLED=true`, `LOKI_ENDPOINT`, and optionally `LOKI_TENANT` for `X-Scope-OrgID`.
- Anomaly detection (optional): set `ANOMALY_DETECTION_ENABLED=true` and tune `ANOMALY_WINDOW_MINUTES`, `ANOMALY_BUCKET_MINUTES`, `ANOMALY_ZSCORE_THRESHOLD`, `ANOMALY_MIN_COUNT`, `ANOMALY_COOLDOWN_SECONDS`.

## Operational checks
- Metrics: `curl http://localhost:$METRICS_PORT$METRICS_PATH` returns Prometheus text with `obs_conversation_events_total`, `obs_interactions_total`, `obs_decisions_total`.
- Traces: spans tagged with `service.name`, `service.version`, and `deployment.environment` appear in your OTLP backend; adjust sampling via `TRACING_SAMPLER` and `TRACING_SAMPLER_STRATEGY`.
- Exporters: confirm Kafka topic receives JSON events; Loki push returns HTTP 2xx.

## Testing
- Unit tests: `go test ./...`.
- Integration placeholder: `go test -tags=integration ./test/integration` (currently skipped until coverage is added).
- Manual validation: start the service, hit an instrumented endpoint, verify metrics and traces, and check Kafka/Loki delivery when enabled.

## Limitations to keep in mind
- Only counters are emitted (no HTTP/gRPC latency histograms or runtime/process metrics).
- Conversation state is in-memory; there is no persistence or ClickHouse exporter in this package.
- Kafka/Loki exporters lack retries/backpressure handling; size your pipelines accordingly.
- Middleware does not enrich spans with tenant/request IDs beyond what you pass in events.
