# platform-rag (scaffold)

Purpose: shared RAG components (metadata models, filters, schemas, clients) for services that ingest, index, and retrieve tenant-scoped knowledge.

Status: scaffold only — no code yet. Safe to keep in repo; does not affect builds/tests.

Suggested contents (next steps):
- `metadata/`: structs for document/chunk metadata (tenant_id, namespace, language, channel, valid_from/valid_to, etag, acl/tags).
- `filters/`: helpers to enforce mandatory tenant/channel/language filters.
- `retrieval/`: interfaces for vector store clients (pgvector/ClickHouse), re-rankers, and cache decorators.
- `obs/`: hooks for tracing/metrics/logs (platform-observability).
- `test/`: contract tests/mocks for vector and rerank clients.
