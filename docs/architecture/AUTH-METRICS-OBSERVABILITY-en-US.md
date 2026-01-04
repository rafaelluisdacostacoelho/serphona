# Auth Gateway Metrics & Alerts (auth-gateway)

Status: Draft
Owner: Platform/Auth
Last updated: 2026-01-04

## Scope
Instrumentation, dashboard, and alerting for authentication flows in auth-gateway, plus cross-service guardrails for auth failures and missing tenant context using the shared platform-auth metrics.

## Metrics
- `auth_gateway_auth_events_total{event,outcome}`: counter of auth events by outcome.
- `auth_gateway_auth_duration_seconds_bucket{event,outcome,le}`: histogram for auth flow latency.
- `auth_gateway_tenant_missing_total{path}`: counter for requests lacking `tenant_id` context.
- Cross-service (platform-auth) counters/histograms: `auth_requests_total{transport,result,service,tenant_id,component}` and `auth_request_duration_seconds_bucket{transport,result,service,component,le}` to pivot by service/component/tenant.

## Dashboard
File: `backend/go/services/auth-gateway/grafana/auth-gateway-auth.json`
Panels:
- Failures rate (5m) stat
- Events by outcome (rate)
- Events by event/outcome (rate)
- Latency p95/p99 by event (success only)
- Tenant missing by path (5m rate)
- Failure ratio per event

Datasource variable: `DS_PROMETHEUS`.

Cross-service Grafana (new JSON TBD under `grafana/cross-service-auth.json`):
- Failure rate by service/component with `sum by (service,component,result)(rate(auth_requests_total{result!="ok"}[5m]))`
- Tenant-missing rate by service/component with `sum by (service,component)(rate(auth_requests_total{result="unauthorized",tenant_id="unknown"}[5m]))`
- Success vs failure split per tenant with `sum by (tenant_id,service,result)(rate(auth_requests_total[5m]))`
- Latency p95 per service/component using `histogram_quantile(0.95, sum by (le,service,component)(rate(auth_request_duration_seconds_bucket[5m])))`
- Top tenants by auth failures using `topk(5, sum by (tenant_id,service)(rate(auth_requests_total{result!="ok"}[5m])))`

## Alerts (PrometheusRule)
File: `infra/helm/auth-gateway/templates/prometheusrule.yaml` (enabled via values).
- AuthGatewayHighFailureRate: failure ratio > 10% for 10m.
- AuthGatewayLatencyHighP95: p95 > 1s for 10m.
- AuthGatewayLatencyHighP99: p99 > 2s for 10m.
- AuthGatewayTenantMissing: any tenant-missing in 5m.
Thresholds configurable in Helm values.

Cross-service PrometheusRule (add under `infra/helm/platform-observability/templates/prometheusrule-auth.yaml`):
- AuthFailuresByServiceHigh: `sum by (service,component)(rate(auth_requests_total{result!="ok"}[5m])) > 1` for 10m; labels: service/component/tenant when available.
- AuthTenantMissingHigh: `sum by (service,component)(rate(auth_requests_total{tenant_id="unknown"}[5m])) > 0` for 5m to catch missing tenant propagation.
- AuthLatencyP95High: `histogram_quantile(0.95, sum by (le,service,component)(rate(auth_request_duration_seconds_bucket[5m]))) > 1` for 10m.

## Helm toggles
`infra/helm/auth-gateway/values*.yaml`:
- `serviceMonitor.enabled`: expose metrics to Prometheus Operator scrape.
- `prometheusRule.enabled`: install alert rules.
- Thresholds and severities under `prometheusRule.*`.

## Adoption notes
- Metrics are emitted by HTTP handlers and middleware in auth-gateway and by platform-auth middleware across services (Gin/HTTP/gRPC). Service label is injected at startup via `SetAuthMetricsService` and tenant labels flow from tokens/headers.
- Ensure Prometheus Operator is present and scraping namespaces for all services that use platform-auth.
- Import the Grafana JSON and point to the Prometheus datasource; add cross-service dashboard once available.

## Next considerations
- Mirror pattern to other services with service-specific metrics/SLOs (tenant-manager, billing, tools, voice, rag).
- Add recording rules for SLIs/SLAs if needed (e.g., availability/error budget for login).
- Wire alerts to Alertmanager routes per env (stg/prod) and on-call rotations.
