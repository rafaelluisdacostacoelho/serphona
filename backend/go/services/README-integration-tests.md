# Go Services — Integration/E2E Tests

Unified helpers to run Go service suites under `test/integration` (build tag `integration`) and `test/e2e` (build tag `e2e`).

## How to run
- Windows: `backend\go\services\test\integration\run-tests.bat --suite integration --with-compose`
- Linux/macOS: `backend/go/services/test/integration/run-tests.sh --suite integration --with-compose`
- To run both suites: add `--suite all`. Append `go test` args after `--`, e.g. `-- --run TestUserFlow`.
- Legacy shims still work: `run-integration-tests.*` delegates to `test/integration/run-tests.* --suite integration`.

## Notes
- The helper optionally starts the root `docker-compose.tests.yml` (Kafka stack). Service-specific deps (e.g., `DATABASE_URL`) must come from your environment or a service-specific compose stack.
- Services without `test/<suite>/*_test.go` are skipped automatically.
- CI/CD entrypoint: call `backend/go/services/test/integration/run-tests.sh --suite integration` (or `--suite all`) from the repo root; pass `--with-compose` when ephemeral deps are needed.
