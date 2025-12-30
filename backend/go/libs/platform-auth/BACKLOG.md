# platform-auth — Backlog (en-US)

## Status snapshot
- Audit complete: HMAC + JWKS (cached with kid allow-list) JWT validation with Gin, net/http + chi, and gRPC (unary + stream) interceptors. Request ID propagation + OTEL spans wired across transports; client resilience and tenant helpers added; service adoption of tenant guards is pending.
- Progress: issuer/audience + leeway, alg allow-list, max token length; scopes/permissions with per-route middleware; JWKS fetch/cache + kid allow-list + rotation handling; cross-framework adapters; Prometheus auth metrics across transports; OTEL spans annotate request/user/tenant/scope (no session ID); panic recovery/body-size/CORS helpers; structured errors unified.

## Audit findings
- JWT validation: HMAC secret plus JWKS (cached with kid allow-list); issuer/audience+leeway, alg allow-list + max token size, scopes/permissions support; refresh flows and service-to-service tokens still not addressed.
- Claims: requires UUID user/tenant and role enum; now includes Scopes []string field with HasScope/HasAnyScope/HasAllScopes helpers; no service-to-service claims pattern.
- Middleware: Gin + net/http + chi + gRPC (unary/stream) with consistent error mapping, request ID propagation, OTEL spans, panic recovery, optional CORS/body-size limits. Response envelope alignment still needs verification across services.
- Client: retries/backoff + circuit breaker, TLS/mTLS, service identity headers, structured errors, context-aware calls. Need service adoption (set service name/instance, TLS files in env) and authz patterns for internal calls.
- RLS/tenant: helpers added (tenant resolution/enforcement + header injection) but not yet adopted in services/DB/Kafka flows.
- Observability: Prometheus counters/histograms across transports, OTEL spans with request/user/tenant/scope attributes, logging redaction/fuzz. Need service-side adoption (sanitize logs) and tenant header propagation verification.
- Security: header redaction expanded; still missing CSRF guidance for cookies and explicit fail-closed defaults review.
- Docs: en/pt-BR config matrix and tenant helper usage added; still want multi-framework examples and breaking-change notes.

## Action items
1) Token validation: ✅ added issuer/audience/clock-skew enforcement, alg allow-list, max token length, scopes/permissions support with RequiredScopes config and Claims.Scopes field, JWKS fetch/cache with kid allow-list and rotation handling. Still needed: refresh-token guidance and service-to-service tokens (client credentials) with separate audience.
2) Middleware/interceptors: ✅ added RequireScopes/RequireAnyScope/RequireAllScopes for Gin plus net/http + chi middleware and gRPC unary/stream interceptors. Still needed: tenant enforcement, panic recovery, optional CORS/body-size limits, and consistent error payloads/shape across transports.
3) Client: ✅ context-aware methods, retries/backoff + circuit breaker, TLS/mTLS, service identity headers, structured errors. Next: ensure services set service identity/TLS env and add authz helper for internal calls if needed.
4) Tenant/RLS helpers: ✅ helpers to inject tenant_id into DB contexts/queries and Kafka headers; guard rails for cross-tenant enforcement; guidance for pgvector/ClickHouse still pending. Next: adopt helpers in services (tenant-manager, billing, analytics) and verify RLS paths.
5) Observability: ✅ Prometheus counters/histograms for auth outcomes/latency across Gin/HTTP/chi/gRPC, OTEL spans tagging request/user/tenant/scope/session, and logging redaction helpers. Outbound HTTP client now propagates request IDs + trace context; still need service-side adoption and handler log scrubbing.
6) Security hardening: algorithm allow-list, token length cap, header parsing strictness, CSRF guidance for cookie-based flows, fail-closed defaults; secret load should not panic in middleware path. Header redaction expanded; add CSRF/fail-closed guidance.
7) Docs/examples: ✅ config matrix (ISSUER, AUDIENCE, JWKS_URL/PEM, REQUIRED_SCOPES, TENANT_CLAIM, CLOCK_SKEW, AUTH_GATEWAY_URL, JWT_SECRET, TLS files, tenant header) in en/pt; ✅ tenant helper usage examples; ✅ tenant/RLS guidance for Postgres/pgvector/ClickHouse. Still needed: multi-framework examples (Gin/chi/gRPC) and breaking-change guidance.
8) Governance: versioning/breaking-change notes for claims format and middleware behaviors; deprecation path for any API changes.
9) Testing/benchmarks: golden tokens (valid/expired/wrong alg/wrong aud/issuer/missing tenant/scope), JWKS rotation tests, middleware scope/role tests, client retry tests, outbound propagation assertions across services, fuzz for header parsing (✅), benchmark middleware overhead (✅).

