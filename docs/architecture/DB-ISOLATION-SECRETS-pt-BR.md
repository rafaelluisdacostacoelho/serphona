# Segredos por serviço para bancos isolados

Use estes segredos por ambiente (dev/stage/prod) ao adotar DB dedicado por serviço.

## tenant-manager
- Nome do Secret: `tenant-manager-db`
- Chave: `url`
- Valor: `postgres://tm_user:tm_pass@tenant-manager-postgres:5432/tenant_manager?sslmode=disable`
- Consumo: `DATABASE_URL` via `valueFrom.secretKeyRef` (Helm `tenant-manager`).

## auth-gateway
- Segredos já existentes: `auth-gateway-db` com `username` e `password`; `DB_HOST/DB_PORT/DB_NAME` vêm de values.
- Para usar URL única, criar/atualizar `auth-gateway-db` com chave `url` e expor no chart antes de remover variáveis separadas.

## billing-service
- Nome do Secret: `billing-service-db`
- Chave: `url`
- Valor: `postgres://bs_user:bs_pass@billing-service-postgres:5432/billing_service?sslmode=disable`
- Consumo: `DATABASE_URL` via `valueFrom.secretKeyRef` (Helm `billing-service`).

## tools-gateway
- Nome do Secret: `tools-gateway-db`
- Chave: `url`
- Valor: `postgres://tg_user:tg_pass@tools-gateway-postgres:5432/tools_gateway?sslmode=disable`
- Consumo: `DATABASE_URL` via secret no chart do `tools-gateway` (atualizar deployment para referenciar `secretKeyRef`).

## tools-manager
- Nome do Secret: `tools-manager-db`
- Chave: `url`
- Valor: `postgres://tmgr_user:tmgr_pass@tools-manager-postgres:5432/tools_manager?sslmode=disable`
- Consumo: `DATABASE_URL` via secret (Makefile/k8s deployment já esperam `DATABASE_URL`).

## agent-orchestrator
- Nome do Secret: `agent-orchestrator-db`
- Chave: `url`
- Valor: `postgres://ao_user:ao_pass@agent-orchestrator-postgres:5432/agent_orchestrator?sslmode=disable`
- Consumo: `DATABASE_URL` via `secretKeyRef` no deployment do `agent-orchestrator`.

## analytics-processor
- Nome sugerido do Secret: `analytics-processor-db`
- Chave: `url`
- Valor: `postgres://ap_user:ap_pass@analytics-processor-postgres:5432/analytics_processor?sslmode=disable`
- Ajustar Helm/Terraform do serviço para mapear `DATABASE_URL` via secret.

## Observações
- `analytics-query-service` usa ClickHouse; manter isolamento pela instância/cluster de ClickHouse, não via este secret.
- `voice-gateway` e componentes de telefonia usam Asterisk/Redis/Kamailio; não requerem secret de Postgres.

## Como aplicar (exemplo kubectl)
```bash
auth_url="postgres://USER:PASS@HOST:5432/DB?sslmode=disable"
kubectl create secret generic tenant-manager-db \
  --from-literal=url="$auth_url" \
  -n <namespace>
```
Substitua nome/valor conforme o serviço acima. Para Terraform/Helm, declarar os secrets via manifests ou módulos próprios.
