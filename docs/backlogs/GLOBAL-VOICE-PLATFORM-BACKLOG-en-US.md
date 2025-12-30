# Global Backlog — Voice Platform (en-US)

Purpose: sequence work across services/libs so voice loop (STT → agent → TTS) is robust, multi-tenant, and observable.

## Steps (follow in order)
1) Voice Gateway — core loop & safety  
   - Backlog: [backend/go/services/voice-gateway/backlog.md](../../backend/go/services/voice-gateway/backlog.md)  
   - Scope: wire STT/TTS provider selection per-tenant, open conversation via agent-orchestrator, stream audio (PCM 16k mono) through STT→agent→TTS, and play back. Add webhook auth/idempotency, tenant propagation, envelope errors.  
   - Exit: automated tests with fakes (Asterisk/Redis/Kafka/agent/tenant), metrics/traces, readiness failing when deps down.

2) Tenant Manager — provider config & DID lookup  
   - Backlog: [backend/go/services/tenant-manager/backlog.md](../../backend/go/services/tenant-manager/backlog.md)  
   - Scope: DID→tenant lookup with auth and envelope; per-tenant STT/TTS/LLM provider config surfaced with `X-Tenant-Id`; contract tests; rate limiting and error mapping.  
   - Exit: stable API consumed by voice-gateway; tests pin contract.

3) Agent Orchestrator — service tokens & SLAs  
   - Backlog: [backend/go/services/agent-orchestrator/backlog.md](../../backend/go/services/agent-orchestrator/backlog.md)  
   - Scope: enforce SERVICE_AUDIENCE tokens, timeouts/retries, envelope with trace/request IDs; ensure conversation create/turn/end endpoints are idempotent and return agent metadata (name/voice).  
   - Exit: integration stub in voice-gateway passes without retries flapping; load-safe defaults.

4) Platform Auth — client transport adoption  
   - Backlog: [backend/go/libs/platform-auth/BACKLOG.md](../../backend/go/libs/platform-auth/BACKLOG.md)  
   - Scope: ensure all outbound clients in voice-gateway/tenant-manager/agent-orchestrator use platform-auth transport with tenant/request/trace propagation and SERVICE_AUDIENCE tokens.  
   - Exit: outbound headers verified in tests.

5) Observability & Infra  
   - Backlogs: [backend/go/libs/platform-observability](../../backend/go/libs/platform-observability), [infra/helm](../../infra/helm), [infra/terraform](../../infra/terraform)  
   - Scope: metrics for ARI/STT/TTS/agent calls, call-state counters, readiness checks (Redis/Kafka/Asterisk), Helm values for secrets/creds, HPA thresholds, and tracing exporter.  
   - Exit: readiness fails closed; dashboards/alerts cover latency/error budget.

## Work split per step
- Implement per project backlog items for the step, then promote to next step only when exit criteria met. Keep PT-BR mirror in GLOBAL-VOICE-PLATFORM-BACKLOG-pt-BR.md.
