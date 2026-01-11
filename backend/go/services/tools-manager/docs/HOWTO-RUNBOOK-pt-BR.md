# Tools Manager — How-to e Runbook (pt-BR)

Público: times de plataforma/operação.

## Pré-requisitos
- Postgres acessível com `DATABASE_URL` configurado; roles `application` e `service_account` criadas (RLS).
- Auth: emissor/audience do JWT configurados; ajuste `TENANT_CLAIM` se necessário.
- Segredos: `SECRETS_ENCRYPTION_KEY` (16/24/32 bytes); credenciais do Vault quando `SECRETS_USE_VAULT=true`.
- Observabilidade: Prometheus coletando `/metrics`; tracing via OTEL se exportador configurado.
- Eventos: configure `EVENTS_WEBHOOK_URL` quando precisar de fanout por webhook.

## Tarefas de ciclo de vida
- Adicionar ferramenta (draft):
  1) POST `/api/v1/tools` com token do tenant. Envie `name`, `display_name`, `version`, `status`=`draft`, schemas, allowlist e limites.
  2) Verifique com GET `/api/v1/tools`.
- Publicar / atualizar ferramenta:
  1) Novo POST com `version` e `status`=`published` (idempotente por name+version).
  2) Caches são invalidados via webhook ou polling em `/api/v1/catalog/changes` (ETag em `/catalog/resolved`).
- Configurar política:
  1) POST `/api/v1/policy/rules` com `effect` (`allow|deny`), `tool_id`/`agent_id`/`env` opcionais, `scopes`, `roles`, `weight`.
  2) GET `/api/v1/policy/rules` para checar ordem (ordenado por weight; deny vence empate).
- Configurar quotas:
  1) POST `/api/v1/quota/rules` com `limit_per_minute`/`limit_per_day`, `tool_id`/`agent_id`/`env` opcionais, `weight`.
  2) GET `/api/v1/quota/rules` para confirmar.
- Habilitar tenant:
  - Ferramentas são habilitadas por tenant na criação; overrides via `tenant_tools` (admin API futura). Tokens devem carregar `tenant_id` correto.
- Rotacionar segredos:
  1) POST `/api/v1/secrets` com `{id,value}` usando token do tenant. Valor é cifrado (AES-GCM) e armazenado no backend (Vault se habilitado).
  2) GET `/api/v1/secrets/:id` retorna valor descriptografado; TTL de cache definido por `SECRETS_CACHE_TTL`.
- Rollback:
  - Catálogo: crie nova versão com payload anterior e reaponte `tenant_tools.tool_version_id` (API atual reatribui na criação). 
  - Config/infra: `make migrate-down` local ou use imagem/tag anterior no K8s e reaplique migrations conforme necessário.

## Checklist de versionamento / breaking change
- Sempre incremente `version`; não mutar `tool_versions` existentes.
- Mantenha `input_schema`/`output_schema` compatíveis ou publique nova versão; documente campos deprecated.
- Atualize allowlist/timeouts/limites por versão; evite ampliar allowlists silenciosamente.
- Ajuste políticas/quotas se forem específicas de tool/version.
- Notifique consumidores: webhook inclui `diff_hash`; registre notas para platform-mcp/Tools Gateway.

## Notas de integração (MCP / Tools Gateway)
- Use `/api/v1/catalog/resolved` com `If-None-Match` para sincronização eficiente; `/catalog/changes` como fallback.
- Vista MCP `/api/v1/catalog/mcp` expõe forma mínima para platform-mcp/agent-orchestrator.
- Eventos de mudança emitem `tool.updated` com `tenant_id`, `tool_id`, `version_id`, `diff_hash`.

## Dashboards e alertas
- Métricas chave: `tools_manager_tools_writes_total`, `tools_manager_catalog_reads_total`, `tools_manager_secret_fetches_total{cache_hit}`, `tools_manager_policy_decisions_total`, `tools_manager_quota_decisions_total`, `tools_manager_errors_total`.
- Alertas: aumento de erros por rota/tenant; pico de denies de quota; aumento de cache_miss em segredos; falhas de webhook (monitore logs do notifier se disponível).

## Runbook de oncall
- 5xx/alerta: verifique `/health` e `/ready`; cheque logs com campos `audit_*` e `diff_hash`.
- Problemas de cache: compare ETag e `/catalog/changes`; reenvie webhook se necessário.
- Negativas de quota/política: GET rules, revise weights/scopes; ajuste ou desabilite regra.
- Problemas de segredos: confirme conexão Vault (se habilitado) e tamanho de `SECRETS_ENCRYPTION_KEY`; rotacione via POST e reteste.
- DB/RLS: confirme role `application` e migrations aplicadas; testes de repositório podem ser reexecutados com `go test ./internal/repository` (sobe Postgres de teste localmente).
