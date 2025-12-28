# Orientações de Tenant e RLS (Postgres + ClickHouse)

## Objetivos
- Manter toda requisição/evento/consulta escopada por `tenant_id`.
- Isolar dados na camada de banco (RLS no Postgres, filtragem/particionamento no ClickHouse).
- Facilitar adoção de helpers em handlers, repositórios e chamadas outbound.

## Helpers do platform-auth
- Entrada: use `middleware.GetTenantIDFromContext(c)` para ler `tenant_id` das claims; se presente, aplique `middleware.EnsureTenantHeader(req.Header, tenantID)` e `middleware.WithTenantID(ctx, tenantID)` para propagar ao downstream.
- Enforcement: chame `middleware.EnforceTenant(ctx, tenantIDDoPayload)` antes de gravar/ler dados para garantir que o payload bate com o tenant autenticado.
- Saída: use `EnsureTenantHeader` ao montar headers HTTP/Kafka para enviar `X-Tenant-Id` a outros serviços.

## Postgres (inclui pgvector)
- Tabelas devem ter `tenant_id UUID NOT NULL` e índices de apoio (ex.: `CREATE INDEX ON table (tenant_id, created_at)` e índice vetorial com `WHERE tenant_id = ...` no pgvector).
- Habilite RLS por tabela:
  ```sql
  ALTER TABLE my_table ENABLE ROW LEVEL SECURITY;
  CREATE POLICY my_table_tenant_isolation ON my_table
    USING (tenant_id = current_setting('app.current_tenant')::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant')::uuid);
  ```
- No middleware de DB, faça `SET app.current_tenant = :tenant_id` a partir de `middleware.TenantIDFromContext(ctx)`; falhe fechando se faltar.
- Consultas (GORM/sqlx): sempre inclua tenant explicitamente (`WHERE tenant_id = $1`). Não confie em tenant vindo do caller; use o do contexto.

## ClickHouse
- Inclua `tenant_id` em MergeTree/ReplicatedMergeTree; particione por tenant ou por (tenant_id, toYYYYMM(timestamp)) conforme volume; chave primária deve iniciar com `tenant_id`.
- Toda query precisa filtrar tenant: `WHERE tenant_id = {tenant:String}`. Crie helper para injetar `tenant_id` do contexto em params; rejeite se ausente.
- Em views/aggregações, mantenha `tenant_id` nas tabelas alvo para preservar isolamento.

## Kafka / eventos
- Ao produzir eventos, coloque `tenant_id` no payload e em headers (`X-Tenant-Id`). Use `EnsureTenantHeader` para headers. Consumidores devem validar que header e payload coincidem antes de processar.

## Checklist de testes
- Unit/integration: falhar requisições sem `tenant_id`; rejeitar mismatch entre JWT e payload; confirmar que políticas RLS bloqueiam leituras/escritas cross-tenant.
- Consultas: garantir que todos os repositórios incluem filtro de tenant; adicionar testes no ClickHouse validando presença do predicado `tenant_id`.
- Outbound: afirmar que `X-Tenant-Id` está presente em produtores HTTP/Kafka.

## Passos de migração
1) Adicionar `tenant_id` em todas as tabelas e backfill dos registros existentes.
2) Adicionar políticas de RLS e `SET app.current_tenant` no middleware de DB.
3) Ligar helpers nos handlers: extrair tenant, aplicar `EnsureTenantHeader`, anexar com `WithTenantID`, impor payload com `EnforceTenant`.
4) Atualizar repositórios para exigir tenant nas assinaturas e incluí-lo em toda query.
5) Adicionar testes (contrato HTTP + repositório) e rodá-los com `docker-compose.tests.yml` quando precisar de ClickHouse/Postgres.
