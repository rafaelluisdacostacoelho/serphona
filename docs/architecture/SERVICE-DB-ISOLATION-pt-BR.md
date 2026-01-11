# Playbook de isolamento de banco por serviço

Propósito: acompanhar a migração de cada serviço do Postgres compartilhado para seu próprio banco e credenciais. Use junto com o [Guia de RLS por tenant](TENANT-RLS-GUIDANCE-pt-BR.md) e a lista de segredos em [Isolamento de DB para segredos](DB-ISOLATION-SECRETS-pt-BR.md).

## Padrão recomendável
- Provisione instância/banco Postgres dedicado por serviço (dev/stage/prod). Nomeie com o serviço (ex.: `serphona_auth`, `serphona_billing`).
- Crie role específica com privilégios mínimos (schema próprio, sem superuser) e force TLS quando disponível.
- Use somente `DATABASE_URL`; elimine variáveis fragmentadas de host/porta/nome após o chart consumir a URL via secret.
- Garanta que migrations/seeds rodem no banco dedicado; evite joins SQL entre serviços (integre via APIs/eventos).
- Aplique RLS e filtros de `tenant_id` para dados multi-tenant.

## Matriz por serviço
- **tenant-manager** — Já isolado (`tenant_manager`, secret `tenant-manager-db.url`). Manter RLS e tabelas de onboarding/outbox por tenant.
- **auth-gateway** — Banco alvo `serphona_auth`; secret `auth-gateway-db.url`; aponte `DATABASE_URL` para a instância dedicada e remova variáveis fragmentadas depois do rollout.
- **billing-service** — Banco alvo `serphona_billing`; secret `billing-service-db.url`; execute migrations no container/instância dedicada e aplique filtros de tenant quando houver cobrança multi-tenant.
- **tools-gateway** — Banco alvo `tools_gateway`; secret `tools-gateway-db.url`; referencie `DATABASE_URL` via secret e adicione migrations + RLS/filtros de tenant nas tabelas do gateway.
- **tools-manager** — Banco alvo `tools_manager`; secret `tools-manager-db.url`; mantenha as roles de RLS (`application`, `service_account`) conforme o runbook e aponte o deployment para o DB dedicado.
- **agent-orchestrator** — Banco alvo `agent_orchestrator`; secret `agent-orchestrator-db.url`; aplique `tenant_id` nas tabelas de agentes/fluxos e rode migrations no DB isolado.
- **analytics-processor** — Banco alvo `analytics_processor`; secret `analytics-processor-db.url`; use para tabelas de controle/estado. Fatos analíticos continuam no ClickHouse.

## Fora de escopo (não Postgres)
- `analytics-query-service` usa ClickHouse; isole via cluster/schema próprio de ClickHouse.
- `voice-gateway`/telefonia dependem de Asterisk/Kamailio/RTPEngine/Redis; não precisam de secret de Postgres.

## Checklist de rollout por serviço
1) Criar instância + banco + usuário Postgres com senha forte.
2) Criar secret (`<service>-db.url`) com o DSN; atualizar Helm/Terraform/compose para ler em `DATABASE_URL`.
3) Rodar migrations no DB dedicado; validar RLS/filtros de tenant quando aplicável.
4) Revogar credenciais do banco compartilhado, monitorar erros e salvar backup do novo DB.
