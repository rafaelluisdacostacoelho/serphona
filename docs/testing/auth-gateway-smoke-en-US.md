# Auth Gateway smoke test (local kind)

## Prereqs
- kind cluster `serphona-kind` with kubectl context on namespace `serphona`.
- Base deps (Postgres, Redis) installed per repo docs.
- Image `serphona/auth-gateway:dev` loaded into the cluster.
- Required secrets for the chart (jwt/oidc, DB, redis) already created via local overrides.

## Deploy/upgrade
```bash
helm upgrade --install auth-gateway infra/helm/auth-gateway \
  -n serphona \
  -f infra/helm/overrides/auth-gateway-values.yaml \
  --set global.imageRegistry=serphona \
  --set image.tag=dev \
  --set serviceMonitor.enabled=false \
  --set prometheusRule.enabled=false \
  --set grafanaDashboard.enabled=false
kubectl rollout status deploy/auth-gateway -n serphona --timeout=180s
```

## Port-forward
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=auth-gateway -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 18080:8080
```
Keep it running in a dedicated terminal.

## Smoke calls
### Health
```bash
curl -i http://127.0.0.1:18080/health
```
Expected: `200` with `healthy` payload.

### /metrics (optional)
```bash
curl -i http://127.0.0.1:18080/metrics | head
```
Expected: Prometheus metrics text.

### Token flow (when configured)
- If using OIDC/JWKS, obtain a valid token for the gateway audience and hit protected endpoints (e.g., `/api/v1/auth/validate`).
- If a dev HS256 secret is configured, mint a token with matching `aud/iss` before calling protected routes.

## Logs
```bash
kubectl logs -n serphona deploy/auth-gateway --tail=50
```
For the forwarded pod: `kubectl logs -n serphona $POD --tail=50`.

## Cleanup
- Stop the port-forward (Ctrl+C or `kill <pid>` if backgrounded).
- Optionally scale down if not needed: `kubectl scale deploy/auth-gateway -n serphona --replicas=0`.
