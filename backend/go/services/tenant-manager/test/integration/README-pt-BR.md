# Tenant Manager — Testes de Integração

Guia rápido para rodar a suíte em `test/integration` (build tag `integration`).

## Como executar
- Windows: `backend\go\services\tenant-manager\test\integration\run-integration-tests.bat --with-compose`
- Linux/macOS: `backend/go/services/tenant-manager/test/integration/run-integration-tests.sh --with-compose`
- Adicione argumentos do `go test` após o script, ex.: `-- -run TestTenantCreateIntegration`.

## Notas
- A flag opcional `--with-compose` sobe o stack `docker-compose.tests.yml` da raiz (Kafka etc.). Variáveis específicas do serviço (ex.: `DATABASE_URL`) precisam ser definidas externamente.
- Usa a build tag `integration`; as suítes ficam em `test/integration`.
- Para rodar várias services de uma vez, use o helper compartilhado em `backend/go/services/test/integration/run-tests.*`.