## Ordered follow-ups (do in this sequence)
- [ ] Voice-gateway: use tenant/agent clients inside StartConversation to open the conversation and pull agent config from tenant-manager before starting the loop.
- [ ] Voice-gateway readiness: wire readinessHandler to check Redis/Kafka/Asterisk (and fail closed when dependencies unavailable).
- [ ] Voice-gateway providers: register real STT/TTS providers instead of empty maps in main wiring.
- [ ] Deployments: when Helm/K8s manifests for voice-gateway are added, include TENANT_MANAGER_TOKEN and AGENT_ORCHESTRATOR_TOKEN env vars.

## Open checklist
- [x] Refresh/service-to-service token guidance (audience separation, issuance pattern) + rollout plan — see docs/architecture/AUTH-GUIDANCE-en-US.md (pt-BR: docs/architecture/AUTH-GUIDANCE-pt-BR.md)
- [x] CSRF + fail-closed defaults documented — see docs/architecture/AUTH-GUIDANCE-en-US.md (pt-BR: docs/architecture/AUTH-GUIDANCE-pt-BR.md); validation pending in services
- [x] Logging hardening + outbound propagation verification (request/trace ID and tenant header) across services
	- [x] Adopt platform-auth outbound transport in services (propagates X-Request-Id/Traceparent/X-Tenant-Id) — agent-orchestrator tools client, rag-gateway openai client
	- [x] Verify no sensitive headers leak in logs (Authorization/Cookie redaction) — none found in agent-orchestrator/rag-gateway/analytics-query-service
	- [x] Tenant-manager/agent-orchestrator: confirm request ID middleware and tenant header forwarding on outbound (tenant-manager has no HTTP outbound; Kafka adds tenant header; agent-orchestrator propagates via transport + tenant header)
	- [x] Rag-gateway/analytics: ensure tenant header not hardcoded; rag-gateway now uses transport; analytics-query-service has no HTTP outbound
	- [x] Define canonical envelope (success: data+meta; error: code/message/details?/trace_id) in docs/architecture with en/pt-BR snippet — see docs/architecture/RESPONSE-ENVELOPE-CONTRACT-en-US.md (pt-BR: docs/architecture/RESPONSE-ENVELOPE-CONTRACT-pt-BR.md)
	- [x] Provide shared helpers for Gin/chi/net/http + gRPC status/metadata mapping (include trace/request IDs) — response.WriteSuccess/WriteError in libs/platform-auth/response
	- [x] Migrate handlers to the helper (auth-gateway, tenant-manager, agent-orchestrator, tools-gateway, voice-gateway, analytics-query-service)
	- [x] Add contract tests for envelope shape and trace_id propagation across transports
		- [x] analytics-query-service SearchEvents (Gin) envelope shape with trace_id/request_id + pagination
		- [x] Remaining Gin/chi handlers (auth-gateway GetOAuthURL, tenant-manager List)
		- [x] gRPC surface (map error codes + metadata)
**Next priority**: Tenant helper adoption in services, outbound authz helper if needed.

