# Voice Gateway — Backlog (en-US)

## Status snapshot (code as of now)
- HTTP server only exposes health endpoints from `cmd/server`; internal router with call APIs/webhooks is not wired at all. No initialization of Redis, Kafka, Asterisk, tenant-manager client, STT/TTS providers, or call/conversation services.
- Asterisk webhook handler accepts JSON without any auth/signature and assigns a random tenant ID instead of resolving DID → tenant; event handling is mostly TODOs (conversation start, hangup/end, cleanup).
- Call lifecycle lives solely in Redis with TTL; Kafka publisher exists but is unused; conversation manager and STT/TTS integration are not invoked.

## Critical gaps to address
1) **Bootstrap/wiring**: Build the real server stack in `cmd/server` (Redis client + call state repo, Kafka publisher, Asterisk ARI client, tenant-manager client, STT/TTS providers, call service, conversation manager) and expose the internal HTTP router instead of the placeholder mux.
2) **Auth/tenant enforcement**: Management APIs and webhooks have no auth; CORS is `*`. Replace random tenant UUID with DID lookup via tenant-manager, enforce tenant scoping on all calls/events, and require auth (JWT/service token) on control APIs.
3) **Webhook hardening & idempotency**: Asterisk events lack signature/basic-auth verification, replay protection, and idempotent processing; no mapping from channel → call on StasisEnd/Hangup, and no backoff/retry policy when downstream (Redis/Kafka) fails.
4) **Call lifecycle correctness**: Redis TTL-only storage, no persistence or cleanup, no dedupe, and concurrency control uses a naive SCAN. Transfer/hangup/bridge operations are TODOs; silence/timeouts/inactivity not enforced; no DTMF/IVR handling.
5) **Conversation/agent orchestration**: `StartConversation` is stubbed; conversation manager is in-memory only; no agent-orchestrator integration, no STT/TTS provider selection per tenant, no audio pipeline (silence detection, resampling, buffering) wired into calls.
6) **Events & billing**: Kafka publisher not integrated; no DLQ/idempotence keys, no call/transcription/LLM/TTS/billing usage events emitted, and no correlation fields (tenant/call/channel) on logs/metrics.
7) **Observability & readiness**: Only zap logging and a bare Prometheus handler; no tracing, no metrics for ARI/STT/TTS latency, call states, failures, or queue depths. Readiness probe ignores dependencies (Redis/Kafka/Asterisk).
8) **Security/PII**: No validation/normalization of phone numbers, no input limits, no rate limiting; secrets for Asterisk/Google/ElevenLabs are used directly with no rotation policy; recording/transcript retention and PII masking are undefined.
9) **Testing/runbooks**: No unit/contract tests for ARI webhooks, call state repo, or provider adapters; no fakes for Asterisk/Redis/Kafka; no load or chaos tests. Missing runbooks for telephony outages, DLQ replay, provider throttling, and tenant misconfiguration.

## Configuration to surface/validate
- Required: Asterisk ARI creds/URLs, Redis URL/TTL, Kafka brokers/topic prefix, tenant-manager URL, STT/TTS provider creds/configs, max concurrent calls, timeouts (read/write/call/silence), recording flags, metrics port/path.
- Optional but recommended: retry/backoff settings, circuit breakers, rate limits per tenant, billing/usage toggle, tracing exporter, webhook secrets/auth.

## Test coverage targets
- Webhook auth + idempotency; DID → tenant resolution; tenant-scoped call control APIs.
- Call state transitions (ringing/answered/active/transferred/ended) with Redis repo and Asterisk client fakes.
- STT/TTS provider selection per tenant, retries/backoff/circuit-breaking, and event publication to Kafka.
- Metrics/tracing emitted with call/tenant IDs; readiness failing when deps are down.
- Billing/usage events for call minutes, STT/TTS/LLM consumption.

## Next steps
- Wire the real router/services in `cmd/server`, replace placeholders in handlers (tenant lookup, conversation start, hangup/end), and add auth middleware + CORS tightening.
- Add webhook validation/idempotency and ARI channel→call lookup; implement retry/backoff + DLQ for Kafka publishes.
- Instrument metrics/tracing for ARI/STT/TTS/call state; add readiness checks for Redis/Kafka/Asterisk.
- Create fakes/mocks for Asterisk/Redis/Kafka/providers and cover the above flows with unit/contract tests; add load profiles for concurrent calls.
