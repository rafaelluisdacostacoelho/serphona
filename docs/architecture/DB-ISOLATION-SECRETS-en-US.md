# Secrets per service for isolated databases

Use these secrets per environment (dev/stage/prod) when giving each service a dedicated Postgres database.

## tenant-manager
- Secret name: `tenant-manager-db`
- Key: `url`
- Value: `postgres://tm_user:tm_pass@tenant-manager-postgres:5432/tenant_manager?sslmode=disable`
- Consumption: `DATABASE_URL` via `valueFrom.secretKeyRef` (Helm `tenant-manager`).

## auth-gateway
- Existing secrets: `auth-gateway-db` with `username` and `password`; `DB_HOST/DB_PORT/DB_NAME` come from chart values.
- To switch to a single URL, create/update `auth-gateway-db` with key `url` and wire it in the chart before removing the separate vars.

## billing-service
- Secret name: `billing-service-db`
- Key: `url`
- Value: `postgres://bs_user:bs_pass@billing-service-postgres:5432/billing_service?sslmode=disable`
- Consumption: `DATABASE_URL` via `valueFrom.secretKeyRef` (Helm `billing-service`).

## tools-gateway
- Secret name: `tools-gateway-db`
- Key: `url`
- Value: `postgres://tg_user:tg_pass@tools-gateway-postgres:5432/tools_gateway?sslmode=disable`
- Consumption: `DATABASE_URL` via secret in the `tools-gateway` chart (update deployment to use `secretKeyRef`).

## tools-manager
- Secret name: `tools-manager-db`
- Key: `url`
- Value: `postgres://tmgr_user:tmgr_pass@tools-manager-postgres:5432/tools_manager?sslmode=disable`
- Consumption: `DATABASE_URL` via secret (Makefile/k8s deployment already expect `DATABASE_URL`).

## agent-orchestrator
- Secret name: `agent-orchestrator-db`
- Key: `url`
- Value: `postgres://ao_user:ao_pass@agent-orchestrator-postgres:5432/agent_orchestrator?sslmode=disable`
- Consumption: `DATABASE_URL` via `secretKeyRef` in the `agent-orchestrator` deployment.

## analytics-processor
- Secret name: `analytics-processor-db`
- Key: `url`
- Value: `postgres://ap_user:ap_pass@analytics-processor-postgres:5432/analytics_processor?sslmode=disable`
- Wire Helm/Terraform for the service to map `DATABASE_URL` from that secret.

## Notes
- `analytics-query-service` uses ClickHouse; keep its isolation by provisioning a dedicated ClickHouse instance/cluster, not via this secret list.
- `voice-gateway` and telephony components rely on Asterisk/Redis/Kamailio; they do not need Postgres secrets.

## How to apply (kubectl example)
```bash
auth_url="postgres://USER:PASS@HOST:5432/DB?sslmode=disable"
kubectl create secret generic tenant-manager-db \
  --from-literal=url="$auth_url" \
  -n <namespace>
```
Replace the secret name/value for each service above. For Terraform/Helm, declare the secrets through manifests or modules.
