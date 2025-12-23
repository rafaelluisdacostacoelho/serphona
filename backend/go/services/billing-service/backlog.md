# Billing Service — Backlog (en-US)

## Status snapshot
- Handles billing flows (likely Stripe/webhooks, invoices). Current implementation not reviewed—needs audit.

## Open items
1) **Stripe/webhook security**: verify signature validation, idempotency keys, replay protection; tests for webhook handlers.
2) **Tenant isolation**: enforce tenant_id on all billing records; RLS in DB; scope checks on APIs.
3) **Plans/quotas**: implement plan enforcement, usage metering (tool calls, embeddings, storage), and overage handling.
4) **Invoices/payments**: ensure correct currency/tax handling; retries/backoff for provider errors; DLQ for failed webhooks.
5) **Observability**: metrics for payment success/fail, webhook latency, retries; tracing and structured logs with tenant labels.
6) **Security/compliance**: PII masking; no secrets in logs; PCI considerations (tokenize, do not store PANs).
7) **Testing**: contract tests for webhook payloads; integration with Stripe testmode; regression tests for quota enforcement.
8) **Runbooks**: webhook outage handling, DLQ replay, refund/credit flows, plan migration steps.

## Config to surface
- Stripe keys/webhook secrets; webhook endpoints; retry/backoff settings; DLQ path/topic.
- Plan definitions/limits; usage event topic; billing cycle configs.
- Metrics/tracing exporters; log level/format.

## Test coverage checklist
- [ ] Webhook sig/idempotency
- [ ] Tenant/RLS enforcement
- [ ] Quota/plan enforcement
- [ ] Retry/DLQ on provider errors
- [ ] Metrics/traces present

## Next suggested steps
- Review webhook handlers and quota logic; add sig/idempotency tests and metrics if missing.
