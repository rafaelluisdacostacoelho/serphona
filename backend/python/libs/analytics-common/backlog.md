# analytics-common (Python) — Backlog (en-US)

## Status snapshot
- Shared analytics/NLP utilities for Python services. Not re-reviewed in this pass.

## Open items
1) **Data models**: validate Pydantic models for events/records; ensure tenant_id/namespace fields are required and normalized.
2) **Text processing utilities**: language detection, tokenization, normalization—document defaults and ensure deterministic behavior; add tests per language.
3) **Storage adapters**: if present (ClickHouse/PG), ensure parameterized queries, RLS alignment, and retries/backoff.
4) **Embedding/helpers**: if any shared embedding functions exist, align with provider configs and dimensions; add validation and tests.
5) **Observability**: logging patterns, metrics/tracing hooks for shared utilities; avoid leaking PII.
6) **Docs**: usage examples per service (analytics-processor, reporting-export), config table, and constraints (e.g., max text length).
7) **Testing**: unit tests for parsers/normalizers, schema validation, and edge cases (empty text, malformed events).

## Config to surface
- Language/default locale settings, text length limits, optional provider keys if used by helpers.

## Test coverage checklist
- [ ] Model validation (tenant_id required)
- [ ] Text normalization/detection tests
- [ ] Storage adapter retry/backoff (if present)
- [ ] Embedding/helper validation (if present)
- [ ] Logging/metric hooks sanity

## Next steps
- Audit the package to replace placeholders with concrete utilities and mark done/todo; add missing tests and docs.
