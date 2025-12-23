# Backlog RAG & MCP (en-US)

## Goals
- Enable multi-tenant RAG with secure ingestion, indexing, and retrieval.
- Expose tools via MCP to decouple agent orchestration from integrations.
- Keep observability, governance, and billing aligned with the current model (platform-auth, platform-observability, Tools Gateway).

## Principles
- Multi-tenant by design (tenant_id in all artifacts, RLS/ACL per namespace).
- Progressive compatibility: MCP as an optional façade while keeping the existing HTTP path.
- Security and compliance first: data isolation, quotas, auditing.
- Incremental testing and rollout by tenant/feature group.

## Initial Assumptions
- Vector store: pgvector (RLS) to start; revisit ClickHouse HNSW if latency/cost require.
- MinIO/S3 available for blobs and metadata.
- Kafka enabled for ingestion and reindex events.
- platform-auth provides identity with tenant_id and scopes for MCP and Tools Gateway.

### Near-term execution (Dec 22, 2025)
1) Implement platform-rag lib with metadata structs (tenant_id, namespace, document_id, version, etag, tags, acl, ttl), filters/validators, and wire rag-gateway to use them.
2) Align rag-gateway ingestion: validate new fields and emit `rag.ingestion.requested` via platform-events (flagged by RAG_EVENTS_ENABLED) with tests/docs.
3) Define minimal chunking strategy (size/overlap/cleanup/lang) and add stub hook for the pipeline.
4) Scaffold Python indexing worker (backend/python/services/rag-processor-service): consume rag.ingestion.requested, fetch blob (S3/MinIO stub), chunk, embed, upsert pgvector with idempotency on document_id+etag; add DLQ/retry + metrics placeholders.
5) Observability/billing hooks: spans/metrics on ingest/query (tenant_id/namespace/top_k) and placeholders for billing counters.

## Epics and Items

### Execution Order (services & phases)
- Phase 1: Auth/Tenant + Agent Router/Guardrails + RAG Retrieval API (tenant filters, citations) + MCP skeleton with authz/logging.
- Phase 2: Domain MCPs (billing, ticketing, CRM) wrapping existing services; telephony-control MCP after voice flows stabilize.
- Phase 3: RAG ingestion/indexing for first domain (20–100 curated docs, tenant filters, validity, re-rank/threshold) and click-through path to LLM with citations.
- Phase 4: Expand MCP tool catalog (3–5 critical tools first), enforce policies/rate limits per tenant, and wire tool calls into agent orchestrator.
- Phase 5: Python analytics processor once Kafka events carry tenant_id and schemas; reporting/export after ClickHouse is populated.

### 1) RAG Foundation
- [x] Decide vector store (pgvector to start) with latency/cost benchmarks and per-tenant limits.
- [x] Define metadata schema: {tenant_id, namespace, document_id, version/etag, tags, acl, ttl}.
- [x] Standardize chunking strategy (size, overlap, text normalization, language handling).
- [ ] Select embedding provider (OpenAI/Azure/OSS) with fallback and quotas.
- [ ] Define encryption at rest (KMS) and retention/TTL policies per namespace.

### 2) Ingestion
- [x] Create Kafka contract "ingestion.requested" with tenant_id, source, document_id, etag
- [x] Implement S3/MinIO connector (pull) with versioning and dedup by etag.
- [ ] REST/GraphQL/gRPC connector for dataset ingestion (pagination, backoff, per-tenant filters).
- [ ] SAP connector (OData/REST) with per-tenant credentials and sensitive field masking.
- [ ] Register all ingestions in Tools Gateway as ingestion tools (for audit/billing).

### 3) Indexing Pipeline
- [x] Python worker scaffold (now in backend/python/services/rag-processor-service) to: download blob, chunk, clean, embed, upsert into index.
- [ ] Support idempotent reprocessing by document_id+etag and skip on etag match.
- [ ] Publish metrics (latency per phase, avg chunk size, error rate) to platform-observability.
- [ ] Optional embedding cache in Redis to reduce cost for reindex.
- [ ] DLQ/retry semantics wired (currently placeholder only).

