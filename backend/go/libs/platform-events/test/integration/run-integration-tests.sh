#!/usr/bin/env bash
set -euo pipefail

# Runs the integration tests for platform-events.
# Usage: ./run-integration-tests.sh [--with-compose] [--keep-compose] [extra go test args]

set -euo pipefail

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

KAFKA_BROKERS_VALUE=${KAFKA_BROKERS:-localhost:9092}
export KAFKA_BROKERS="$KAFKA_BROKERS_VALUE"

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

  echo "[info] starting Kafka test stack using $COMPOSE_FILE"
  "${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" up -d
  STACK_STARTED=true

  echo "[info] waiting for Kafka to be ready"
  sleep 8

  for topic in auth.user.created tenant.created agent.created; do
    echo "[info] ensuring topic $topic exists"
    "${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" exec -T kafka kafka-topics \
      --bootstrap-server localhost:9092 \
      --create --if-not-exists --topic "$topic" --partitions 1 --replication-factor 1
  done

  for topic in auth.user.created tenant.created agent.created; do
    echo "[info] verifying topic $topic availability"
    ready=false
    for _ in {1..10}; do
      if "${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" exec -T kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic "$topic" >/dev/null 2>&1; then
        ready=true
        break
      fi
      sleep 1
    done

    if ! $ready; then
      echo "[error] topic $topic did not become available" >&2
      exit 1
    fi
  done

  if ! $KEEP_COMPOSE; then
    trap 'if $STACK_STARTED; then "${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" down; fi' EXIT
  fi
fi

GO_TEST_FLAGS=("-tags=integration")
GO_TEST_PATH="./test/integration"
MODULE_DIR="$ROOT_DIR/backend/go/libs/platform-events"

echo "[info] using KAFKA_BROKERS=$KAFKA_BROKERS_VALUE"
echo "[info] running integration tests in $MODULE_DIR/$GO_TEST_PATH"
cd "$MODULE_DIR"
if (( ${#GO_TEST_ARGS[@]} )); then
  go test "${GO_TEST_FLAGS[@]}" "$GO_TEST_PATH" "${GO_TEST_ARGS[@]}"
else
  go test "${GO_TEST_FLAGS[@]}" "$GO_TEST_PATH"
fi

if $STACK_STARTED && $KEEP_COMPOSE; then
  echo "[info] leaving Kafka test stack running (requested with --keep-compose)"
fi
