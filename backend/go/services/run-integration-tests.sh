#!/usr/bin/env bash
set -euo pipefail

# Backward-compatible shim that delegates to the shared test/integration runner for integration suite only.
SCRIPT_DIR=$(cd -- "$(dirname -- "$0")" && pwd)
exec "$SCRIPT_DIR/test/integration/run-tests.sh" --suite integration "$@"