### 4) Retrieval
- [ ] Retrieval service with filters by tenant_id, namespace, acl, score threshold, per-call limits.
- [ ] Optional re-ranker (BM25+rerank) and snippet deduplication.
- [ ] Format context with citations (source, score, snippet) and safe content (masking).
- [ ] Hot query cache (Redis) with short TTL and invalidation by document_id.
- [ ] Observability: end-to-end tracing and audit logs (query, score, top-k) for billing.

### 5) Agent Orchestrator/Tools Integration
- [ ] Extend `prompts.yaml` spec with `rag_sources`, `rag_query` (params: namespace, top_k, filters, rerank).
- [ ] Expose `rag_query` as a tool via Tools Gateway with schema validation and billing per returned tokens.
- [ ] System prompt templates enforcing citations and context limits per channel (voice/text).
- [ ] Guardrail hook to check source ACLs before building context.
- [ ] Contract tests (golden prompts) for behavioral regression.

### 6) MCP
- [ ] Implement thin MCP server with resource discovery from Tools Gateway catalog.
- [ ] MCP auth middleware using platform-auth (token with tenant_id and scoped resources).
- [ ] Map provider types: REST, GraphQL, gRPC, S3, `rag_query`.
- [ ] MCP observability: tracing, per-resource metrics, logs to reconcile with billing.
- [ ] Client shim in Agent Orchestrator to resolve tools via MCP with HTTP fallback.
- [ ] Define tool catalog v1 (lookup_customer, create_ticket, send_whatsapp, transfer_call, billing.get_open_invoices, billing.generate_second_copy, rag_query) with schemas, error codes, idempotency keys where needed.

### 7) Security and Governance
- [ ] Quota and rate-limit policies per tenant/namespace for ingestion and retrieval.
- [ ] PII masking (email, SSN/card) in ingestion and context responses.
- [ ] Access control per namespace (owner, readers, service accounts) and MCP scopes.
- [ ] Secrets/credentials review: Vault/K8s Secrets, rotation, audit trail.
- [ ] Incident playbook for data leakage and MCP access revocation.

### 8) Observability and Billing
- [ ] Dashboards: ingestion (lag, throughput, error), indexing (phase latency), retrieval (p95/p99, cache hit-rate).
- [ ] Alerts: high latency, embedding provider errors, growing DLQ, quota exceeded.
- [ ] Billing: cost per tool call (ingestion/retrieval), embedding tokens, vector storage.
- [ ] Audit logging: who queried which source, filter params, returned scores.

### 9) Quality and Testing
- [ ] Unit and contract suites for ingestion, indexing, retrieval (offline).
- [ ] Performance tests (latency/throughput) with per-tenant loads and varying corpus sizes.
- [ ] Resilience tests (retry, backpressure, circuit breaker, DLQ).
- [ ] Security tests (ACL bypass, filter injection, context leakage).

### 10) Rollout and Operations
- [ ] Feature flags per tenant (ingestion, retrieval, MCP independently).
- [ ] Migration plan: pilot tenants, monitor, expand.
- [ ] Runbooks: ingestion failure, reindex, MCP key rotation, namespace cleanup.
- [ ] Documentation for support and customers (source onboarding, limits, usage examples).

## Deliverables
- Updated `prompts.yaml` specification for RAG and MCP.
- Functional MCP server with discovery and multi-tenant auth.
- Ingestion and indexing pipeline running with at least one source (S3/MinIO) and an end-to-end retrieval example.
- Basic dashboards and alerts published.

## Risks and Mitigations
- Embedding cost: cache, batch, provider fallback.
- Retrieval latency: tunable top-k, cache, optional rerank.
- Cross-tenant leakage: enforce tenant_id on all paths + security tests.
- Operational debt: runbooks and feature flags for quick rollback.
