# Tenant and RLS Guidance (Postgres + ClickHouse)

## Goals
- Keep every request, event, and query scoped by `tenant_id`.
- Enforce tenant isolation at the data layer (Postgres RLS, ClickHouse filtering/partitioning).
- Make helpers easy to adopt in handlers, repositories, and outbound calls.

## Platform-auth helpers to use
- Inbound: `middleware.GetTenantIDFromContext(c)` to read `tenant_id` from JWT claims; if present, call `middleware.EnsureTenantHeader(req.Header, tenantID)` and `middleware.WithTenantID(ctx, tenantID)` so downstream code receives the tenant.
- Enforcement: `middleware.EnforceTenant(ctx, tenantIDFromPayload)` before mutating or reading data to ensure the payload matches the authenticated tenant.
- Outbound: wrap HTTP/Kafka headers with `EnsureTenantHeader` to propagate `X-Tenant-Id` to other services.

## Postgres (including pgvector)
- Tables must include `tenant_id UUID NOT NULL` and supporting indexes (e.g., `CREATE INDEX ON table (tenant_id, created_at)` and vector index with `WHERE tenant_id = ...` if using pgvector).
- Enable RLS per table:
  ```sql
  ALTER TABLE my_table ENABLE ROW LEVEL SECURITY;
  CREATE POLICY my_table_tenant_isolation ON my_table
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);
  ```
- In the DB connection middleware, set `SET app.current_tenant = :tenant_id` from `middleware.TenantIDFromContext(ctx)`; fail closed if missing.
- When writing queries (GORM/sqlx): always bind tenant explicitly (e.g., `WHERE tenant_id = $1`). Do not trust caller-provided tenant; use the one from context.

## ClickHouse
- Include `tenant_id` column in MergeTree/ReplicatedMergeTree; partition by tenant or by (tenant_id, toYYYYMM(timestamp)) depending on volume; primary key should start with `tenant_id`.
- Every query must filter by tenant: `WHERE tenant_id = {tenant:String}`. Add a helper to inject `tenant_id` from context into query params; reject if absent.
- For materialized views/aggregations, keep `tenant_id` in the target tables to preserve isolation.

## Kafka / events
- When producing events, add `tenant_id` to the message payload and to headers (e.g., `X-Tenant-Id`). Use `EnsureTenantHeader` for headers. Consumers must validate that the header matches the payload before processing.

## Testing checklist
- Unit/integration: fail requests without `tenant_id`; reject mismatched tenant between JWT and payload; confirm RLS policies block cross-tenant reads/writes.
- Query-level: ensure all repository methods include tenant filters; add tests for ClickHouse queries verifying `tenant_id` predicate is present.
- Outbound: assert `X-Tenant-Id` is set on HTTP/Kafka producers.

## Migration steps
1) Add `tenant_id` to all tables and backfill existing rows.
2) Add RLS policies and `SET app.current_tenant` in DB middleware.
3) Wire helpers in handlers: extract tenant, call `EnsureTenantHeader`, attach with `WithTenantID`, enforce payload tenant with `EnforceTenant`.
4) Update repositories to require tenant in signatures and include it in every query.
5) Add tests (HTTP contract + repository) and run them with `docker-compose.tests.yml` when ClickHouse/Postgres are needed.
