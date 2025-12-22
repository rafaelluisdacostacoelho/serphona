#!/usr/bin/env bash
set -euo pipefail

# Runs the integration tests for platform-core.
# Usage: ./run-integration-tests.sh [--with-compose] [--keep-compose] [extra go test args]

USE_COMPOSE=false
KEEP_COMPOSE=false
GO_TEST_ARGS=()

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
      GO_TEST_ARGS+=("$1")
      shift
      ;;
  esac
done

ROOT_DIR=$(git -C "$(dirname "$0")" rev-parse --show-toplevel 2>/dev/null || cd "$(dirname "$0")/../../../../../.." && pwd)
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

GO_TEST_FLAGS=("-tags=integration")
GO_TEST_PATH="./test/integration"
MODULE_DIR="$ROOT_DIR/backend/go/libs/platform-core"

echo "[info] running integration tests in $MODULE_DIR/$GO_TEST_PATH"
cd "$MODULE_DIR"
if (( ${#GO_TEST_ARGS[@]} )); then
  go test "${GO_TEST_FLAGS[@]}" "$GO_TEST_PATH" "${GO_TEST_ARGS[@]}"
else
  go test "${GO_TEST_FLAGS[@]}" "$GO_TEST_PATH"
fi

if $STACK_STARTED && $KEEP_COMPOSE; then
  echo "[info] leaving test stack running (requested with --keep-compose)"
fi
