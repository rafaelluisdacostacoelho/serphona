# Tenant Manager Database Isolation

## Context
- Previously the tenant-manager shared the main Postgres database (`serphona`) using the default `public` schema.
- We want clearer blast-radius containment and simpler access control per service.

## Decision
- Provide a dedicated Postgres instance (container) and database for `tenant-manager` in local/docker-compose.
- Connection string now targets `postgres://tm_user:tm_pass@tenant-manager-postgres:5432/tenant_manager?sslmode=disable`.
- Other services remain on the shared Postgres for now; they can be migrated later following the same pattern.

## Rationale
- Stronger isolation for data, backups, and credentials.
- Easier to rotate credentials and revoke access per service.
- Avoids schema/enum/extension collisions across services.
- Small overhead locally is acceptable; production should also favor per-service DBs when feasible.

## Consequences
- Additional Postgres container/volume in docker-compose.
- Migrations for tenant-manager run against its own DB; cross-service SQL joins are not expected (integration via APIs/events).
- Backups/restores and tuning can be scoped to this service.

## Migration Steps (local)
1) Pull latest `docker-compose.yml` with the new `tenant-manager-postgres` service and updated `DATABASE_URL`.
2) `docker compose up -d tenant-manager-postgres` (or `docker compose up -d` for all).
3) Run tenant-manager migrations (Makefile target `migrate-up` or service startup if `AutoMigrate` is enabled).

## Next Steps
- Decide if auth-gateway, billing, etc., should also get dedicated DBs (repeat the pattern: container, credentials, `DATABASE_URL`).
- Align prod/staging manifests (Helm/Terraform) to mirror this isolation.
