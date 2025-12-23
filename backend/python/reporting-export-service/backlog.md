# Reporting Export Service — Backlog (en-US)

## Status snapshot
- Exports reporting data (likely from ClickHouse/PG) to files/BI destinations. Code not reviewed—needs audit.

## Open items
1) **Auth & tenant isolation**: require tenant_id on requests; enforce filters/RLS on queries; prevent cross-tenant exports.
2) **Export formats/targets**: define supported formats (CSV/Parquet) and sinks (S3/email); validate params; size limits.
3) **Scheduling**: if scheduled exports exist, ensure per-tenant quotas, retries, and idempotency for runs.
4) **Performance**: chunked exports, streaming responses; backpressure; temp file management; encryption at rest (S3/KMS).
5) **Observability**: metrics for export latency, rows/bytes, failures; tracing; structured logs with tenant labels.
6) **Billing**: usage events per export (rows/bytes) for cost attribution.
7) **Resilience**: retries/backoff for DB/S3; circuit breakers; DLQ for failed exports.
8) **Security/PII**: masking for sensitive columns; configurable column allowlist/denylist.
9) **Testing**: contract tests for params, integration with DB/S3 fakes, performance tests for large datasets.
10) **Runbooks**: retry failed exports, clean temp storage, rotate credentials, handle quota overruns.

## Config to surface
- Auth scopes/claims; DB DSN; S3 endpoint/creds; KMS config; export size limits.
- Metrics/tracing exporters; log level/format; retry/backoff; DLQ path/topic.

## Test coverage checklist
- [ ] Tenant/RLS enforcement
- [ ] Param validation and size limits
- [ ] Retry/DLQ on export failures
- [ ] Metrics/traces present
- [ ] Billing/usage events emitted

## Next suggested steps
- Review code/config to fill concrete details; add validation and observability if missing; create fakes for DB/S3 to cover integration tests.
