# platform-observability — Backlog (en-US)

## Status snapshot
- Audit complete: tracing initializes OTLP gRPC with basic resource and adaptive sampler; metrics only expose a few conversation counters and a Prometheus handler; middleware just wraps otelhttp/otelgrpc defaults; exporters for Kafka/Loki are minimal with no retries/backpressure/TLS auth; no logging package despite README claims; no HTTP/gRPC latency metrics, no dashboards/alerts, no config validation.

## Audit findings
- Config: large surface but no validation, defaults are permissive (tracing enabled even without endpoint), no env prefix, no required checks, no separation between per-service and exporter configs, no sampling bounds check, no secret masking.
- Tracing: OTLP gRPC only; no propagator configuration (baggage/tracecontext), no error/status mapping helpers, no tenant/user attributes, no server/client span helpers, no shutdown hook wiring, no OTLP HTTP option; sampler ratio unchecked.
- Metrics: only three counters (conversation/interaction/decision) with tenant labels; no HTTP/gRPC latency/error histograms, no process/go runtime metrics registration, no cardinality safeguards, no request/trace correlation helpers, no Kafka/ClickHouse/exporter metrics.
- Middleware: HTTP/gRPC wrappers without tenant/request ID enrichment, no custom span names, no metrics, no panic/timeout logging, no filters.
- Exporters: Kafka exporter has no topic creation, no retries/backoff, no TLS/SASL beyond plain, no batching; Loki exporter lacks TLS/auth options beyond tenant header, no retries or backoff, and no label normalization. No ClickHouse exporter present despite README.
- Types/features: conversation/interaction types exist but no validation or pii controls; anomaly/ML flags in config unused in code.
- Docs: README overstates components (logging, clickhouse, conversation tracking) not present; missing config tables and dashboard/alert references; no examples for otelhttp/otelgrpc setup.
- Testing: no unit/integration tests for tracing/middleware/metrics/exporters; no golden traces/metrics; no config validation tests.

## Action items
1) Config: add env prefix and validation with typed errors; enforce required OTLP endpoint when tracing enabled; clamp sampler ratios; mask secrets in logs; split config sections (tracing/metrics/exporters) and provide defaults per env.
2) Tracing: expose propagator setup (W3C tracecontext + baggage), OTLP HTTP option, resource attrs (service/env/version/tenant optional), server/client span helpers with status mapping, shutdown hook helper, and sampling presets per env.
3) Metrics: add HTTP/gRPC latency/error histograms, request counters, exporter success/failure metrics (Kafka/Loki), process/go runtime registration, cardinality guidance; allow tenant/request labels with opt-in.
4) Middleware: HTTP/gRPC interceptors with span naming, tenant/request/trace IDs, panic recovery, error recording, metrics hooks, and optional filters.
5) Exporters: add retries/backoff, batching, TLS/SASL/mTLS options, topic creation or validation for Kafka; Loki TLS/auth headers, backoff, and label sanitizer; expose close/shutdown.
6) Logging: either add zap logger with trace/span correlation and redaction, or remove claims from docs; ensure correlation IDs are propagated.
7) Docs: update README/guide to match features, add config tables (TRACING_*, METRICS_*, KAFKA_*, LOKI_*), quickstart wiring snippets, and links to Grafana dashboards/alerts.
8) Testing: unit tests for config validation, tracing sampler/propagators, middleware span/metric emission, exporter retry/backoff, and golden metrics/traces; add integration tests with OTLP and Prometheus scrape.

## Config to surface
- TRACING_ENABLED/ENDPOINT/SAMPLER/STRATEGY/INSECURE/TLS_CA/CERT/KEY/BEARER, METRICS_ENABLED/PORT/PATH, KAFKA_ENABLED/BROKERS/TOPIC/CLIENT_ID/SASL/TLS, LOKI_ENABLED/ENDPOINT/TENANT/TLS/AUTH, SERVICE_NAME/VERSION/ENVIRONMENT, LOG_LEVEL (if logging added).

## Test coverage checklist
- [ ] Config validation and sampler bounds
- [ ] Tracing propagators and span emission (HTTP/gRPC)
- [ ] Metrics: HTTP/gRPC histograms, exporter metrics, runtime collectors
- [ ] Middleware: panic/error recording, IDs propagation
- [ ] Exporters: Kafka/Loki retry/backoff/TLS
- [ ] Docs examples compile and reflect actual features

## Next steps
- Add config validation + tracing propagators; implement HTTP/gRPC latency metrics and middleware with span naming/IDs; harden Kafka/Loki exporters with retry/TLS; then reconcile README with implemented surface and add tests/golden outputs.
