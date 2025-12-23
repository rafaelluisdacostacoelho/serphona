# Tenant Manager — Backlog (en-US)

## Status snapshot
- Manages tenant lifecycle, namespaces, ACLs. Not fully reviewed—needs audit.

## Open items
1) **Tenant lifecycle**: create/update/delete flows; namespace creation; enforce uniqueness; emit events for other services.
2) **ACLs/roles**: manage owners/readers/service accounts; enforce on APIs; integrate with platform-auth scopes.
3) **RLS & quotas**: ensure DB schemas have RLS per tenant; store quotas (ingestion, retrieval, telephony) and enforce.
4) **Observability**: metrics for tenant creation failures, latency; tracing with tenant_id; structured audit logs.
5) **Billing hooks**: emit usage/events on tenant creation/plan changes; sync with billing-service.
6) **Resilience**: retries/backoff on DB/external calls; idempotency keys for create/update.
7) **Testing**: contract tests for APIs, RLS leakage tests, quota enforcement tests.
8) **Runbooks**: tenant deletion/cleanup, quota override, namespace recovery.

## Config to surface
- Auth scopes; DB DSN/pool; RLS policies; quota defaults; event topics.
- Metrics/tracing exporters; log level/format.

## Test coverage checklist
- [ ] ACL/role enforcement
- [ ] RLS/tenant isolation
- [ ] Quota enforcement
- [ ] Event emission
- [ ] Metrics/traces present

## Next suggested steps
- Audit handlers and DB schema; add RLS/ACL tests and quota enforcement coverage.
