#!/usr/bin/env bash
set -euo pipefail

# Rebuild and start key services with Docker Compose v2.
# Customize the service list via SERVICES env (space-separated) or CLI args.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# Prefer CLI args over SERVICES env; fallback to defaults.
if [[ $# -gt 0 ]]; then
  SERVICES=("$@")
elif [[ -n "${SERVICES:-}" ]]; then
  # shellcheck disable=SC2206
  SERVICES=(${SERVICES})
else
  SERVICES=(
    tenant-manager
    auth-gateway
    agent-orchestrator
    tools-gateway
    billing-service
    analytics-processor
  )
fi

if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
  echo "[compose-rebuild] Using docker compose v2 (cli plugin)…" >&2
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
  echo "[compose-rebuild] docker compose plugin not found; using docker-compose (v1)…" >&2
else
  echo "[compose-rebuild] ERROR: docker compose plugin or docker-compose not found" >&2
  exit 1
fi

$COMPOSE -f docker-compose.yml build "${SERVICES[@]}"
$COMPOSE -f docker-compose.yml up -d "${SERVICES[@]}"

echo "[compose-rebuild] Done. Services: ${SERVICES[*]}" >&2
