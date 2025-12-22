# Platform Observability — Integration Tests

Helper scripts to run the library integration suite.

## How to run
- Windows: `backend\go\libs\platform-observability\test\integration\run-integration-tests.bat` (add `--with-compose` to reuse the root test stack).
- Linux/macOS: `backend/go/libs/platform-observability/test/integration/run-integration-tests.sh` (use `--with-compose` when you want `docker-compose.tests.yml`).
- Pass extra `go test` args after the script, e.g., `-run TestSomething`.

## Notes
- No external services are required for the placeholder tests; `--with-compose` is optional and just reuses the root stack when needed.
- Build tag: `integration`. Tests live under `test/integration`.
- VS Code picks up the tag; CI can call this script or `backend/go/libs/test/integration/run-tests.sh --suite integration`.
