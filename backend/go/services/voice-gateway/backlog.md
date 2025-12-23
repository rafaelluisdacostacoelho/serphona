# Voice Gateway — Backlog (en-US)

## Status snapshot
- Handles telephony/voice flows; details not reviewed here—needs audit. Likely integrates with SIP/telephony provider and Agent Orchestrator.

## Open items
1) **Auth & tenant**: ensure calls/events carry tenant_id; enforce ACLs on call control APIs.
2) **Telephony provider**: secure webhooks (sig validation), retries/backoff; idempotency for call events; handle dual-channel media if applicable.
3) **Call flows**: DTMF/IVR handling, transcription pipeline hooks; handoff to Agent Orchestrator/tools with context.
4) **Quality/latency**: tune timeouts, jitter buffers; monitor p95/p99 for media/control paths.
5) **Observability**: metrics for call success/fail, drop reasons, ASR latency; tracing with call/tenant IDs; structured logs.
6) **Billing**: cost per minute/call; emit usage events; track tool/LLM usage triggered via voice.
7) **Resilience**: circuit breakers for downstream ASR/TTS/LLM; DLQ for failed events; retry policies for provider callbacks.
8) **Security/PII**: mask sensitive data; storage/retention policies for recordings/transcripts; encryption at rest/in transit.
9) **Testing**: contract tests for webhooks/call events; load tests for concurrent calls; failover drills.
10) **Runbooks**: telephony provider outage, DLQ replay, ASR/TTS degradation, rate-limit adjustments.

## Config to surface
- Provider creds/webhook secrets; callback URLs; media settings.
- ASR/TTS provider configs; timeouts; retries/backoff; circuit settings.
- Auth scopes; logging/tracing exporters; metrics port.

## Test coverage checklist
- [ ] Webhook validation/idempotency
- [ ] Tenant/ACL enforcement
- [ ] ASR/TTS retries/circuit
- [ ] Metrics/traces present
- [ ] Billing events emitted

## Next suggested steps
- Audit code/config to replace placeholders; add webhook validation tests and observability if missing.
