# Auth Gateway — Backlog (en-US)

## Status snapshot
- Central entry for authentication/authorization; issues/validates tokens with tenant_id. Code not reviewed here—needs audit.

## Open items
1) **Token issuance/validation**: ensure correct issuer/audience, signing keys rotation, clock skew handling; include tenant_id and scopes.
2) **RLS alignment**: propagate tenant_id to downstream services; enforce namespace/ACL checks where relevant.
3) **Protocols**: confirm OAuth2/OIDC flows supported; PKCE for public clients; device/refresh token policies; revocation/blacklist if required.
4) **Security**: rate limits/quotas per client/tenant; brute-force protections; mTLS if used; input validation for redirect URIs.
5) **Observability**: metrics for login success/fail, latency, token issuance; tracing and structured logs.
6) **Billing/Audit**: audit logs for auth events; optional usage events for MAU/billing if applicable.
7) **Resilience**: retry/backoff for upstream IdP (if any); circuit breakers; fail-closed defaults.
8) **Testing**: contract tests for token claims; negative tests for invalid scopes/expired tokens; integration with downstream middleware.
9) **Runbooks**: key rotation steps, incident playbook for auth outage, revocation handling.

## Config to surface
- Issuer, audience, JWKS/keys, token TTLs, refresh settings.
- Required scopes/claims; tenant_id claim mapping; allowed redirect URIs.
- Rate limits; logging/tracing exporters; metrics port.

## Test coverage checklist
- [ ] Token validation happy/negative
- [ ] Tenant_id/scopes propagation
- [ ] Rate limits/brute-force protections
- [ ] Metrics/traces present
- [ ] Key rotation path tested

## Next suggested steps
- Audit current handlers/configs to populate specifics and mark completed items; add auth contract tests and metrics if missing.
