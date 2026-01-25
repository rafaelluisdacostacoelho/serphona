# Smoke test do Tools Gateway (local kind)

## Premissas
- Cluster kind `serphona-kind` ativo e contexto kubectl apontando para o namespace `serphona`.
- Dependencias basicas instaladas via Helm (Kafka, Postgres, Redis, MinIO, ClickHouse) conforme instrucoes do repo.
- Imagem `serphona/tools-gateway:dev` ja carregada no cluster.
- Secrets presentes: `tools-gateway-auth` (chave HS256 `dev-jwt-secret`), `tools-gateway-db` (url de Postgres), `tools-gateway-secrets` (redis-url), `kafka-user-passwords` (para testes com Kafka se reativar).

## Configurar o chart
Para rodar o smoke sem Kafka, garanta `RAG_INGEST_KAFKA_ENABLED=false` no override:

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

Se precisar testar o caminho Kafka, volte `RAG_INGEST_KAFKA_ENABLED=true` e ajuste para SCRAM (usuario `user1`, senha em `kafka-user-passwords`, mecanismo SCRAM).

## Gerar token HS256 (dev)
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
Guarde o token em `TOKEN`.

## Port-forward
Use um pod especifico para evitar queda de conexao:
```bash
POD=$(kubectl get pods -n serphona -l app.kubernetes.io/name=tools-gateway -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n serphona pod/$POD 19080:8080
```
Deixe em um terminal separado.

## Chamadas de smoke
### Health
```bash
curl -i http://127.0.0.1:19080/health
```
Esperado: `200` com status `healthy`.

### RAG ingestion (Kafka off)
```bash
curl -i -sS \
  -X POST http://127.0.0.1:19080/api/v1/rag/ingestions/rest \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 5843292c-2b31-46ca-a6ba-c8e73a0dc2ed" \
  -H "Content-Type: application/json" \
  --data '{"namespace":"demo","document_id":"doc-1","uri":"https://example.com","source":"smoke-test","metadata":{"foo":"bar"}}'
```
Esperado: `202 Accepted` com corpo `{"document_id":"doc-1","status":"queued"}` (publicacao pulada porque Kafka desabilitado).

### /api/v1/tools (opcional)
```bash
curl -i -sS http://127.0.0.1:19080/api/v1/tools \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: platform"
```
Esperado: `200` (lista vazia se nao houver tools).

## Logs
```bash
kubectl logs -n serphona deploy/tools-gateway --tail=50
```
Para ver apenas o pod do port-forward: `kubectl logs -n serphona $POD --tail=50`.

## Limpeza
- Encerrar o port-forward (Ctrl+C ou `kill <pid>` se em background).
- Manter deployment ativo ou escalar para zero se nao precisar: `kubectl scale deploy/tools-gateway -n serphona --replicas=0`.
