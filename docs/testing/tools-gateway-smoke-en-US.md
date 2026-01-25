# Tools Gateway smoke test (local kind)

## Prereqs
- kind cluster `serphona-kind` with context pointed to namespace `serphona`.
- Core deps installed via Helm (Kafka, Postgres, Redis, MinIO, ClickHouse) as per repo docs.
- Image `serphona/tools-gateway:dev` loaded into the cluster.
- Secrets present: `tools-gateway-auth` (HS256 key `dev-jwt-secret`), `tools-gateway-db` (Postgres URL), `tools-gateway-secrets` (redis-url), `kafka-user-passwords` (for Kafka path if re-enabled).

## Chart config
To run the smoke without Kafka, ensure `RAG_INGEST_KAFKA_ENABLED=false` in the override:
```bash
helm upgrade --install tools-gateway infra/helm/tools-gateway \
  -n serphona \
  -f infra/helm/overrides/tools-gateway-values.yaml \
  --set global.imageRegistry=serphona \
  --set image.tag=dev \
  --set serviceMonitor.enabled=false \
  --set prometheusRule.enabled=false \
  --set grafanaDashboard.enabled=false
kubectl rollout status deploy/tools-gateway -n serphona
```
If you need to test Kafka publishing, set `RAG_INGEST_KAFKA_ENABLED=true` and switch to SCRAM (user `user1`, password from `kafka-user-passwords`, SCRAM mechanism in the publisher).

## Generate HS256 token (dev)
```bash
python3 - <<'PY'
import jwt, time
secret='dev-jwt-secret'
now=int(time.time())
payload={
    'service':'tools-gateway-smoke',
    'tenantId':'5843292c-2b31-46ca-a6ba-c8e73a0dc2ed',
    'scopes':['tools:read','tools:write','tools:execute'],
    'iss':'serphona',
    'aud':'serphona-services',
    'iat':now,
    'exp':now+3600,
}
print(jwt.encode(payload, secret, algorithm='HS256'))
PY
```
Store it in `TOKEN`.

## Port-forward
Prefer a concrete pod to avoid restarts affecting the tunnel:
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=tools-gateway -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 19080:8080
```
Keep this running in a separate terminal.

## Smoke calls
### Health
```bash
curl -i http://127.0.0.1:19080/health
```
Expected: `200` with `healthy`.

### RAG ingestion (Kafka off)
```bash
curl -i -sS \
  -X POST http://127.0.0.1:19080/api/v1/rag/ingestions/rest \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 5843292c-2b31-46ca-a6ba-c8e73a0dc2ed" \
  -H "Content-Type: application/json" \
  --data '{"namespace":"demo","document_id":"doc-1","uri":"https://example.com","source":"smoke-test","metadata":{"foo":"bar"}}'
```
Expected: `202 Accepted` with `{"document_id":"doc-1","status":"queued"}` (publish is skipped because Kafka is disabled).

### /api/v1/tools (optional)
```bash
curl -i -sS http://127.0.0.1:19080/api/v1/tools \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: platform"
```
Expected: `200` (empty list if no tools exist).

## Logs
```bash
kubectl logs -n serphona deploy/tools-gateway --tail=50
```
For the specific forwarded pod: `kubectl logs -n serphona $POD --tail=50`.

## Cleanup
- Stop the port-forward (Ctrl+C or `kill <pid>` if backgrounded).
- Keep the deployment running or scale down if not needed: `kubectl scale deploy/tools-gateway -n serphona --replicas=0`.
