# Smoke test do Tenant Manager (local kind)

## Premissas
- Cluster kind `serphona-kind` com contexto kubectl no namespace `serphona`.
- Dependencias basicas instaladas (Postgres, Redis) conforme docs do repo.
- Imagem `serphona/tenant-manager:dev` carregada no cluster.
- Secrets referenciados pelo chart (DB, redis, jwt/oidc) ja criados via overrides locais.

## Implantar/atualizar
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

## Token HS256 (dev)
Use a mesma chave `dev-jwt-secret` e issuer/audience `serphona`/`serphona-services`, com claim `tenantId` (camelCase) e escopos de leitura:
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
Guarde em `TOKEN`.

## Port-forward
Use um pod especifico para evitar queda do tunel. Ajuste a porta local se precisar (ex.: 19081 -> 8080).
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=tenant-manager -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 19081:8080
```
Deixe em um terminal dedicado.

## Chamadas de smoke
### Health
```bash
curl -i http://127.0.0.1:19081/health
```
Esperado: `200` com status `healthy`.

### Metrics (opcional)
```bash
curl -i http://127.0.0.1:19081/metrics | head
```
Esperado: métricas Prometheus em texto.

### GET /api/v1/tenants
```bash
curl -i -sS http://127.0.0.1:19081/api/v1/tenants \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 5843292c-2b31-46ca-a6ba-c8e73a0dc2ed"
```
Esperado: `200` com lista de tenants (pode estar vazia). Se 401/403, revisar issuer/audience/claim de tenant e escopos.

## Logs
```bash
kubectl logs -n serphona deploy/tenant-manager --tail=50
```
Para o pod do port-forward: `kubectl logs -n serphona $POD --tail=50`.

## Limpeza
- Encerrar o port-forward (Ctrl+C ou `kill <pid>` se em background).
- Se nao precisar rodando, opcional escalar: `kubectl scale deploy/tenant-manager -n serphona --replicas=0`.
