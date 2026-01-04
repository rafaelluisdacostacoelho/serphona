# Billing Service — Backlog (en-US)

## Status snapshot
- Audit complete: service is mostly stubbed. Gin handlers return placeholders; no Stripe SDK usage, no webhook signature verification, no idempotency, no auth/tenant enforcement. DB connection is created but no migrations/models wired; Redis/Kafka/Wallet/Subscriptions repos unused. CORS/rate limits absent, observability configs unused, and security/compliance requirements unmet.

## Audit findings
- API surface: All endpoints (customers, subscriptions, invoices, usage, checkout/portal) return static JSON with no validation, auth, or tenant context. No OpenAPI/contract. Health is static. No request size/timeouts beyond server defaults.
- Stripe/webhooks: `/webhooks/stripe` reads body only; no signature verification, idempotency, retry handling, or event routing. Stripe keys/secrets not used anywhere; no SDK client.
- Billing logic: No implementation of customer creation, subscription lifecycle, plan catalog, invoices, payments, portal/checkout sessions, or usage metering. Wallet/credits domain types exist but are not used; Kafka/Redis configs unused.
- Data layer: GORM connection uses DATABASE_URL with pool caps; no migrations or schema setup; no RLS/tenant guards; no repositories wired except a Postgres customer repository that is unused. No transaction boundaries, no idempotency keys, no soft-delete patterns defined.
- Tenant/auth: No auth middleware; tenant_id not required or validated; no linkage to tenant-manager; no scoping on routes or data.
- Observability: No structured logging, tracing, or Prometheus metrics; observability config unused; no audit log for financial events.
- Security/compliance: Default Gin CORS (permissive); no TLS/mTLS knobs; no PCI/PII controls; secrets loaded from env without validation; webhook endpoint lacks replay protection.
- Resilience/reliability: No retry/backoff around Stripe or DB; no DLQ for failed webhooks; no background jobs for cleanup (sessions, invoices). Server uses basic timeouts but lacks graceful dependency checks on readiness.

## Action items
1) Implement secure config: validate required Stripe keys/webhook secret, database/redis TLS options, allowed origins, max body size, and per-route auth/tenant requirements; fail fast on missing secrets.
2) Auth/tenant enforcement: require platform-auth middleware with tenant claims; enforce tenant scoping on all routes and DB queries; integrate with tenant-manager for customer mapping and plan eligibility.
3) Stripe integration: add Stripe client with API version, signature verification, and idempotency keys; implement portal/checkout session creation, customer sync, subscription lifecycle, invoice fetch; persist Stripe IDs.
4) Webhook processing: verify signature, parse events, handle subscription/invoice/payment events with idempotency store, retries/backoff, and DLQ/Kafka emission for failures; add structured audit logs.
5) Billing/quotas: define plans/price catalog, enforce quotas (usage events from Kafka), overage billing, and proration; align wallet/credits with usage (debit/credit with locking); add currency/tax handling.
6) Data layer: add migrations (customers, subscriptions, invoices, wallet, transactions, idempotency), RLS/tenant columns, transactional writes; hash secrets (refresh/idempotency) where applicable.
7) Observability/compliance: add tracing (HTTP/Stripe/DB), Prometheus metrics (webhook latency, errors, retries, quota denials), structured logging with tenant and event IDs; PII masking, no secrets in logs.
8) Security: tighten CORS, require HTTPS, optional mTLS for webhooks, request size limits, replay protection; store no PANs—use tokens only; secure secret management (no .env in prod).
9) Resilience/runbooks: readiness checks for DB/Stripe; background cleaners for stale idempotency keys; rate limits on public endpoints; runbooks for webhook backlog, replay, refunds/credits, plan migrations.
10) Testing: Stripe testmode contract tests (webhooks, portal/checkout), idempotency/retry tests, tenant isolation tests, quota enforcement/overage, wallet debit/credit invariants, metrics/tracing assertions.

## Config to surface
- Server host/port, read/write timeouts, max body size, allowed origins, TLS/mTLS.
- Database URL/SSL, pool caps, migration toggle; Redis/Kafka URLs if used for idempotency/DLQ.
- Stripe secret/publishable keys, webhook secret, API version, idempotency TTL, retry/backoff limits.
- JWT issuer/audience/keys for auth middleware; tenant-manager base URL.
- Plan catalog source, quota limits, billing cycle, tax/currency settings.
- Observability exporters (Prometheus/Otel), log level/format, audit sink.

## Test coverage checklist
- [ ] Webhook signature/idempotency/retry flows with Stripe test payloads
- [ ] Tenant scoping and auth on all routes/data
- [ ] Subscription/plan lifecycle, proration, and quota enforcement
- [ ] Wallet debit/credit invariants and locking
- [ ] Metrics/tracing/audit emission and config validation

## Next suggested steps
- Wire validated config and auth/tenant middleware, then implement Stripe client + signature verification with idempotency store; add real customer/subscription flows and migrations, followed by webhook processors, quotas, observability, and test suites.

## Referências
- Este serviço depende do [BACKLOG-AUTH.md](../../../BACKLOG-AUTH.md) para alinhamento com autenticação e autorização multi-tenant.
