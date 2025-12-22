# RAG + MCP Architecture for Serphona (en-US)

## Why RAG + MCP
- RAG supplies current, tenant-scoped knowledge (playbooks, policies, catalogs) to cut hallucinations and anchor answers with citations.
- MCP standardizes actions (tooling) with typed schemas, authz, and audit trails, reducing brittle prompt-based tool calls.
- Voice constraints: low latency, predictable behavior, and explainability (what was said, why, based on which sources, and which actions ran).

## Where They Fit
- **Realtime Voice Layer**: SIP ingress (Kamailio) → rtpengine → Asterisk/WebRTC → STT/TTS → Turn Manager.
- **Agent Runtime**: Agent orchestrator/router, session state, guardrails/policies.
- **Knowledge Layer (RAG)**: Ingestion → chunking → embeddings → vector store with tenant filters → re-rank → generation with citations; hot KB cache.
- **Action Layer (MCP)**: MCP servers per domain exposing typed tools; authz/policies per tenant/agent/environment; logging + idempotency.
- **Data & Observability**: Kafka for events; ClickHouse for analytics; S3/MinIO for artifacts (transcripts, tool logs, retrieval snapshots); platform-observability for traces/metrics/logs.

## RAG Architecture
- **Ingestion/Governance**: Treat KB as product. Normalize, chunk, embed, index; metadata: tenant_id, namespace/domain, language, channel, valid_from/valid_to, version/etag, acl/tags.
- **Storage**: Vector DB with strict tenant filters (or per-tenant index/namespace); doc blobs in S3/MinIO; ingestion events in Kafka.
- **Retrieval**: Filters by tenant_id + channel + domain + language + validity; low topK, re-ranker, similarity threshold; citations with snippet IDs; hot cache with invalidation by document_id.
- **Separation**: Call/session memory in transactional store (Redis/DB) vs corporate KB in vector store.

## MCP Architecture
- **Pattern**: MCP servers per domain (recommended) — mcp-crm, mcp-ticketing, mcp-billing, mcp-telephony-control, mcp-analytics.
- **Contracts**: Typed inputs/outputs, error codes, idempotency keys where needed; examples: crm.lookup_customer, ticket.create, billing.get_open_invoices, telephony.transfer_call, messaging.send_whatsapp, analytics.log_event, rag_query.
- **Policy Engine**: Rules by tenant_id, agent_role, environment (dev/hml/prd), tool allow/deny; rate limits and quotas per tenant.
- **Audit/Obs**: Log tool calls (tenant_id, agent_id, call_id, tool, input/output hash, duration, status); trace every call; align with billing.

## Turn Flow (voice example)
Audio → STT → Router (intent/risk/lang) → RAG retrieval (filtered) → LLM generation with citations → MCP tool calls (if needed) → validation/logging/state update → TTS reply. Example: “2nd invoice copy” → router tags billing → RAG fetches policy + ERP steps → plan asks ID, calls billing.get_invoice + messaging.send_whatsapp → logs context + tool call.

## Migration Plan (phased)
1) **RAG MVP (one domain, one tenant pilot)**: 20–100 curated docs; filters by tenant_id/valid_to; re-rank + threshold; citations.
2) **MCP for 3–5 critical tools**: Wrap existing tooling; add authz + logging; tools like lookup_customer, create_ticket, send_whatsapp, transfer_call, billing.get_invoice.
3) **Router + guardrails**: Intent/risk classifier; low-confidence → clarification or human handoff; high-risk → stricter policies.
4) **Hard multi-tenancy**: Separate indices or enforced filters; per-tenant credentials/scopes; rate limits and quotas.
5) **Expand domains**: More MCP servers (billing/crm/ticketing/telephony), more KB namespaces; add reporting/export once ClickHouse is populated.

## Implementation Order (services)
- **First**: Agent orchestrator/router + guardrails; RAG service (retrieval API) with tenant filters; MCP skeleton with authz/logging.
- **Then**: Domain MCPs (billing, ticketing, CRM), integrating existing services; telephony-control MCP after voice flows stable.
- **Python**: Analytics processor once Kafka events include tenant_id and basic schemas; reporting/export after ClickHouse is populated.

## Operational Notes
- Fail closed in production: if context is low-quality or tool fails, ask clarification or transfer; do not guess.
- Track IDs of retrieved chunks per turn for explainability and audits.
- Keep prompts slim; rely on retrieval for “manuals”; enforce citations in system prompts.
- Version and deprecate documents with valid_to; prefer incremental ingestion and reindex via Kafka.
