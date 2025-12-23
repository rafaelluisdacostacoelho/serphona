# platform-events — Backlog (en-US)

## Status snapshot
- Audit complete: publisher/consumer are minimal (no metrics/tracing/DLQ), rely on kafka-go defaults, and only hydrate headers into the payload. Tenant/namespace/version are not enforced; security/backpressure/config docs are thin.

## Audit findings
- Envelope: `types.Event` accepts arbitrary `Data` and does not validate `tenant_id` or `version`; headers are optional and metadata is not promoted to headers (only hydrated on consume).
- Publisher: no metrics/tracing; no idempotency hints; retries are only kafka-go writer attempts; topic ensure uses the first broker only with fixed partition/replication (1/1) and no admin ACL/TLS validation.
- Consumer: no DLQ; handler errors with auto-commit still commit; retry is tight loop; no backpressure/pause/resume; no context propagation; filters are positional per handler.
- Config: missing DLQ topic/retention, backoff jitter, max in-flight controls, TLS CA/cert config, SASL mechanism validation in docs; env vars are incomplete.
- Tests/docs: no integration harness with Kafka; examples omit tenant/trace enforcement; no fakes for writer/reader stats/headers.

## Action items
1) Envelope enforcement: add `namespace`, `tenant_id` requirement (except system/internal topics), `version` defaulting/validation, and helpers for header binding; consider schema registry hooks.
2) Publisher resiliency: add explicit retry/backoff with jitter and classification; support DLQ topic + idempotent key builder; expose metrics (publish success/fail/latency/bytes) and OpenTelemetry spans; allow topic replication/partition config and admin creation across brokers.
3) Consumer resiliency: add DLQ routing with poison-pill thresholds; configurable max in-flight and pause/resume on failures; optional manual commit per handler outcome; surface lag metrics and trace propagation into handler context.
4) Security/config: support TLS CA/cert/key paths and SASL mechanism docs; env table for all knobs (batch sizes, timeouts, commit interval, retry/backoff, DLQ, min/max bytes, concurrency); sane defaults per env.
5) Docs/examples: update README/guide to show tenant/trace headers, DLQ usage, per-service bootstrap snippets, and event naming/versioning guidance.
6) Testing: add writer/reader fakes with header assertions; unit tests for envelope validation and handler retry/backoff; integration tests via docker-compose.tests for publish/consume/DLQ with TLS/SASL matrix.

## Config to surface
- KAFKA_BROKERS, KAFKA_GROUP_ID, KAFKA_CLIENT_ID/SERVICE_NAME, KAFKA_AUTO_COMMIT, KAFKA_COMMIT_INTERVAL, KAFKA_SESSION_TIMEOUT, KAFKA_PUBLISHER_BATCH_SIZE/TIMEOUT/MAX_RETRIES/RETRY_INTERVAL, KAFKA_CONSUMER_MAX_RETRIES/RETRY_INTERVAL/CONCURRENCY, KAFKA_DLQ_TOPIC, KAFKA_MAX_INFLIGHT, KAFKA_MIN_BYTES/MAX_BYTES, KAFKA_TLS*, KAFKA_SASL*.

## Test coverage checklist
- [ ] Envelope validation + header hydration
- [ ] Publisher retry/backoff + DLQ/idempotency
- [ ] Consumer manual commit/backpressure + DLQ
- [ ] Metrics/tracing propagation
- [ ] TLS/SASL config paths (positive/negative)

## Next steps
- Prioritize envelope validation + DLQ plumbing; then add metrics/tracing and publish/consume integration harness (compose). Update docs/examples once APIs stabilize.