### In progress (current cycle)
- Add envelope contract tests on services (started: analytics-query-service SearchEvents; next: auth-gateway/tenant-manager + gRPC surface).
- Add Gin/chi/gRPC usage snippets to response envelope docs (examples section added).
- Begin tenant propagation adoption (reuse tenant_id from context/envelope; next: wire tenant helpers in billing/analytics repositories and outbound headers).
- [x] Tenant/RLS guidance for pgvector/ClickHouse and DB helper examples
- [ ] Service adoption follow-up: tenant helpers + outbound EnsureTenantHeader in remaining services
- [x] Client authz helper for internal calls (service identity/scopes) if needed
- [ ] Tests to reach 100%:
	- [x] Golden token matrix (expired/nbf/issuer/audience/alg/missing tenant/scopes)
	- [x] JWKS rotation and kid allow-list
	- [x] Middleware scope/role enforcement
	- [x] Tenant helpers
	- [x] Outbound propagation (request/trace ID + tenant header)
	- [x] Client retries/TLS
	- [x] Explicit coverage target surfaced (100% statements for platform-auth lib)

## Config to surface
- JWT_SECRET (fallback), JWKS_URL/JWKS_CACHE_TTL/JWKS_ALLOWED_KIDS, ISSUER, AUDIENCE, SERVICE_AUDIENCE, REQUIRED_SCOPES, TENANT_CLAIM, CLOCK_SKEW, MAX_TOKEN_BYTES, AUTH_GATEWAY_URL, TLS_CA/CERT/KEY for client, CORS settings, TRACE/REQUEST ID header names.

## Test coverage checklist
- [x] Valid token acceptance (HMAC)
- [x] Invalid/expired/nbf/issuer/audience/alg errors
- [x] Missing scope rejection with RequiredScopes config
- [x] Per-route scope enforcement with RequireScopes/RequireAnyScope middleware
- [x] JWKS validation and kid errors
- [x] Middleware context injection (trace/request IDs) across Gin/chi/gRPC (unary + stream)
- [x] Client retries/backoff and TLS/mTLS paths
- [x] Metrics/tracing emitted without leaking secrets
- [x] Benchmarks for middleware overhead

## Service adoption (MVP blockers)
- [ ] tenant-manager: apply tenant helpers in repositories and ensure outbound HTTP/Kafka include EnsureTenantHeader; add table-driven tests for header injection.
- [ ] billing-service: enforce tenant helpers in DB/Kafka layers; ensure outbound HTTP clients use platform-auth transport + EnsureTenantHeader; add tests.
- [ ] analytics-query-service: wire tenant helpers in query path (ClickHouse/Postgres) and validate tenant header on outbound (if/when added); add tests.
- [ ] tools-gateway: audit all outbound HTTP clients to use platform-auth transport + EnsureTenantHeader and add tests.
- [ ] auth-gateway: audit outbound calls (if any) to ensure platform-auth transport + EnsureTenantHeader; add tests.
- [ ] voice-gateway: propagate tenant header on agent/tenant clients and Kafka events; add tests around conversation start ensuring tenant header set.
- [ ] rag-gateway: verify tenant header injection on outbound remains correct; add tests if missing.
- [ ] Cross-service tests: add integration-level assertions that outbound calls include X-Tenant-Id when tenant is present in context.

## Next steps
- ✅ Validation hardening (Phase 1): issuer/audience/clock-skew, alg allow-list, max token length
- ✅ Scopes/permissions (Phase 2): ValidationConfig.RequiredScopes, Claims.Scopes field with helpers, RequireScopes/RequireAnyScope middleware
- ✅ JWKS support with rotation + kid allow-list for key management
- ✅ Middleware/interceptors — added net/http + chi middleware and gRPC unary/stream interceptors with scope enforcement and consistent error mapping
- ✅ Observability (metrics phase): Prometheus counters/histograms for auth outcomes/latency across Gin/HTTP/chi/gRPC
- **Next priority**: Logging hardening and client propagation — keep secrets out of logs, and ensure outbound clients forward request/trace IDs.
- Client: retries/backoff/circuit breaker + TLS/mTLS; structured errors; service identity headers; context-aware methods and request ID/trace propagation are in place.
- Tenant/RLS helpers for DB/Kafka (helpers added: context/header enforcement; service adoption pending).
- Docs: config matrix (issuer/audience/jwks/scopes/tenant/skew/max token bytes), examples multi-framework, breaking changes.
- Tests: golden tokens (valid/expired/nbf/issuer/audience/alg/kid/scope/tenant), JWKS rotation, middleware scope/role, client retries/TLS, outbound propagation, fuzz header parsing, benchmarks middleware overhead.
