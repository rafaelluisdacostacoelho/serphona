# Billing Service smoke test (local kind)

## Prereqs
- kind cluster `serphona-kind` with context on namespace `serphona`.
- Base deps installed (Postgres, Redis, Kafka if using events) per repo docs.
- Image `serphona/billing-service:dev` loaded into the cluster.
- Required secrets for the chart (DB, redis, jwt/oidc or HS256 dev, event sinks) already created via local overrides.

## Deploy/upgrade
```bash
helm upgrade --install billing-service infra/helm/billing-service \
  -n serphona \
  -f infra/helm/overrides/billing-service-values.yaml \
  --set global.imageRegistry=serphona \
  --set image.tag=dev \
  --set serviceMonitor.enabled=false \
  --set prometheusRule.enabled=false \
  --set grafanaDashboard.enabled=false
kubectl rollout status deploy/billing-service -n serphona --timeout=180s
```

## HS256 token (dev)
Use `dev-jwt-secret`, `iss=serphona`, `aud=serphona-services`, `tenantId` claim and a read scope (adjust if endpoint needs more):
```bash
python3 - <<'PY'
import jwt, time
secret='dev-jwt-secret'
now=int(time.time())
payload={
    'service':'billing-smoke',
    'tenantId':'5843292c-2b31-46ca-a6ba-c8e73a0dc2ed',
    'scopes':['billing:read'],
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
Pick a specific pod; adjust local port if needed (e.g., 19082 -> 8080):
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=billing-service -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 19082:8080
```
Keep in a dedicated terminal.

## Smoke calls
### Health
```bash
curl -i http://127.0.0.1:19082/health
```
Expected: `200` with `healthy` payload.

### Metrics (optional)
```bash
curl -i http://127.0.0.1:19082/metrics | head
```
Expected: Prometheus metrics text.

### GET /api/v1/usage (or main billing endpoint)
```bash
curl -i -sS http://127.0.0.1:19082/api/v1/usage \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 5843292c-2b31-46ca-a6ba-c8e73a0dc2ed"
```
Expected: `200` (list may be empty). If 401/403, recheck issuer/audience/tenantId/scopes.

## Logs
```bash
kubectl logs -n serphona deploy/billing-service --tail=50
```
For the forwarded pod: `kubectl logs -n serphona $POD --tail=50`.

## Cleanup
- Stop the port-forward (Ctrl+C or `kill <pid>` if backgrounded).
- Optionally scale down if not needed: `kubectl scale deploy/billing-service -n serphona --replicas=0`.
