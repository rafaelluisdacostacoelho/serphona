# Bibliotecas Go — Testes de Integração/E2E

Helpers unificados para rodar suítes em `test/integration` (build tag `integration`) e `test/e2e` (build tag `e2e`).

## Como executar
- Windows: `backend\go\libs\test\integration\run-tests.bat --suite integration --with-compose`
- Linux/macOS: `backend/go/libs/test/integration/run-tests.sh --suite integration --with-compose`
- Para rodar ambas as suítes: `--suite all`. Adicione argumentos do `go test` após `--`, ex.: `-- --run TestIntegrationPublisher`.

## Notas
- Usa o `docker-compose.tests.yml` da raiz quando `--with-compose` é informado. Platform Events precisa de Kafka; o helper garante os tópicos `auth.user.created`, `tenant.created` e `agent.created` quando o compose é iniciado.
- Bibliotecas sem `test/<suite>/*_test.go` são ignoradas automaticamente.
- Entrada para CI/CD: chame `backend/go/libs/test/integration/run-tests.sh --suite integration` (ou `--suite all`) a partir da raiz do repositório; inclua `--with-compose` quando precisar de dependências efêmeras.
