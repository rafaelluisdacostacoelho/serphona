# Billing MFE (Frontend) — Backlog (en-US)

## Status snapshot
- Billing micro-frontend (Vite). Not audited in this pass.

## Open items
1) **Billing flows**: invoices list, payment status, plan upgrades/downgrades; handle errors/retries; reflect quotas/usage.
2) **Auth/tenant**: require valid session with tenant_id; scope checks for billing features.
3) **Security**: avoid exposing secrets; sanitize data; guard against XSS; CSP via nginx config.
4) **Observability**: UI telemetry for billing actions; error tracking; feature flags.
5) **UX/a11y**: responsive tables, accessibility, clear error/success states; loading/empty states.
6) **Testing**: component tests for views/forms; integration/e2e for billing journeys; contract tests with billing-service mocks.
7) **Docs**: env vars, build/deploy steps, integration instructions for host shell/console.

## Config to surface
- Billing API base, observability DSNs, feature flags, auth endpoints.

## Test coverage checklist
- [ ] Auth/tenant checks
- [ ] Plan/usage display
- [ ] Payment/invoice flows
- [ ] Telemetry emits
- [ ] Host integration contract

## Next steps
- Audit current implementation; fill concrete tasks and mark done/todo; add missing tests and docs.
