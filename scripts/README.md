# Scripts

Utilitários para o cluster local (kind/serphona).

## cluster-tools-setup.sh
- Instala/atualiza UIs de apoio: Kafka (Redpanda Console), Redis (redis-commander), ClickHouse (CloudBeaver). Opcionalmente inicia port-forwards.
- Variáveis:
  - `NAMESPACE` (default `serphona`)
  - `KAFKA_USER` / `KAFKA_PASSWORD` (default `user1` / `Cgs8eEw4WS`)
  - `REDIS_PASSWORD` (default `Y7chJlHeI1`)
  - `START_PORT_FORWARD` (`true|false`, default `false`)
- Uso:
  - `START_PORT_FORWARD=true NAMESPACE=serphona ./cluster-tools-setup.sh`
- Portas expostas (quando START_PORT_FORWARD=true):
  - Kafka UI: http://localhost:8080
  - Redis UI: http://localhost:8083
  - MinIO UI: http://localhost:9090 (creds do secret minio: access `minio`, secret `minio123`)
  - ClickHouse UI (CloudBeaver): http://localhost:8978

## cluster-tools-teardown.sh
- Encerra port-forwards (usa PIDs salvos em `/tmp/*-pf.pid`) e remove UIs auxiliares (rp-console, redis-commander, cloudbeaver, tabix se existir).
- Variáveis:
  - `NAMESPACE` (default `serphona`)
- Uso:
  - `NAMESPACE=serphona ./cluster-tools-teardown.sh`

## Dicas
- Para criar um tópico Kafka de teste:
  ```bash
  kubectl -n serphona exec kafka-controller-0 -- bash -c 'cat > /tmp/client.properties <<"EOF"
  security.protocol=SASL_PLAINTEXT
  sasl.mechanism=PLAIN
  sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username="user1" password="Cgs8eEw4WS";
  EOF
  /opt/bitnami/kafka/bin/kafka-topics.sh --bootstrap-server kafka-controller-0.kafka-controller-headless.serphona.svc.cluster.local:9092 --command-config /tmp/client.properties --create --topic usage.reported --partitions 1 --replication-factor 1
  '
  ```
- Conexão ClickHouse (CloudBeaver): host `clickhouse.serphona.svc.cluster.local`, porta 9000 (ou HTTP 8123), autenticação padrão conforme setup do chart.
