# Platform RAG — Testes de Integração

Scripts de apoio para rodar a suíte de integração da biblioteca.

## Como executar
- Windows: `backend\go\libs\platform-rag\test\integration\run-integration-tests.bat` (use `--with-compose` para reutilizar o stack de testes da raiz).
- Linux/macOS: `backend/go/libs/platform-rag/test/integration/run-integration-tests.sh` (use `--with-compose` quando quiser o `docker-compose.tests.yml`).
- Passe argumentos extras do `go test` após o script, ex.: `-run TestSomething`.

## Notas
- Nenhum serviço externo é necessário para os testes placeholder; `--with-compose` é opcional e apenas reutiliza o stack raiz se precisar.
- Build tag: `integration`. Os testes ficam em `test/integration`.
- O VS Code já reconhece a tag; o CI pode chamar este script ou `backend/go/libs/test/integration/run-tests.sh --suite integration`.
