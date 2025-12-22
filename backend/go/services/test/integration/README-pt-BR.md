# Serviços Go — Testes de Integração/E2E

Ajuda unificada para as suítes de serviços Go que usam as build tags `integration` ou `e2e`. Todos os scripts e notas ficam nesta pasta.

## Como executar
- Windows: `backend\go\services\test\integration\run-tests.bat --suite integration --with-compose`
- Linux/macOS: `backend/go/services/test/integration/run-tests.sh --suite integration --with-compose`
- Para rodar ambas as suítes: adicione `--suite all`. Passe argumentos adicionais do `go test` após `--`, por exemplo, `-- --run TestUserFlow`.

## Notas
- O helper pode iniciar opcionalmente o `docker-compose.tests.yml` (stack Kafka). Dependências específicas do serviço (por exemplo, `DATABASE_URL`) devem vir do seu ambiente ou de um compose específico do serviço.
- Serviços sem arquivos `test/<suite>/*_test.go` são ignorados automaticamente.
- Entrada de CI: chame `backend/go/services/test/integration/run-tests.sh --suite integration` (ou `--suite all`) a partir da raiz do repositório; use `--with-compose` quando precisar de dependências efêmeras.
