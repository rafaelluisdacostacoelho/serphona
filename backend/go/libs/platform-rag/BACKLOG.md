# platform-rag — Backlog (en-US)

## Status snapshot
- Provides chunking helper (defaults/benchmarks/MaxSegments), metadata normalize/validate, typed models (chunks/filters), vector store interface with pgvector store/schema helper, embedding adapters/decorators (retry/cache/OpenAI), reranker decorator, Prometheus observer, and updated README (en/pt).
- Missing: language-aware chunking boundaries, richer filters (language/channel), structured errors pass-through.

## Gaps and tasks (ordered)
1) **Models & metadata**
   - ✅ Extend `model.Chunk`/`Query` with typed metadata/filters (tags/acl/ttl/version/source/uri) and defaults/validation.
   - ✅ Document canonical metadata shape in README (jsonb keys, tags/acl arrays, ttl_seconds, source/version/uri).
   - ✅ Add more validation tests for edge cases (e.g., empty embeddings, bad URIs, ttl upper bounds).
2) **Chunking**
   - ✅ Expose sane defaults/examples (align with ingestion worker) and benchmarks; MaxSegments guard for long inputs.
   - ✅ Add language-aware boundaries option (LanguageCode) for punctuation hints.
3) **pgvector store**
   - ✅ Add schema helper with table/index DDL; table name validation.
   - ✅ Add context timeouts and observability hooks.
   - ✅ Add integration tests (pgvector env) for upsert/query, filters jsonb, min_score.
4) **Interfaces & adapters**
   - ✅ Add cache decorator and embedding client (OpenAI) with retries/backoff and dimension checks.
   - ✅ Add reranker decorator for vector.Store.
5) **Observability & errors**
   - ✅ Add trace/latency hooks around store operations; metrics wiring (Prometheus) and structured StoreError wrapper.
6) **Docs**
   - ✅ README updated with usage, chunking, metadata and schema contract (en-US/pt-BR).
7) **Tooling**
   - ✅ Docker-compose target for pgvector integration test (make rag-pgvector-int).
   - Lint/format checks, go version sanity (go 1.24 placeholder), module replace guidance for services.

## Config to surface (doc)
- pgvector DSN/table name/dimension/lists, RLS requirements, retry/backoff settings.
- Embedding provider envs (API key/model/base URL) once clients added.

## Test coverage checklist
- [x] Metadata normalize/validate edge cases (negative ttl, missing tenant/ns/doc).
- [x] Chunking boundaries/overlap — defaults + benchmarks + MaxSegments guard.
- [x] pgvector upsert/query happy path and filters/min_score (integration test behind `-tags=integration`).
- [x] Observability hooks emitting metrics/traces (Prometheus observer wired).

## Next steps
- Add language-aware chunking boundaries (optional) and richer filters (language/channel) if needed by services.
- Consider structured errors surface in vector store/reranker for observability correlation.
