# Tenant Manager smoke test (local kind)

## Prereqs
- kind cluster `serphona-kind` with context on namespace `serphona`.
- Base deps (Postgres, Redis) installed per repo docs.
- Image `serphona/tenant-manager:dev` loaded into the cluster.
- Required secrets for the chart (DB, redis, jwt/oidc) already created via local overrides.

## Deploy/upgrade
```bash
helm upgrade --install tenant-manager infra/helm/tenant-manager \
  -n serphona \
  -f infra/helm/overrides/tenant-manager-values.yaml \
  --set global.imageRegistry=serphona \
  --set image.tag=dev \
  --set serviceMonitor.enabled=false \
  --set prometheusRule.enabled=false \
  --set grafanaDashboard.enabled=false
kubectl rollout status deploy/tenant-manager -n serphona --timeout=180s
```

## HS256 token (dev)
Use `dev-jwt-secret`, `iss=serphona`, `aud=serphona-services`, `tenantId` (camelCase) and `tenants:read` scope:
```bash
python3 - <<'PY'
import jwt, time
secret='dev-jwt-secret'
now=int(time.time())
payload={
    'service':'tenant-manager-smoke',
    'tenantId':'5843292c-2b31-46ca-a6ba-c8e73a0dc2ed',
    'scopes':['tenants:read'],
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
Pick a specific pod to avoid tunnel drops; adjust local port if needed (e.g., 19081 -> 8080):
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=tenant-manager -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 19081:8080
```
Keep in a dedicated terminal.

## Smoke calls
### Health
```bash
curl -i http://127.0.0.1:19081/health
```
Expected: `200` with `healthy`.

### Metrics (optional)
```bash
curl -i http://127.0.0.1:19081/metrics | head
```
Expected: Prometheus metrics text.

### GET /api/v1/tenants
```bash
curl -i -sS http://127.0.0.1:19081/api/v1/tenants \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 5843292c-2b31-46ca-a6ba-c8e73a0dc2ed"
```
Expected: `200` with a tenant list (possibly empty). If 401/403, recheck issuer/audience/tenant claim and scopes.

## Logs
```bash
kubectl logs -n serphona deploy/tenant-manager --tail=50
```
For the forwarded pod: `kubectl logs -n serphona $POD --tail=50`.

## Cleanup
- Stop the port-forward (Ctrl+C or `kill <pid>` if backgrounded).
- If not needed, optionally scale down: `kubectl scale deploy/tenant-manager -n serphona --replicas=0`.
