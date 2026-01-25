# Analytics Query Service smoke test (local kind)

## Prereqs
- kind cluster `serphona-kind` with kubectl context and namespace `serphona`.
- Dependencies up (ClickHouse; Postgres/Redis if required) per repo docs.
- Image `serphona/analytics-query-service:dev` built/loaded into the cluster.
- Required secrets/configs for the chart present (ClickHouse creds, jwt/oidc or dev HS256 secret) via local overrides.

## Deploy/upgrade
```bash
helm upgrade --install analytics-query-service infra/helm/analytics-query-service \
  -n serphona \
  -f infra/helm/overrides/analytics-query-service-values.yaml \
  --set global.imageRegistry=serphona \
  --set image.tag=dev \
  --set serviceMonitor.enabled=false \
  --set prometheusRule.enabled=false \
  --set grafanaDashboard.enabled=false
kubectl rollout status deploy/analytics-query-service -n serphona --timeout=180s
```

## HS256 token (dev)
Use `dev-jwt-secret`, `iss=serphona`, `aud=serphona-services`, camelCase `tenantId`, scope `analytics:read`:
```bash
python3 - <<'PY'
import jwt, time
secret='dev-jwt-secret'
now=int(time.time())
payload={
    'service':'analytics-query-smoke',
    'tenantId':'5843292c-2b31-46ca-a6ba-c8e73a0dc2ed',
    'scopes':['analytics:read'],
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
Pick a specific pod; adjust local port if needed (e.g., 19084 -> 8080):
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=analytics-query-service -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 19084:8080
```

## Smoke calls
### Health
```bash
curl -i http://127.0.0.1:19084/health
```
Expected: `200` and `healthy` payload.

### Metrics (optional)
```bash
curl -i http://127.0.0.1:19084/metrics | head
```
Expect Prometheus metrics output.

### Simple query (example)
Adjust the route to the service's query endpoint (e.g., `/api/v1/query` or `/api/v1/analytics/query`).
```bash
curl -i -sS http://127.0.0.1:19084/api/v1/query \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 5843292c-2b31-46ca-a6ba-c8e73a0dc2ed" \
  -H "Content-Type: application/json" \
  --data '{"sql":"SELECT 1"}'
```
Expected: `200` with query result. If 401/403, check issuer/audience/tenantId/scopes. If ClickHouse errors, verify credentials/URL.

## Logs
```bash
kubectl logs -n serphona deploy/analytics-query-service --tail=50
```
For the port-forwarded pod: `kubectl logs -n serphona $POD --tail=50`.

## Cleanup
- Stop the port-forward (Ctrl+C or `kill <pid>` if backgrounded).
- Optional: scale down if not needed: `kubectl scale deploy/analytics-query-service -n serphona --replicas=0`.
