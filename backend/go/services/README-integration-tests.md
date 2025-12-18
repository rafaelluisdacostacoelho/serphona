# Go Services — Integration Tests

This helper standardizes how to run Go service integration tests using the `integration` build tag.

## How to run
- Windows: `backend\go\services\run-integration-tests.bat --with-compose`
- Linux/macOS: `backend/go/services/run-integration-tests.sh --with-compose`
- Append any `go test` arguments after the script, e.g. `-- -run TestUserFlow`.

## Notes
- The helper optionally starts the root `docker-compose.tests.yml` (Kafka stack). Service-specific deps (e.g., `DATABASE_URL`) must be provided by your env or your own compose stack.
- Services without `test/integration/*_test.go` are skipped.
- Tests run with `-tags=integration` in each service directory under `backend/go/services`.
