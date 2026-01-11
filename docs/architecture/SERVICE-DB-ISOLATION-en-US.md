# Per-service database isolation playbook

Purpose: track how each service should migrate from the shared Postgres to its own database + credentials. Pair this with [Tenant RLS guidance](TENANT-RLS-GUIDANCE-en-US.md) and the secrets list in [DB isolation for secrets](DB-ISOLATION-SECRETS-en-US.md).

## Standard pattern
- Provision a dedicated Postgres instance/database per service (dev/stage/prod). Name DBs with the service (e.g., `serphona_auth`, `serphona_billing`).
- Create a service-specific role with least privileges (own schema, no superuser) and enforce TLS where available.
- Add connection via `DATABASE_URL` only; remove host/port/name fragments once charts consume the secret URL.
- Ensure migrations/seed jobs target the dedicated DB; avoid cross-service SQL joins (integrate via APIs/events).
- Apply RLS and `tenant_id` filters for multi-tenant data stores.

## Service matrix
- **tenant-manager** — Already isolated (`tenant_manager` DB, `tenant-manager-db.url`). Keep RLS + onboarding/outbox tables scoped per tenant.
- **auth-gateway** — Target DB `serphona_auth`; secret `auth-gateway-db.url`; update Helm/compose to point `DATABASE_URL` to the dedicated instance and drop split env vars after rollout.
- **billing-service** — Target DB `serphona_billing`; secret `billing-service-db.url`; ensure migrations run against the dedicated container/instance and apply tenant filters when billing by tenant.
- **tools-gateway** — Target DB `tools_gateway`; secret `tools-gateway-db.url`; wire `DATABASE_URL` via secret and add migrations + tenant/RLS enforcement in the gateway tables.
- **tools-manager** — Target DB `tools_manager`; secret `tools-manager-db.url`; keep RLS roles (`application`, `service_account`) per the service runbook and point deployments to the dedicated DB.
- **agent-orchestrator** — Target DB `agent_orchestrator`; secret `agent-orchestrator-db.url`; enforce `tenant_id` on agent/workflow tables; migrations must run against the isolated DB.
- **analytics-processor** — Target DB `analytics_processor`; secret `analytics-processor-db.url`; use for control/state tables only. Analytics facts stay in ClickHouse.

## Out of scope (not Postgres)
- `analytics-query-service` uses ClickHouse; isolate via its own ClickHouse cluster/schema.
- `voice-gateway`/telephony rely on Asterisk/Kamailio/RTPEngine/Redis; no Postgres isolation needed.

## Rollout checklist per service
1) Create Postgres instance + DB + user with strong password.
2) Create secret (`<service>-db.url`) with the DSN; update Helm/Terraform/compose to read it as `DATABASE_URL`.
3) Run migrations against the dedicated DB; validate RLS/tenant filters where applicable.
4) Rotate old shared credentials, monitor errors, and back up the new DB.
