# Voice Gateway Design - Serphona Platform (en-US)

## 1. Overview
`voice-gateway` bridges telephony (Asterisk/ARI) with the AI agent orchestration layer. It handles real-time voice, streams audio to/from STT/TTS, and coordinates with agent-orchestrator for LLM conversations.

## 2. Responsibilities
### Does:
- Asterisk Integration (ARI; AMI fallback)
- Audio Streaming (RTP) and buffering
- STT integration (Google/Azure/AWS/Whisper)
- TTS integration (Google/Azure/AWS/ElevenLabs)
- Call state management (tenant-scoped)
- Transfer orchestration (queues/external)
- DTMF handling

### Does NOT:
- LLM processing (agent-orchestrator)
- Tool execution (tools-gateway/MCP)
- Tenant management (tenant-manager)
- Long-term storage (publishes events)
- Authentication logic beyond platform-auth JWT
- Business rules (pure orchestration)

## 3. Architecture Principles
- Hexagonal Architecture
- Event-Driven (platform-events)
- Multi-tenant (tenant_id everywhere)
- Observable (platform-observability)
- Resilient (timeouts, fallbacks)
- Grounded + Governed: RAG for knowledge, MCP for actions with schemas/authz/audit; fail-closed on low confidence.

## 4. Stack
- Go 1.23+
- Asterisk via ARI (primary) / AMI (fallback)
- Audio: PCM/WAV/opus
- Streaming: WebSocket bidirectional audio
- State: Redis (call state) + in-memory cache
- Events: Kafka via platform-events
- Metrics: Prometheus via platform-observability

## 5. Service Structure
(Aligned to current repo layout: adapters for asterisk/stt/tts/agent/tenant/http/redis/events; application for call/conversation/audio; domain aggregates; config; pkg utilities.)

## 6. Call Flow
### Incoming:
1) Asterisk → ARI webhook to voice-gateway
2) Resolve tenant_id from DID/trunk; create call entity
3) Fetch tenant settings (STT/TTS/LLM) from tenant-manager
4) Init conversation with agent-orchestrator; answer call
5) Loop: audio → STT → text → agent-orchestrator → LLM → text → TTS → audio → caller
6) Stop when hangup/transfer/error; publish events; cleanup state

### Outbound:
1) External trigger → voice-gateway API
2) Validate tenant → originate via ARI → on answer, same loop

## 7. State (Redis example)
```json
{
  "call_id": "uuid",
  "tenant_id": "uuid",
  "channel_id": "SIP/trunk-00000001",
  "conversation_id": "uuid",
  "state": "active|ringing|answered|transferred|ended",
  "direction": "inbound|outbound",
  "caller_number": "+5511999998888",
  "callee_number": "+55112345",
  "answered_at": "2024-01-01T10:00:00Z",
  "ended_at": null,
  "agent_id": "tech-support-agent",
  "stt_provider": "google",
  "tts_provider": "elevenlabs",
  "audio_buffer": "reference_to_redis_stream",
  "metadata": {"trunk_id": "uuid", "did_id": "uuid"}
}
```

## 8. Integration Points
- tenant-manager: GET telephony settings/trunks/DIDs
- tools-gateway / MCP: invoke governed tools (billing.get_invoice, crm.lookup_customer) with tenant-scoped tokens; enforce allow/deny by agent role/env; log status/latency/input-output hash
- agent-orchestrator: create conversation, send turns, get agent, transfer
- platform-events: call.started/answered/ended, stt.transcribed, llm.responded, tts.generated, call.transferred/escalated, error events

## 9. Error Handling
- STT: timeout → prompt; error → DTMF/transfer; retries = 3
- RAG/MCP: low-score or timeout → do not inject; clarify or transfer (fail-closed for voice). Tool error/denied → safe message; retry if idempotent; always log chunk IDs and tool status.
- LLM: timeout → fallback; error → cached/transfer; rate-limit → queue or human
- TTS: timeout → backup provider; error → prerecorded message
- Network: Asterisk reconnect (3x); STT/TTS backup; Redis fallback in-memory (short TTL)

## 10. Performance Targets
- 1000+ concurrent calls/instance; STT p95 < 500ms partial; LLM < 2s; TTS < 1s; end-to-end < 3s; audio 16kHz/16-bit PCM.

## 11. Security
- Trunk auth via Asterisk; API auth via platform-auth JWT
- TLS for audio/HTTP; mask PCI data in transcripts; GDPR opt-out; audit all call events with tenant_id

## 12. Deployment
- Horizontal scale; HA with shared Redis; 2 CPU/4GB per ~100 concurrent calls; low-latency link to Asterisk (<10ms).
