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
2) Middleware/interceptors: ✅ added RequireScopes/RequireAnyScope/RequireAllScopes for Gin plus net/http + chi middleware and gRPC unary/stream interceptors. Still needed: tenant enforcement, panic recovery, optional CORS/body-size limits; envelope helpers now covered by service contract tests.
3) Client: ✅ context-aware methods, retries/backoff + circuit breaker, TLS/mTLS, service identity headers, structured errors. Next: ensure services set service identity/TLS env and add authz helper for internal calls if needed.
4) Tenant/RLS helpers: ✅ helpers to inject tenant_id into DB contexts/queries and Kafka headers; guard rails for cross-tenant enforcement; guidance for pgvector/ClickHouse still pending. Next: adopt helpers in services (tenant-manager, billing, analytics) and verify RLS paths.
5) Observability: ✅ Prometheus counters/histograms for auth outcomes/latency across Gin/HTTP/chi/gRPC, OTEL spans tagging request/user/tenant/scope/session, and logging redaction helpers. Outbound HTTP client now propagates request IDs + trace context; still need service-side adoption and handler log scrubbing.
6) Security hardening: algorithm allow-list, token length cap, header parsing strictness, CSRF guidance for cookie-based flows, fail-closed defaults; secret load should not panic in middleware path. Header redaction expanded; add CSRF/fail-closed guidance.
7) Docs/examples: ✅ config matrix (ISSUER, AUDIENCE, JWKS_URL/PEM, REQUIRED_SCOPES, TENANT_CLAIM, CLOCK_SKEW, AUTH_GATEWAY_URL, JWT_SECRET, TLS files, tenant header) in en/pt; ✅ tenant helper usage examples; ✅ tenant/RLS guidance for Postgres/pgvector/ClickHouse. Still needed: multi-framework examples (Gin/chi/gRPC) and breaking-change guidance.
8) Governance: versioning/breaking-change notes for claims format and middleware behaviors; deprecation path for any API changes.
9) Testing/benchmarks: golden tokens (valid/expired/wrong alg/wrong aud/issuer/missing tenant/scope), JWKS rotation tests, middleware scope/role tests, client retry tests, outbound propagation assertions across services, fuzz for header parsing (✅), benchmark middleware overhead (✅).

## Ordered follow-ups (do in this sequence)
- [x] Voice-gateway: use tenant/agent clients inside StartConversation to open the conversation and pull agent config from tenant-manager before starting the loop.
- [x] Voice-gateway readiness: wire readinessHandler to check Redis/Kafka/Asterisk (and fail closed when dependencies unavailable).
- [x] Voice-gateway providers: register real STT/TTS providers instead of empty maps in main wiring.
- [x] Deployments: when Helm/K8s manifests for voice-gateway are added, include TENANT_MANAGER_TOKEN and AGENT_ORCHESTRATOR_TOKEN env vars.

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
		- [x] Gin/chi handlers (auth-gateway OAuth + login success; tenant-manager List + Create)
		- [x] gRPC surface (map error codes + metadata)
**Next priority**: Tenant helper adoption in services, outbound authz helper if needed.

### In progress (current cycle)
- Add Gin/chi/gRPC usage snippets to response envelope docs (examples section added).
- Begin tenant propagation adoption (reuse tenant_id from context/envelope; next: wire tenant helpers in billing/analytics repositories and outbound headers).
- [x] Tenant/RLS guidance for pgvector/ClickHouse and DB helper examples
- [x] Service adoption follow-up: tenant helpers + outbound EnsureTenantHeader in remaining services
- [x] Client authz helper for internal calls (service identity/scopes) if needed
- [x] Tests to reach 100%:
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

## Service adoption (MVP blockers — scope locked)
- tenant-manager
	- [x] Repositories use tenant context helpers (DB/Kafka).
	- [x] Outbound HTTP/Kafka inject EnsureTenantHeader (platform-auth transport where applicable).
	- [x] Table-driven tests for tenant guard + header injection.
- billing-service
	- [x] DB layers use tenant helpers.
	- [x] Kafka producer injects tenant header via helpers.
	- [x] Outbound HTTP uses platform-auth transport + EnsureTenantHeader.
	- [x] Integration tests for tenant guard + header injection.
- analytics-query-service
	- [x] Apply tenant helpers on ClickHouse/Postgres query path.
	- [x] Enforce EnsureTenantHeader on outbound (if any) via platform-auth transport. (N/A — no outbound clients)
	- [x] Integration test for tenant guard on queries.
- tools-gateway
	- [x] HTTP client uses platform-auth transport + EnsureTenantHeader.
	- [x] GraphQL client uses platform-auth transport + EnsureTenantHeader.
	- [x] SOAP client uses platform-auth transport + EnsureTenantHeader.
	- [x] Integration tests for each client verifying X-Tenant-Id propagation.
- auth-gateway
	- [x] Outbound HTTP (if any) uses platform-auth transport + EnsureTenantHeader.
	- [x] Test propagation of tenant header.
- voice-gateway
	- [x] Agent/tenant clients and Kafka events propagate tenant header. (Kafka N/A)
	- [x] Test conversation start enforces X-Tenant-Id. (agent/tenant client header tests)
- rag-gateway
	- [x] Verify outbound tenant header injection via platform-auth transport; add test if missing.
- Cross-service
	- [x] One integration assertion per service that outbound calls include X-Tenant-Id when tenant is in context.
		- [x] billing-service (service client tenant header integration test)
		- [x] auth-gateway (tenant client propagation integration test)
		- [x] rag-gateway (OpenAI embedding client tenant header test)
		- [x] agent-orchestrator (tools client tenant header tests)
		- [x] tools-gateway (HTTP/GraphQL/SOAP client tenant header integration tests)
		- [x] voice-gateway (tenant client header propagation integration test)
		- [x] tenant-manager (no outbound HTTP; documented N/A, add coverage if outbound appears)

## Next steps (execution order)
1) Ship service-to-service token pattern (SERVICE_AUDIENCE + client credentials) with docs and validation tests. ✅
2) Close "Service adoption" checklist above (tenant helpers + EnsureTenantHeader + tests per service).
3) Enforce response envelope helpers across Gin/chi/gRPC; keep contract tests current with new handlers and transports.
4) Logging/observability: ensure outbound transports in all services and redaction defaults; add propagation tests.
5) Docs/examples: add Gin/chi/gRPC snippets and breaking-change notes; confirm CSRF/fail-closed defaults in guidance.
