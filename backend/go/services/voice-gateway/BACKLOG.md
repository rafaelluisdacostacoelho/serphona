# Voice Gateway — Backlog (en-US)

## Status snapshot (code as of now)
- HTTP server boots the real stack (Redis client + call state repo, Kafka publisher with DLQ attempt, Asterisk ARI client, tenant-manager client, optional STT/TTS providers) and exposes the internal router with auth middleware and CORS allowlist.
- Asterisk webhook handler now enforces basic-auth/HMAC signature (configurable header/secret), replay window, and Redis idempotência; resolve DID → tenant; channel → call mapping used on StasisEnd/Hangup; retries/backoff around incoming call handling and Kafka DLQ on publish failure. Conversation start remains TODO.
- Call lifecycle persisted in Redis with TTL; answer/end flows wired; transfer/bridge, silence/inactivity/DTMF handling still missing. Conversation manager/STT/TTS pipeline still not invoked.
- Metrics server is exposed; readiness checks exist for Redis/Kafka but Asterisk readiness is stubbed; tracing/metrics for ARI/STT/TTS and call transitions still absent.

## Critical gaps to address (prioritized execution order)

- [ ] Auth + response envelope
	- [ ] Garantir todas as rotas de gestao com JWT/service-token (middleware ativo em router)
	- [ ] CORS limitado a `SERVER_CORS_ALLOWED_ORIGINS`
	- [ ] Lookup DID → tenant em webhooks e APIs de chamada (sem UUID aleatorio)
	- [ ] Envelope de resposta padrao (success meta ou error code/message/details/trace_id) em APIs e webhooks

- [ ] Webhook hardening & idempotency
	- [ ] Validar configuracao de assinatura/basic-auth e registrar falhas (logs/metrics)
	- [ ] Enforce replay window por timestamp
	- [ ] Idempotencia via Redis `SETNX` com TTL configuravel
	- [ ] Manter mapeamento canal → chamada para StasisEnd/Hangup
	- [ ] Retry/backoff para Redis/Kafka e publicar em DLQ em caso de falha

- [ ] Call lifecycle correctness
	- [ ] Migrar de TTL-only para persistencia/cleanup e dedupe reais
	- [ ] Controle de concorrencia sem SCAN (locks/chaves deterministicas)
	- [ ] Implementar transfer/hangup/bridge
	- [ ] Timeouts de silencio/inatividade e suporte a DTMF/IVR

- [ ] Conversation/agent orchestration
	- [ ] Finalizar `StartConversation`
	- [ ] Conversation manager persistente e multi-tenant
	- [ ] Integrar agent-orchestrator (selecionar agente)
	- [ ] Selecionar STT/TTS por tenant e ligar pipeline de audio (silence detection, resampling, buffering)

- [ ] Eventos & billing
	- [ ] Emitir eventos de call/transcription/LLM/TTS/billing com chaves de idempotencia
	- [ ] Caminho de DLQ ativo e correlação tenant/call/channel em logs/metrics

- [ ] Observabilidade & readiness
	- [ ] Tracing e metricas para ARI/STT/TTS/estado de chamada/falhas/filas
	- [ ] Readiness real checando Redis/Kafka/Asterisk (sem stub) e expondo motivo de falha

- [ ] Seguranca/PII
	- [ ] Validar/normalizar numeros; limites de tamanho/taxa por rota/tenant
	- [ ] Segredos via config/rotacao (Asterisk/Google/ElevenLabs)
	- [ ] Politicas de retencao de gravacao/transcricao e mascaramento de PII

- [ ] Testes, fakes e runbooks
	- [ ] Testes de webhook auth/idempotencia e DID → tenant
	- [ ] Testes das APIs de controle por tenant e transicoes de estado com fakes ARI/Redis
	- [ ] Testes de selecao STT/TTS com retries/backoff/circuit breaking + publicacao em Kafka
	- [ ] Metricas/tracing e readiness degradando com deps down
	- [ ] Eventos de billing/uso para minutos, STT/TTS/LLM
	- [ ] Fakes de Asterisk/Redis/Kafka/provedores e perfis de carga/caos + runbooks (falhas de telefonia, replay DLQ, throttling, misconfig de tenant)

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
- Follow the global sequencing in [docs/backlogs/GLOBAL-VOICE-PLATFORM-BACKLOG-en-US.md](../../docs/backlogs/GLOBAL-VOICE-PLATFORM-BACKLOG-en-US.md) (pt-BR: [docs/backlogs/GLOBAL-VOICE-PLATFORM-BACKLOG-pt-BR.md](../../docs/backlogs/GLOBAL-VOICE-PLATFORM-BACKLOG-pt-BR.md)).
- Wire the real router/services in `cmd/server`, replace placeholders in handlers (tenant lookup, conversation start, hangup/end), and add auth middleware + CORS tightening.
- Add webhook validation/idempotency and ARI channel→call lookup; implement retry/backoff + DLQ for Kafka publishes.
- Instrument metrics/tracing for ARI/STT/TTS/call state; add readiness checks for Redis/Kafka/Asterisk.
- Create fakes/mocks for Asterisk/Redis/Kafka/providers and cover the above flows with unit/contract tests; add load profiles for concurrent calls.

## Referências
- Este serviço depende do [BACKLOG-AUTH.md](../../../BACKLOG-AUTH.md) para alinhamento com autenticação e autorização multi-tenant.
