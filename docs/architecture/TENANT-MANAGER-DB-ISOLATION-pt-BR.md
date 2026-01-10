# Isolamento de Banco para o Tenant Manager

## Contexto
- Antes, o tenant-manager usava o Postgres principal (`serphona`) no schema `public`.
- Buscamos reduzir o raio de impacto e simplificar controle de acesso por serviço.

## Decisão
- Criar uma instância/dataset Postgres dedicada para o `tenant-manager` no docker-compose.
- A conexão agora aponta para `postgres://tm_user:tm_pass@tenant-manager-postgres:5432/tenant_manager?sslmode=disable`.
- Os demais serviços permanecem no Postgres compartilhado por enquanto; podem migrar depois seguindo o mesmo padrão.

## Racional
- Isolamento mais forte de dados, backups e credenciais.
- Rotação e revogação de acesso por serviço ficam mais simples.
- Evita colisão de schemas/enums/extensões entre serviços.
- Overhead local é pequeno; em produção também é desejável isolar por serviço sempre que viável.

## Consequências
- Novo container/volume Postgres no docker-compose.
- Migrations do tenant-manager rodam no seu DB; joins SQL entre serviços não são esperados (integração via APIs/eventos).
- Backups/restores e tuning ficam escopados ao serviço.

## Passos de Migração (local)
1) Atualizar o `docker-compose.yml` com o serviço `tenant-manager-postgres` e a nova `DATABASE_URL`.
2) Subir `docker compose up -d tenant-manager-postgres` (ou `docker compose up -d`).
3) Rodar migrations do tenant-manager (Makefile `migrate-up` ou auto-migrate no start se habilitado).

## Próximos Passos
- Avaliar se auth-gateway, billing etc. também devem ganhar DB dedicado (repetir container, credenciais, `DATABASE_URL`).
- Alinhar manifests de prod/stage (Helm/Terraform) para refletir o mesmo isolamento.
