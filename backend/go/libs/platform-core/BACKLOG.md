# platform-core — Backlog (en-US)

## Status snapshot
- Audit complete: core helpers exist (config, logger, health, secrets) but are minimal; no tracing/metrics hooks, no redaction or request/tenant context, config lacks prefixes/validation/reload, health is plain text, and secrets only support env provider.

## Audit findings
- Config: uses global Viper without namespace; no required-field enforcement unless callers invoke `ValidateRequired`; defaults are sparse; no duration/URL parsing or structured sections for timeouts/otel; no reload/watch; Kafka broker parsing only via comma env; errors lack context; no secret masking or config prefix.
- Logger: thin zap wrapper with fixed JSON encoding; no sampling, redaction, request/tenant correlation, trace/span context helpers, stdlib redirect, or OTEL/Prometheus integration; time format not configurable.
- Health: single handler with plain-text body; no per-check details/duration, no timeouts, no metrics, no cache window, and readiness checks run sequentially without isolation.
- Secrets: only env provider; no provider chain/file support; errors leak env keys; no caching/TTL; provider swap is global without guard; no typed accessors.
- Docs: README/guide claim broader usage but lack tracing/metrics/redaction examples and env matrix for all knobs.

## Action items
1) Config: add env prefix (e.g., SERPHONA_), duration/size parsing, required validation with typed errors, secret masking in logs, YAML+env list merge, reload hooks, structured sections for server timeouts/db/cache/kafka/otel; provide sample config and env table.
2) Logger: add context helpers (request_id, tenant_id, trace/span), sampling, redaction filters, stdlib bridge, optional OTEL log exporter, configurable encoding/time format.
3) HTTP/gRPC middleware: request/trace/tenant IDs, panic recovery, timeouts/deadlines, gzip/body limits, CORS; standard error mappers (HTTP status ↔ gRPC codes) and response writers.
4) Health: JSON output with per-check status/duration and version/build info, cache TTL, per-check timeout, metrics hooks; separate liveness/readiness handlers.
5) Secrets: built-in provider chain (env → file → custom), caching with TTL, env prefix support, redacted errors, thread-safe provider swap, typed helpers for common keys; optional lint to detect leaked secret names in errors.
6) Observability: OTEL trace/metric helpers, Prometheus HTTP handler, common attributes (service/env/tenant), log/trace correlation utilities.
7) Lifecycle: graceful shutdown helpers for HTTP/gRPC (drain timeouts, signal handling) and background worker utilities with backoff/jitter.
8) Docs/examples: align README and implementation guide with new helpers (config usage, logger with context/redaction, health JSON, middleware wiring, secrets chain) and add end-to-end snippets.
9) Governance: changelog/breaking-changes note for core API evolution and versioning guidance for dependents.
10) Testing: unit tests for config merge/validation, logger levels/redaction/sampling, middleware paths (panic, timeout, IDs), health handler statuses/timeouts, secrets providers/caching, observability helpers, lifecycle helpers; examples should compile.

## Config to surface
- HTTP_ADDR/GRPC_ADDR, HTTP_READ/WRITE/TIMEOUTS, DATABASE_URL/REDIS_URL, KAFKA_BROKERS, CLICKHOUSE_HOST/PORT, MINIO_ENDPOINT/ACCESS/SECRET, JWT_SECRET/JWT_EXPIRATION, OTLP_ENDPOINT, LOG_LEVEL, ENVIRONMENT, LOG_SAMPLING, HEALTH_CACHE_TTL, HEALTH_CHECK_TIMEOUT, SECRETS_PROVIDER/FILE_PATH.

## Test coverage checklist
- [ ] Config merge/validation and prefix handling
- [ ] Logger context fields, invalid level, redaction/sampling
- [ ] HTTP/gRPC middleware (panic, timeout, IDs)
- [ ] Health liveness/readiness, JSON body, timeouts
- [ ] Secrets provider chaining/caching and redaction lint
- [ ] Observability hooks (trace/metric wiring)
- [ ] Lifecycle helpers (graceful shutdown/worker backoff)
- [ ] Docs examples compile

## Next steps
- Implement config prefix/validation and health JSON output first; then logger context/redaction and secrets chain; refresh docs/examples after APIs stabilize.
