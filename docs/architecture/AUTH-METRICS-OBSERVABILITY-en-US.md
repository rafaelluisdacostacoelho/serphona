# Auth Gateway Metrics & Alerts (auth-gateway)

Status: Draft
Owner: Platform/Auth
Last updated: 2026-01-04

## Scope
Instrumentation, dashboard, and alerting for authentication flows in auth-gateway.

## Metrics
- `auth_gateway_auth_events_total{event,outcome}`: counter of auth events by outcome.
- `auth_gateway_auth_duration_seconds_bucket{event,outcome,le}`: histogram for auth flow latency.
- `auth_gateway_tenant_missing_total{path}`: counter for requests lacking `tenant_id` context.

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

## Alerts (PrometheusRule)
File: `infra/helm/auth-gateway/templates/prometheusrule.yaml` (enabled via values).
- AuthGatewayHighFailureRate: failure ratio > 10% for 10m.
- AuthGatewayLatencyHighP95: p95 > 1s for 10m.
- AuthGatewayLatencyHighP99: p99 > 2s for 10m.
- AuthGatewayTenantMissing: any tenant-missing in 5m.
Thresholds configurable in Helm values.

## Helm toggles
`infra/helm/auth-gateway/values*.yaml`:
- `serviceMonitor.enabled`: expose metrics to Prometheus Operator scrape.
- `prometheusRule.enabled`: install alert rules.
- Thresholds and severities under `prometheusRule.*`.

## Adoption notes
- Metrics are emitted by HTTP handlers and middleware in auth-gateway.
- Ensure Prometheus Operator is present and scraping namespace of auth-gateway.
- Import the Grafana JSON and point to the Prometheus datasource.

## Next considerations
- Mirror pattern to other services with service-specific metrics/SLOs (tenant-manager, billing, tools, voice, rag).
- Add recording rules for SLIs/SLAs if needed (e.g., availability/error budget for login).
- Wire alerts to Alertmanager routes per env (stg/prod) and on-call rotations.
