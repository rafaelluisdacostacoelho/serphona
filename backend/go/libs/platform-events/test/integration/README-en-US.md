# Platform Events — Integration Tests

Quick guide to run Kafka-backed integration tests.

## Prerequisites
- Go toolchain available in `PATH`.
- Kafka reachable through `KAFKA_BROKERS` (e.g., `localhost:9092`).
- Optional: `DEBUG=true` for verbose logs.

## How to run
- Using the helper (brings up Kafka and creates topics when `--with-compose` is set):
  ```powershell
  # Windows (Command Prompt or PowerShell)
  backend\go\libs\platform-events\test\integration\run-integration-tests.bat --with-compose
  ```
  ```bash
  # Linux/macOS
  backend/go/libs/platform-events/test/integration/run-integration-tests.sh --with-compose
  ```
  Tips: append `go test` args after the script (e.g., `-run TestIntegrationPublisher`). Use `--keep-compose` to keep Kafka running after tests.

- Manual execution (Kafka already running):
  ```powershell
  $env:KAFKA_BROKERS="localhost:9092"; go test -tags=integration ./backend/go/libs/platform-events/test/integration; Remove-Item Env:KAFKA_BROKERS
  ```
  ```bash
  KAFKA_BROKERS=localhost:9092 go test -tags=integration ./backend/go/libs/platform-events/test/integration
  ```

## Spinning up dependencies quickly
- Minimal stack (Kafka + Zookeeper):
  ```bash
  docker compose -f docker-compose.tests.yml up -d
  ```
  Then run tests with `KAFKA_BROKERS=localhost:9092`.

## Required topics
- Helpers create and verify `auth.user.created`, `tenant.created`, `agent.created` when `--with-compose` is used.
- If you must create them manually (auto-create disabled):
  ```bash
  docker compose -f docker-compose.tests.yml exec -T kafka kafka-topics --bootstrap-server localhost:9092 --create --topic auth.user.created --partitions 1 --replication-factor 1
  docker compose -f docker-compose.tests.yml exec -T kafka kafka-topics --bootstrap-server localhost:9092 --create --topic agent.created       --partitions 1 --replication-factor 1
  docker compose -f docker-compose.tests.yml exec -T kafka kafka-topics --bootstrap-server localhost:9092 --create --topic tenant.created      --partitions 1 --replication-factor 1
  ```

## Notes
- VS Code is configured to include the `integration` build tag for analysis.
- `test/e2e` is reserved for future end-to-end suites.
