#!/usr/bin/env bash
set -euo pipefail

# Brings up dev UIs for Kafka (Redpanda Console), Redis (redis-commander), and ClickHouse (CloudBeaver).
# Defaults target the serphona kind cluster.

NAMESPACE="${NAMESPACE:-serphona}"
KAFKA_USER="${KAFKA_USER:-user1}"
KAFKA_PASSWORD="${KAFKA_PASSWORD:-Cgs8eEw4WS}"
REDIS_PASSWORD="${REDIS_PASSWORD:-Y7chJlHeI1}"
START_PORT_FORWARD="${START_PORT_FORWARD:-false}"

# Headless Kafka broker DNS entries (Bitnami chart controllers)
KAFKA_BROKERS=(
  "kafka-controller-0.kafka-controller-headless.${NAMESPACE}.svc.cluster.local:9092"
  "kafka-controller-1.kafka-controller-headless.${NAMESPACE}.svc.cluster.local:9092"
  "kafka-controller-2.kafka-controller-headless.${NAMESPACE}.svc.cluster.local:9092"
)

require() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing dependency: $1" >&2; exit 1; }
}

require kubectl
require helm

start_pf() {
  local name="$1" svc="$2" local_port="$3" remote_port="$4"
  echo "Starting port-forward for ${name} on localhost:${local_port} -> ${svc}:${remote_port}"
  kubectl -n "$NAMESPACE" port-forward "$svc" "${local_port}:${remote_port}" >/tmp/${name}-pf.log 2>&1 &
  echo $! > "/tmp/${name}-pf.pid"
}

# Redpanda Console for Kafka
helm repo add redpanda https://charts.redpanda.com >/dev/null 2>&1 || true
helm upgrade --install rp-console redpanda/console -n "$NAMESPACE" \
  --set config.kafka.brokers[0]="${KAFKA_BROKERS[0]}" \
  --set config.kafka.brokers[1]="${KAFKA_BROKERS[1]}" \
  --set config.kafka.brokers[2]="${KAFKA_BROKERS[2]}" \
  --set config.kafka.sasl.enabled=true \
  --set config.kafka.sasl.mechanism=PLAIN \
  --set config.kafka.sasl.username="$KAFKA_USER" \
  --set config.kafka.sasl.password="$KAFKA_PASSWORD" \
  --set config.kafka.tls.enabled=false \
  --set service.type=ClusterIP

# Redis Commander UI
cat <<EOF | kubectl apply -n "$NAMESPACE" -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis-commander
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis-commander
  template:
    metadata:
      labels:
        app: redis-commander
    spec:
      containers:
      - name: redis-commander
        image: rediscommander/redis-commander:latest
        env:
        - name: REDIS_HOSTS
          value: local:redis-master.serphona.svc.cluster.local:6379:0:${REDIS_PASSWORD}
        ports:
        - containerPort: 8081
        resources:
          requests:
            cpu: 50m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 256Mi
---
apiVersion: v1
kind: Service
metadata:
  name: redis-commander
spec:
  type: ClusterIP
  selector:
    app: redis-commander
  ports:
  - port: 8081
    targetPort: 8081
    name: http
EOF

# CloudBeaver UI for ClickHouse
cat <<EOF | kubectl apply -n "$NAMESPACE" -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cloudbeaver
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cloudbeaver
  template:
    metadata:
      labels:
        app: cloudbeaver
    spec:
      containers:
      - name: cloudbeaver
        image: dbeaver/cloudbeaver:23.3.4
        ports:
        - containerPort: 8978
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 300m
            memory: 512Mi
---
apiVersion: v1
kind: Service
metadata:
  name: cloudbeaver
spec:
  type: ClusterIP
  selector:
    app: cloudbeaver
  ports:
  - port: 8978
    targetPort: 8978
    name: http
EOF

echo "\nReady to port-forward:"
echo "Kafka UI:   kubectl -n ${NAMESPACE} port-forward svc/rp-console-console 8080:8080"
echo "Redis UI:   kubectl -n ${NAMESPACE} port-forward svc/redis-commander 8083:8081"
echo "MinIO UI:   kubectl -n ${NAMESPACE} port-forward svc/minio-console 9090:9090"
echo "ClickHouse: kubectl -n ${NAMESPACE} port-forward svc/cloudbeaver 8978:8978"

if [ "$START_PORT_FORWARD" = "true" ]; then
  start_pf kafka-ui svc/rp-console-console 8080 8080
  start_pf redis-ui svc/redis-commander 8083 8081
  start_pf minio-ui svc/minio-console 9090 9090
  start_pf clickhouse-ui svc/cloudbeaver 8978 8978
  echo "Port-forwards started in background. Logs under /tmp/*-pf.log; PIDs in /tmp/*-pf.pid."
fi
