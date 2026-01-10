# Segredos por serviço para bancos isolados

Use estes segredos por ambiente (dev/stage/prod) ao adotar DB dedicado por serviço.

## tenant-manager
- Nome do Secret: `tenant-manager-db`
- Chave: `url`
- Valor: URL completa de conexão (ex.: `postgres://tm_user:tm_pass@tenant-manager-postgres:5432/tenant_manager?sslmode=disable`)
- Consumo: `DATABASE_URL` via `valueFrom.secretKeyRef` (Helm `tenant-manager`)

## auth-gateway
- Segredos já existentes para DB: `auth-gateway-db` com `username` e `password`; `DB_HOST/DB_PORT/DB_NAME` vêm de values.
- Se preferir URL única, criar `auth-gateway-db` com chave `url` e expor no chart antes de trocar variáveis separadas.

## billing-service
- Nome do Secret: `billing-service-db`
- Chave: `url`
- Valor: URL de conexão do banco dedicado (ex.: `postgres://bs_user:bs_pass@billing-service-postgres:5432/billing_service?sslmode=disable`)
- Consumo: `DATABASE_URL` via `valueFrom.secretKeyRef` (Helm `billing-service`)

## analytics-processor
- Criar chart/manifest para consumir URL dedicada.
- Nome sugerido do Secret: `analytics-processor-db`
- Chave: `url`
- Valor: `postgres://ap_user:ap_pass@analytics-processor-postgres:5432/analytics_processor?sslmode=disable`
- Ajustar Helm/Terraform do serviço para mapear `DATABASE_URL` via secret.

## Como aplicar (exemplo kubectl)
```bash
auth_url="postgres://USER:PASS@HOST:5432/DB?sslmode=disable"
kubectl create secret generic tenant-manager-db \
  --from-literal=url="$auth_url" \
  -n <namespace>
```
Substitua nome/valor conforme o serviço acima. Para Terraform/Helm, declarar os secrets via manifests ou módulos próprios.
