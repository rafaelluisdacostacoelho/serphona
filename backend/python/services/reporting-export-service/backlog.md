# Reporting Export Service — Backlog (en-US)

## Status snapshot (code now)
- FastAPI stub only: export creation returns a random job_id and `pending`; status endpoint always returns `completed` with a hardcoded download URL; download streams a fixed CSV string. No ClickHouse queries, no file generation, no S3/SFTP delivery, no persistence/queue.
- Delivery endpoint accepts any payload and returns `pending`; background tasks are commented out; no Celery/Redis usage despite `.env.example`.
- No auth, no tenant isolation beyond a free-form `tenant_id` field; no validation of date ranges/format combinations; no size/row limits enforced.
- Settings use ad-hoc `os.getenv` in a BaseModel without required flags; many configs in `.env.example` (JWT, Redis, S3, rate limits, metrics) are unused. No metrics/tracing, rate limiting, or audit logs.
- Tests only cover happy-path placeholders (health, stub export, stub delivery); no failure paths or integration fakes.

## Critical gaps to address
1) **Real export pipeline**: Implement query layer (ClickHouse/PG) with tenant-scoped filters, chunked fetch, streaming writes (CSV/Parquet/JSON/Excel), temp storage, and integrity checks. Persist jobs (DB/Redis) with statuses and expirations.
2) **Auth/tenant enforcement**: Require auth (JWT/service token), validate tenant_id against claims, enforce RLS/where clauses, and block cross-tenant access; sanitize inputs.
3) **Delivery & storage**: Wire S3/MinIO (or SFTP/email) delivery with signed URLs, encryption (KMS/SSE), checksum, and expiry; clean temp files; configurable retention.
4) **Scheduling/quotas**: Add async workers (Celery/Redis) or background queue with retries/backoff; per-tenant rate limits and per-hour/day export quotas; idempotency on repeated requests.
5) **Validation & limits**: Strong schema for report_type, date ranges, filters, allowed formats; row/byte/time limits; prevent unbounded queries; pagination for list endpoints.
6) **Observability & billing**: Metrics for rows/bytes, duration, failures, queue time, delivery success; tracing spans; structured logs with tenant/job IDs; emit usage/billing events.
7) **Security/PII**: Column allow/deny lists per report; masking/anonymization; size caps; TLS for DB/S3; secrets from env/secret manager; request/response logging hygiene.
8) **Testing & runbooks**: Add unit/contract tests with ClickHouse/S3 fakes, quota tests, retry/DLQ paths, and performance smoke. Document runbooks: replay failed jobs, clean temp, rotate keys, handle storage outages.

## Configuration to surface/validate
- DB/ClickHouse DSN and TLS; S3/SFTP endpoints/creds/buckets; export temp dir; max rows/bytes/timeouts; rate limits/quotas; JWT issuer/audience/alg; feature flags (PDF/Excel/email/scheduling).
- Metrics/tracing endpoints; retry/backoff settings; DLQ location; retention/TTL for jobs/files.

## Test coverage targets
- Auth + tenant scoping on all endpoints; parameter validation and size/row limits.
- Export pipeline happy/failure paths: query errors, chunked write, storage upload, checksum verification, and retry/backoff with DLQ.
- Delivery variants (S3/SFTP/email) with signing/expiry; cleanup of temp files.
- Metrics/tracing emitted; quotas/rate limits enforced.

## Next steps
- Design job model + persistence, implement authenticated tenant-scoped exports with real data fetch/storage and delivery, add metrics/tracing/rate limits, and expand tests with fakes for DB/S3/queue.
