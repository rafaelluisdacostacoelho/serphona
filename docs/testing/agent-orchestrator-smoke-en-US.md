# Agent Orchestrator smoke test (local kind)

## Prereqs
- kind cluster `serphona-kind` with context on namespace `serphona`.
- Base deps (Postgres/Redis, Kafka if events are used) installed per repo docs.
- Image `serphona/agent-orchestrator:dev` loaded into the cluster.
- Required secrets/configs for the chart (DB, redis, jwt/oidc or HS256 dev) already created via local overrides.

## Deploy/upgrade
```bash
helm upgrade --install agent-orchestrator infra/helm/agent-orchestrator \
  -n serphona \
  -f infra/helm/overrides/agent-orchestrator-values.yaml \
  --set global.imageRegistry=serphona \
  --set image.tag=dev \
  --set serviceMonitor.enabled=false \
  --set prometheusRule.enabled=false \
  --set grafanaDashboard.enabled=false
kubectl rollout status deploy/agent-orchestrator -n serphona --timeout=180s
```

## HS256 token (dev)
Use `dev-jwt-secret`, `iss=serphona`, `aud=serphona-services`, `tenantId` claim (camelCase) and read/execute scopes for agents (adjust if needed):
```bash
python3 - <<'PY'
import jwt, time
secret='dev-jwt-secret'
now=int(time.time())
payload={
    'service':'agent-orchestrator-smoke',
    'tenantId':'5843292c-2b31-46ca-a6ba-c8e73a0dc2ed',
    'scopes':['agents:read','agents:execute'],
    'iss':'serphona',
    'aud':'serphona-services',
    'iat':now,
    'exp':now+3600,
}
print(jwt.encode(payload, secret, algorithm='HS256'))
PY
```
Store in `TOKEN`.

## Port-forward
Pick a specific pod (adjust local port if needed, e.g., 19083 -> 8080):
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=agent-orchestrator -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 19083:8080
```

## Smoke calls
### Health
```bash
curl -i http://127.0.0.1:19083/health
```
Expected: `200` with `healthy` payload.

### Metrics (optional)
```bash
curl -i http://127.0.0.1:19083/metrics | head
```
Expected: Prometheus metrics text.

### GET /api/v1/agents (main route)
```bash
curl -i -sS http://127.0.0.1:19083/api/v1/agents \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 5843292c-2b31-46ca-a6ba-c8e73a0dc2ed"
```
Expected: `200` (list may be empty). If 401/403, recheck issuer/audience/tenantId/scopes.

## Logs
```bash
kubectl logs -n serphona deploy/agent-orchestrator --tail=50
```
For the forwarded pod: `kubectl logs -n serphona $POD --tail=50`.

## Cleanup
- Stop the port-forward (Ctrl+C or `kill <pid>` if backgrounded).
- Optionally scale down if not needed: `kubectl scale deploy/agent-orchestrator -n serphona --replicas=0`.
