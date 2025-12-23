# Auth MFE (Frontend) — Backlog (en-US)

## Status snapshot
- Auth micro-frontend (Vite). Not audited in this pass.

## Open items
1) **Flows**: login/logout, passwordless/OTP if applicable; error states and retries; session refresh.
2) **Tenant handling**: capture tenant selection/tenant_id; propagate in tokens; enforce redirect allowlist.
3) **Security**: XSS/CSRF protections, safe storage of tokens (httpOnly if possible), CSP headers via nginx config.
4) **Observability**: telemetry for auth success/fail, latency; error tracking; feature flags for experiments.
5) **UX/a11y**: forms validation, keyboard nav, focus, locales; responsive design.
6) **Testing**: component/unit tests for forms, integration/e2e for full auth flows; contract tests for redirect/PKCE if used.
7) **Docs**: env vars, build/deploy steps, integration instructions for host shell/console.

## Config to surface
- Auth endpoints, client IDs/secrets (if public, ensure PKCE), redirect URIs, observability DSNs, feature flags.

## Test coverage checklist
- [ ] Login/logout happy/negative
- [ ] Token/tenant propagation
- [ ] CSRF/XSS mitigations
- [ ] Telemetry emits
- [ ] Host integration contract

## Next steps
- Review current flows/config to populate concrete tasks and mark done/todo; add missing tests and docs.
