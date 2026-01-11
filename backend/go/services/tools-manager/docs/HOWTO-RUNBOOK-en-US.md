# Tools Manager — How-to and Runbook (en-US)

Audience: operators and platform engineers.

## Prereqs
- Postgres reachable; `DATABASE_URL` set (RLS requires `application` and `service_account` roles).
- Auth: JWT issuer/audience configured; set `TENANT_CLAIM` if customized.
- Secrets: `SECRETS_ENCRYPTION_KEY` (16/24/32 bytes); Vault creds when `SECRETS_USE_VAULT=true`.
- Observability: Prometheus scraping `/metrics`; tracing exporter configured via OTEL envs if present.
- Events: `EVENTS_WEBHOOK_URL` set when webhook fanout is needed.

## Lifecycle tasks
- Add tool (create draft):
  1) POST `/api/v1/tools` with tenant-scoped token. Provide `name`, `display_name`, `version`, `status`=`draft`, schemas, allowlist, and limits. 
  2) Verify creation via GET `/api/v1/tools`.
- Publish / update tool:
  1) Re-POST with new `version` and `status`=`published` (idempotent per name+version). 
  2) Downstream caches refresh via webhook or `/api/v1/catalog/changes` polling (ETag on `/catalog/resolved`).
- Configure policy:
  1) POST `/api/v1/policy/rules` with `effect` (`allow|deny`), optional `tool_id`/`agent_id`/`env`, `scopes`, `roles`, `weight`.
  2) Check ordering via GET `/api/v1/policy/rules` (sorted by weight, deny wins ties).
- Configure quotas:
  1) POST `/api/v1/quota/rules` with `limit_per_minute`/`limit_per_day`, optional `tool_id`/`agent_id`/`env`, `weight`.
  2) GET `/api/v1/quota/rules` to confirm.
- Enable tenant access:
  - Tools are auto-enabled per tenant on create; manage overrides via `tenant_tools` (future admin API). Ensure tokens carry correct `tenant_id`.
- Rotate secrets:
  1) POST `/api/v1/secrets` with `{id,value}` under tenant token. Value is AES-GCM encrypted and backend-stored (Vault if enabled).
  2) GET `/api/v1/secrets/:id` fetches decrypted value; cache TTL controlled by `SECRETS_CACHE_TTL`.
- Rollback:
  - Catalog: create new version with previous payload and set tenant binding via `tenant_tools.tool_version_id` (current API rebinds on create). 
  - Config/infra: run `make migrate-down` (local) or rerun prior image/tag in K8s and reapply migrations as needed.

## Versioning / breaking-change checklist
- Bump `version` field; never mutate existing `tool_versions` payloads.
- Keep `input_schema`/`output_schema` backward-compatible or publish a new version; document deprecated fields.
- Update allowlist/timeouts/limits per version metadata; avoid widening allowlists silently.
- Ensure policies/quotas updated to cover new tool/version if scoped.
- Notify consumers: webhook payloads include `diff_hash`; add release notes for platform-mcp/Tools Gateway.

## Integration notes (MCP / Tools Gateway)
- Use `/api/v1/catalog/resolved` with `If-None-Match` for cache-friendly sync; `catalog/changes` as polling fallback.
- MCP view `/api/v1/catalog/mcp` exposes minimal shape for platform-mcp/agent-orchestrator.
- Change events emit `tool.updated` with `tenant_id`, `tool_id`, `version_id`, `diff_hash`.

## Dashboards and alerts
- Key metrics: `tools_manager_tools_writes_total`, `tools_manager_catalog_reads_total`, `tools_manager_secret_fetches_total{cache_hit}`, `tools_manager_policy_decisions_total`, `tools_manager_quota_decisions_total`, `tools_manager_errors_total`.
- Alerts: high error rate per path/tenant; quota denies spike; secret fetch cache_miss surge; webhook delivery failures (add scrape on notifier logs if available).

## Oncall runbook
- 5xx/alert: check `/health` and `/ready`; inspect logs for `audit_*` fields and `diff_hash` references.
- Cache invalidation issues: compare ETag responses and recent `/catalog/changes`; replay webhook if needed.
- Quota/policy denials: GET rules, confirm weights/scopes; adjust or disable offending rule.
- Secret issues: verify Vault connectivity (when enabled) and `SECRETS_ENCRYPTION_KEY` length; rotate secret via POST and retry.
- DB/RLS issues: ensure `application` role exists and migrations applied; repository tests can be rerun with `go test ./internal/repository` (auto-spins Postgres container locally).
