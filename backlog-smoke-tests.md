# Backlog de Smoke Tests (local kind)

## Concluídos
- Tools Gateway (Kafka off): health, RAG REST 202, docs PT/EN
- Auth Gateway: health, docs PT/EN
- Tenant Manager: health, /api/v1/tenants, docs PT/EN
- Billing Service: health, metrics, /api/v1/usage (docs PT/EN)
- Agent Orchestrator: health, metrics, /api/v1/agents (docs PT/EN)

## Próximos
- Tools Gateway (Kafka on): reativar publish com SCRAM (user1) e validar evento
- Analytics Query Service: health, query simples em endpoint de consultas
- Voice Gateway/Telephony smoke básico (se aplicável)

## Notas
- Usar issuer `serphona`, audience `serphona-services`, claim `tenantId` camelCase, chave HS256 `dev-jwt-secret` para tokens de desenvolvimento.
- Port-forward sempre em pod específico para evitar quedas.
- Desabilitar serviceMonitor/prometheusRule/grafanaDashboard em kind, salvo necessidade.
