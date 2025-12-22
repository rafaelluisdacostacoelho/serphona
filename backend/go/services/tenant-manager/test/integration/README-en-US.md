# Tenant Manager — Integration Tests

Quick guide to run the integration suite under `test/integration` (build tag `integration`).

## How to run
- Windows: `backend\go\services\tenant-manager\test\integration\run-integration-tests.bat --with-compose`
- Linux/macOS: `backend/go/services/tenant-manager/test/integration/run-integration-tests.sh --with-compose`
- Append `go test` args after the script, e.g. `-- -run TestTenantCreateIntegration`.

## Notes
- The optional `--with-compose` flag starts the root `docker-compose.tests.yml` stack (Kafka, etc.). Service-specific vars (e.g., `DATABASE_URL`) must be provided externally.
- Uses the `integration` build tag; suites live in `test/integration`.
- For cross-service runs, prefer the shared helper at `backend/go/services/test/integration/run-tests.*`.
