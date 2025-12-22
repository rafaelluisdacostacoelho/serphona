# Go Libraries — Integration/E2E Tests

Unified helpers to run library suites under `test/integration` (build tag `integration`) and `test/e2e` (build tag `e2e`).

## How to run
- Windows: `backend\go\libs\test\integration\run-tests.bat --suite integration --with-compose`
- Linux/macOS: `backend/go/libs/test/integration/run-tests.sh --suite integration --with-compose`
- To run both suites: `--suite all`. Append `go test` args after `--`, e.g. `-- --run TestIntegrationPublisher`.

## Notes
- Uses the root `docker-compose.tests.yml` when `--with-compose` is provided. Platform Events needs Kafka; the helper ensures the topics `auth.user.created`, `tenant.created`, and `agent.created` exist when compose is started.
- Libraries without `test/<suite>/*_test.go` are skipped automatically.
- CI/CD entrypoint: invoke `backend/go/libs/test/integration/run-tests.sh --suite integration` (or `--suite all`) from the repo root; add `--with-compose` when ephemeral deps are required.
