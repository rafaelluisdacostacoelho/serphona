# RAG Gateway — Backlog (en-US)

## Status snapshot
- Receives RAG ingestion requests; expected to validate metadata and emit `rag.ingestion.requested` to Kafka via platform-events.
- Uses platform-auth for tenant_id/scopes; metadata structs/validators from platform-rag should be enforced.

## Open items
1) **Contract enforcement**: validate metadata fields (tenant_id, namespace, document_id, version/etag, tags, acl, ttl) and reject bad payloads; update tests.
2) **Event emission**: ensure `rag.ingestion.requested` is published with RLS-safe fields; feature flag (RAG_EVENTS_ENABLED) honored; add retries/backoff and DLQ for Kafka publish failures.
3) **Connectors**: HTTP ingest path done; add pagination/backoff for REST/GraphQL sources if applicable; document supported sources.
4) **Observability**: add tracing/metrics (ingest latency, validation errors, Kafka publish success/fail) with tenant/namespace labels where safe.
5) **Billing/Audit**: emit audit logs and usage events per ingestion; include size/chunk counts if known.
6) **Security**: ensure ACL/namespace checks; input size limits; deny disallowed URI schemes early.
7) **Testing**: contract tests for validation and event payload; integration test with Kafka fake; regression tests for feature flag behavior.
8) **Runbooks**: handling Kafka outages (DLQ/retry), invalid payload surges, and flag toggles.

## Config to surface
- Auth issuer/audience/scopes; required claims for tenant_id.
- Kafka bootstrap/topic; RAG_EVENTS_ENABLED; retries/backoff; DLQ topic/path.
- Allowed URI schemes; max payload size; timeouts.
- Observability exporters; log level/format.

## Test coverage checklist
- [ ] Metadata validation pass/fail
- [ ] Feature flag toggling event emission
- [ ] Kafka publish retry/DLQ on failure
- [ ] Metrics/traces exposed
- [ ] Allowed/disallowed scheme enforcement

## Next suggested steps
- Audit handlers to align with platform-rag structs and mark completed items; add Kafka publish retry/DLQ tests.
