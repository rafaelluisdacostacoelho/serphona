# platform-rag — Backlog (en-US)

## Status snapshot
- Provides chunking helper, document metadata normalize/validate, vector store interfaces, and a pgvector store implementation.
- Missing: filters, retrieval/rerank abstractions, observability, table schema management, provider-specific embedding client.
- README still says “scaffold only”; needs alignment with current code.

## Gaps and tasks (ordered)
1) **Models & metadata**
   - Extend `model.Chunk`/`Query` to cover tags/acl/ttl/version/source/uri and strict types (avoid bare map[string]string for metadata/filters where possible).
   - Enforce min_score/top_k defaults in query helpers; add validation functions with tests.
   - Decide on canonical metadata shape for storage/query (jsonb keys, reserved prefixes) and document in README.
2) **Chunking**
   - Expose sane defaults (align with ingestion worker) and add examples; consider language-aware boundaries.
   - Add benchmarks and guardrails for extremely long inputs.
3) **pgvector store**
   - Add schema management helper (DDL) or documented table contract; indexes on (tenant_id, namespace, chunk_id) + vector ivfflat parameters.
   - Enforce RLS guidance and table name validation tests; handle context timeouts.
   - Add unit/integration tests (docker-compose.tests) for upsert/query, dimension mismatch, filters jsonb, and score/min_score enforcement.
4) **Interfaces & adapters**
   - Implement filter helpers (tenant/namespace required, tag/acl filters) and retrieval decorators (reranker, cache) per README suggestions.
   - Provide embedding client implementations (OpenAI/Azure) or stubs with retries/backoff and dimension checks.
5) **Observability & errors**
   - Add metrics/tracing hooks around store operations (latency, errors) and structured errors; avoid leaking PII.
6) **Docs**
   - Update README to reflect existing code; add usage snippets for chunking, metadata validation, pgvector store wiring, and schema contract.
7) **Tooling**
   - Lint/format checks, go version sanity (go 1.24 placeholder), and module replace guidance for services.

## Config to surface (doc)
- pgvector DSN/table name/dimension/lists, RLS requirements, retry/backoff settings.
- Embedding provider envs (API key/model/base URL) once clients added.

## Test coverage checklist
- [ ] Metadata normalize/validate edge cases (negative ttl, missing tenant/ns/doc) — partial present.
- [ ] Chunking boundaries/overlap — exists, expand for edge cases/benchmarks.
- [ ] pgvector upsert/query happy path and dimension mismatch.
- [ ] Filters (tags/acl/jsonb) and min_score enforcement.
- [ ] Observability hooks emitting metrics/traces.

## Next steps
- Add tests/docs for pgvector store and extend models/filters; update README and version constraints before integrating deeper with rag-gateway.
