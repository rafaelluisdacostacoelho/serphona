#!/usr/bin/env bash
set -euo pipefail

# Runs integration tests for Go services using the `integration` build tag.
# Usage: ./run-integration-tests.sh [--with-compose] [--keep-compose] [go test args]
# Notes:
# - This helper only starts the root docker-compose.tests.yml when --with-compose is used (Kafka stack).
# - Service-specific dependencies (e.g., DATABASE_URL) must be provided by the environment or your own compose stack.

USE_COMPOSE=false
KEEP_COMPOSE=false
GO_ARGS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-compose)
      USE_COMPOSE=true
      shift
      ;;
    --keep-compose)
      KEEP_COMPOSE=true
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [--with-compose] [--keep-compose] [go test args]"
      exit 0
      ;;
    *)
      GO_ARGS+=("$1")
      shift
      ;;
  esac
done

ROOT_DIR=$(git -C "$(dirname "$0")" rev-parse --show-toplevel 2>/dev/null || cd "$(dirname "$0")/../../.." && pwd)
cd "$ROOT_DIR"

COMPOSE_CMD=()
COMPOSE_FILE="$ROOT_DIR/docker-compose.tests.yml"
STACK_STARTED=false

if $USE_COMPOSE; then
  if command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD=(docker-compose)
  elif command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD=(docker compose)
  else
    echo "[error] docker compose is required for --with-compose" >&2
    exit 1
  fi

  echo "[info] starting test stack using $COMPOSE_FILE"
  "${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" up -d
  STACK_STARTED=true

  echo "[info] waiting for stack to be ready"
  sleep 8

  if ! $KEEP_COMPOSE; then
    trap 'if $STACK_STARTED; then "${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" down; fi' EXIT
  fi
fi

SERVICES=(
  tenant-manager
  auth-gateway
  billing-service
  analytics-query-service
  agent-orchestrator
  tools-gateway
  voice-gateway
)

for svc in "${SERVICES[@]}"; do
  svc_dir="$ROOT_DIR/backend/go/services/$svc"
  test_dir="$svc_dir/test/integration"

  if [[ ! -d "$test_dir" ]]; then
    echo "[skip] $svc has no test/integration; skipping"
    continue
  fi

  if ! ls "$test_dir"/*_test.go >/dev/null 2>&1; then
    echo "[skip] $svc integration dir has no *_test.go; skipping"
    continue
  fi

  echo "[info] running integration tests for $svc"
  (cd "$svc_dir" && go test -tags=integration ./test/integration "${GO_ARGS[@]}")

done

if $STACK_STARTED && $KEEP_COMPOSE; then
  echo "[info] leaving test stack running (requested with --keep-compose)"
fi
