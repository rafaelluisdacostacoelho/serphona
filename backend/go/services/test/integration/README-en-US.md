# Go Services — Integration/E2E Tests

Unified helpers for Go service suites that use the `integration` or `e2e` build tags. All runner scripts and notes live in this folder.

## How to run
- Windows: `backend\go\services\test\integration\run-tests.bat --suite integration --with-compose`
- Linux/macOS: `backend/go/services/test/integration/run-tests.sh --suite integration --with-compose`
- Run both suites: add `--suite all`. Append `go test` args after `--`, e.g., `-- --run TestUserFlow`.

## Notes
- The helper can optionally start `docker-compose.tests.yml` (Kafka stack). Service-specific dependencies (e.g., `DATABASE_URL`) must come from your environment or a service-specific compose stack.
- Services without `test/<suite>/*_test.go` are skipped automatically.
- CI entrypoint: call `backend/go/services/test/integration/run-tests.sh --suite integration` (or `--suite all`) from the repo root; pass `--with-compose` when ephemeral deps are needed.
