#!/usr/bin/env bash
set -euo pipefail

# Runs tenant-manager integration tests with optional docker-compose stack.
# Usage: ./run-integration-tests.sh [--with-compose] [--keep-compose] [-- go test args]

USE_COMPOSE=false
KEEP_COMPOSE=false
GO_ARGS=()
PASSTHRU=false

while [[ $# -gt 0 ]]; do
  if $PASSTHRU; then
    GO_ARGS+=("$1")
    shift
    continue
  fi
  case "$1" in
    --with-compose)
      USE_COMPOSE=true
      shift
      ;;
    --keep-compose)
      KEEP_COMPOSE=true
      shift
      ;;
    --)
      PASSTHRU=true
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [--with-compose] [--keep-compose] [-- go test args]"
      exit 0
      ;;
    *)
      GO_ARGS+=("$1")
      shift
      ;;
  esac
done

SCRIPT_DIR=$(cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null || cd "$SCRIPT_DIR/../../../.." && pwd)
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

echo "[info] running tenant-manager integration tests"
cd "$ROOT_DIR/backend/go/services/tenant-manager"
go test -tags=integration ./test/integration "${GO_ARGS[@]}"

if $STACK_STARTED && $KEEP_COMPOSE; then
  echo "[info] leaving test stack running (requested with --keep-compose)"
fi
