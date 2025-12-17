# Platform Events — Integration Tests

Integration tests here exercise Kafka-backed flows. They are disabled by default and guarded by the `integration` build tag.

## Prerequisites
- Go toolchain available in `$PATH`
- Kafka reachable and advertised via `KAFKA_BROKERS` (e.g., `localhost:9092`)
- Optional: `DEBUG=true` to enable verbose logs

## How to run
- From repo root with go directly:
  ```bash
  # Linux/macOS
  KAFKA_BROKERS=localhost:9092 go test -tags=integration ./backend/go/libs/platform-events/test/integration
  ```
  ```powershell
  # PowerShell (valor vale apenas na sessão atual)
  $env:KAFKA_BROKERS="localhost:9092"; go test -tags=integration ./backend/go/libs/platform-events/test/integration; Remove-Item Env:KAFKA_BROKERS
  ```
- Using the helper scripts (they default `KAFKA_BROKERS` to `localhost:9092` if unset):
  ```bash
  # Linux/macOS
  backend/go/libs/platform-events/test/integration/run-integration-tests.sh --with-compose
  ```
  ```powershell
  # Windows (Command Prompt or PowerShell)
  backend\go\libs\platform-events\test\integration\run-integration-tests.bat --with-compose
  ```
  Pass extra `go test` args after the script (e.g., `-run TestIntegrationPublisher`). Use `--keep-compose` to leave the Docker stack up after tests.
  The helpers automatically ensure the core topics (`auth.user.created`, `tenant.created`, `agent.created`) exist when `--with-compose` is used.

### Spinning up dependencies quickly
- Minimal stack for tests (Kafka + Zookeeper):
  ```bash
  docker-compose -f docker-compose.tests.yml up -d
  ```
  Then run tests with `KAFKA_BROKERS=localhost:9092`.

### Creating required topics
If auto-create is disabled on Kafka, create the topics before running:
```bash
docker exec serphona-kafka-1 kafka-topics --create --topic auth.user.created --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
docker exec serphona-kafka-1 kafka-topics --create --topic agent.created       --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
```
If your container name differs, check with `docker ps`. Alternatively, set `KAFKA_AUTO_CREATE_TOPICS_ENABLE=true` in `docker-compose.tests.yml` (already set by default) and recreate the stack.

## Notes
- VS Code is already configured to include the `integration` tag for analysis (.vscode/settings.json).
- The `test/e2e` folder is reserved for future end-to-end suites; consider a separate `e2e` build tag when adding them.
