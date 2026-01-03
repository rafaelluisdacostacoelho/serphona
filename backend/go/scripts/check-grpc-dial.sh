#!/usr/bin/env bash
set -euo pipefail

# Guardrail to keep gRPC clients using the shared helper.
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$REPO_ROOT"

matches=$(grep -R -n --include='*.go' --exclude-dir='vendor' -E 'grpc\.Dial(Context)?\s*\(' backend/go \
  | grep -v '_test\.go' \
  | grep -v '/internal/domain/service/grpc_client.go' || true)

if [[ -n "$matches" ]]; then
  echo "ERROR: direct grpc.Dial usage detected; use the shared client helper instead." >&2
  echo "$matches" >&2
  exit 1
fi
