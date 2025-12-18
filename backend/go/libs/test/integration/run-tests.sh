#!/usr/bin/env bash
set -euo pipefail

# Runs Go library test suites (integration, e2e) with optional docker-compose stack.
# Usage: ./run-tests.sh [--suite integration|e2e|all] [--with-compose] [--keep-compose] [-- go test args]
# Defaults to integration when no suite is provided.

SUITES=()
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
    --suite)
      if [[ $# -lt 2 ]]; then
        echo "[error] --suite requires a value (integration|e2e|all)" >&2
        exit 1
      fi
      case "$2" in
        integration)
          SUITES=("integration")
          ;;
        e2e)
          SUITES=("e2e")
          ;;
        all)
          SUITES=("integration" "e2e")
          ;;
        *)
          echo "[error] unknown suite: $2" >&2
          exit 1
          ;;
      esac
      shift 2
      ;;
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
      echo "Usage: $0 [--suite integration|e2e|all] [--with-compose] [--keep-compose] [-- go test args]"
      exit 0
      ;;
    *)
      GO_ARGS+=("$1")
      shift
      ;;
  esac
done

if [[ ${#SUITES[@]} -eq 0 ]]; then
  SUITES=("integration")
fi

ROOT_DIR=$(git -C "$(dirname "$0")" rev-parse --show-toplevel 2>/dev/null || cd "$(dirname "$0")/../../../.." && pwd)
cd "$ROOT_DIR"

COMPOSE_CMD=()
COMPOSE_FILE="$ROOT_DIR/docker-compose.tests.yml"
STACK_STARTED=false

topics=(auth.user.created tenant.created agent.created)

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

  for topic in "${topics[@]}"; do
    echo "[info] ensuring topic $topic exists"
    "${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" exec -T kafka kafka-topics \
      --bootstrap-server localhost:9092 \
      --create --if-not-exists --topic "$topic" --partitions 1 --replication-factor 1
  done

  if ! $KEEP_COMPOSE; then
    trap 'if $STACK_STARTED; then "${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" down; fi' EXIT
  fi
fi

LIBS=(
  platform-core
  platform-auth
  platform-events
  platform-observability
)

run_suite_for_lib() {
  local suite="$1"
  local lib="$2"
  local lib_dir="$ROOT_DIR/backend/go/libs/$lib"
  local test_dir="$lib_dir/test/$suite"

  if [[ ! -d "$test_dir" ]]; then
    echo "[skip] $lib has no test/$suite; skipping"
    return 0
  fi

  if ! ls "$test_dir"/*_test.go >/dev/null 2>&1; then
    echo "[skip] $lib test/$suite has no *_test.go; skipping"
    return 0
  fi

  if [[ "$lib" == "platform-events" && "$suite" == "integration" ]]; then
    export KAFKA_BROKERS=${KAFKA_BROKERS:-localhost:9092}
  fi

  echo "[info] running $suite tests for $lib"
  (cd "$lib_dir" && go test -tags="$suite" "./test/$suite" "${GO_ARGS[@]}")
}

for suite in "${SUITES[@]}"; do
  for lib in "${LIBS[@]}"; do
    run_suite_for_lib "$suite" "$lib"
  done
done

if $STACK_STARTED && $KEEP_COMPOSE; then
  echo "[info] leaving test stack running (requested with --keep-compose)"
fi
