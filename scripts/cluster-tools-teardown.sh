#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-serphona}"

kill_pf() {
  local name="$1" pid_file="/tmp/${name}-pf.pid"
  if [ -f "$pid_file" ]; then
    local pid
    pid=$(cat "$pid_file")
    if kill -0 "$pid" >/dev/null 2>&1; then
      echo "Stopping port-forward ${name} (pid $pid)"
      kill "$pid" >/dev/null 2>&1 || true
    fi
    rm -f "$pid_file" "/tmp/${name}-pf.log"
  fi
}

kill_pf kafka-ui
kill_pf redis-ui
kill_pf minio-ui
kill_pf clickhouse-ui

# Delete UI resources
helm uninstall rp-console -n "$NAMESPACE" >/dev/null 2>&1 || true
kubectl delete deploy,svc redis-commander cloudbeaver tabix -n "$NAMESPACE" --ignore-not-found

echo "Teardown complete."
