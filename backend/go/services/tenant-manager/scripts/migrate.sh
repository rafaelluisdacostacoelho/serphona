#!/bin/bash
set -euo pipefail

DATABASE_URL=${DATABASE_URL:-"postgres://postgres:postgres@localhost:5432/tenant_management?sslmode=disable"}
MIGRATIONS_PATH=${MIGRATIONS_PATH:-"./migrations"}

echo "Running migrations from ${MIGRATIONS_PATH} on ${DATABASE_URL}"
migrate -path "${MIGRATIONS_PATH}" -database "${DATABASE_URL}" up
