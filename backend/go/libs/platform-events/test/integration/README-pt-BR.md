# Platform Events — Testes de Integração

Guia rápido para rodar os testes de integração com Kafka.

## Pré-requisitos
- Go instalado no `PATH`.
- Kafka acessível via `KAFKA_BROKERS` (ex.: `localhost:9092`).
- Opcional: `DEBUG=true` para logs detalhados.

## Como executar
- Usando o helper (sobe Kafka e cria tópicos automaticamente quando `--with-compose` é usado):
  ```powershell
  # Windows (Prompt ou PowerShell)
  backend\go\libs\platform-events\test\integration\run-integration-tests.bat --with-compose
  ```
  ```bash
  # Linux/macOS
  backend/go/libs/platform-events/test/integration/run-integration-tests.sh --with-compose
  ```
  Dicas: adicione argumentos do `go test` após o script (ex.: `-run TestIntegrationPublisher`). Use `--keep-compose` se quiser manter o Kafka rodando.

- Execução manual (Kafka já rodando):
  ```powershell
  $env:KAFKA_BROKERS="localhost:9092"; go test -tags=integration ./backend/go/libs/platform-events/test/integration; Remove-Item Env:KAFKA_BROKERS
  ```
  ```bash
  KAFKA_BROKERS=localhost:9092 go test -tags=integration ./backend/go/libs/platform-events/test/integration
  ```

## Subindo dependências rapidamente
- Stack mínima (Kafka + Zookeeper):
  ```bash
  docker compose -f docker-compose.tests.yml up -d
  ```
  Em seguida rode os testes apontando `KAFKA_BROKERS=localhost:9092`.

## Tópicos exigidos
- Os helpers criam e validam `auth.user.created`, `tenant.created`, `agent.created` quando `--with-compose` é usado.
- Se precisar criar manualmente (auto-create desligado):
  ```bash
  docker compose -f docker-compose.tests.yml exec -T kafka kafka-topics --bootstrap-server localhost:9092 --create --topic auth.user.created --partitions 1 --replication-factor 1
  docker compose -f docker-compose.tests.yml exec -T kafka kafka-topics --bootstrap-server localhost:9092 --create --topic agent.created       --partitions 1 --replication-factor 1
  docker compose -f docker-compose.tests.yml exec -T kafka kafka-topics --bootstrap-server localhost:9092 --create --topic tenant.created      --partitions 1 --replication-factor 1
  ```

## Notas
- VS Code já inclui a build tag `integration` para análise.
- `test/e2e` fica reservado para futuras suítes end-to-end.
