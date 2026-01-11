# Tools Manager (MVP scaffold)

Governance/control-plane for tools catalog and policies. This initial scaffold focuses on a runnable service with health/ready/metrics and a stub API.

## Quick start

```bash
cd backend/go/services/tools-manager
cp .env.example .env
make migrate-up   # requires DATABASE_URL
make run
# health
curl http://localhost:8087/health
# list tools (stub)
curl http://localhost:8087/api/v1/tools
```

## Endpoints
- `/health` – liveness
- `/ready` – readiness (stub: always ready)
- `/metrics` – Prometheus metrics
- `/debug/pprof` – pprof
- `/api/v1/tools` – authenticated + tenant guard; returns empty list (placeholder for catalog)

## Config (env)
- `SERVER_HOST` (default `0.0.0.0`)
- `SERVER_PORT` (default `8087`)
- `ENV` (`development`|`production`)
- `METRICS_PATH` (default `/metrics`)
- `HEALTH_PATH` (default `/health`)
- `READY_PATH` (default `/ready`)
- `DATABASE_URL` (Postgres DSN for catalog)
- `JWT_SECRET` (HS256 shared secret; optional when using JWKS)
- `JWT_ALLOWED_ALGS` (default `HS256`)
- `JWT_ISSUER`, `JWT_AUDIENCE`, `JWT_SERVICE_AUDIENCE`
- `JWT_CLOCK_SKEW` (default `30s`), `JWT_MAX_TOKEN_BYTES` (default `4096`)
- `JWKS_URL`, `JWKS_CACHE_TTL` (default `5m`), `JWKS_ALLOWED_KIDS`
- `JWT_REQUIRED_SCOPES` (comma-separated)
- `TENANT_CLAIM` (default `tenant_id`)

## Migrations
- Local: `make migrate-up` / `make migrate-down` (uses `DATABASE_URL`).
- Kubernetes: init container `run-migrations` executes SQL from ConfigMap `tools-manager-migrations` before the app starts.

## Next steps (from backlog)
- Persist catalog (tools, versions, tenant overrides) with Postgres + RLS and migrations.
- Implement validation (JSON Schema, allowlists) and policy/quotas.
- Integrate auth (platform-auth), response envelope contract, and tenant guard.
- Add secrets via Vault/KMS, audit logging, and change events for Tools Gateway / platform-mcp.
- Add tests (migrations, policy matrix, schema limits, consumer contracts).
