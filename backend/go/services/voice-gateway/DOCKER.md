# Voice Gateway - Docker Deployment Guide

Guia completo para deploy do Voice Gateway usando Docker e Docker Compose.

## 📦 Componentes

O `docker-compose.yml` inclui todos os serviços necessários:

| Serviço | Porta | Descrição |
|---------|-------|-----------|
| **voice-gateway** | 8080, 9091 | Aplicação principal |
| **redis** | 6379 | State management |
| **kafka** | 9092 | Event streaming |
| **zookeeper** | 2181 | Kafka coordination |
| **asterisk** | 8088, 5060 | Telefonia (ARI + SIP) |
| **prometheus** | 9090 | Monitoring |
| **grafana** | 3000 | Dashboards |

## 🚀 Quick Start

### 1. Build e Start

```bash
# Build e iniciar todos os serviços
docker-compose up --build -d

# Ver logs
docker-compose logs -f voice-gateway

# Ver status
docker-compose ps
```

### 2. Verificar Health

```bash
# Voice Gateway
curl http://localhost:8080/health

# Redis
docker exec voice-gateway-redis redis-cli ping

# Kafka topics
docker exec voice-gateway-kafka kafka-topics --list --bootstrap-server localhost:9092
```

### 3. Stop

```bash
# Parar serviços
docker-compose down

# Parar e remover volumes (limpa dados)
docker-compose down -v
```

## ⚙️ Configuração

### Variáveis de Ambiente

Edite `docker-compose.yml` para ajustar:

```yaml
environment:
  # Asterisk ARI
  - ASTERISK_ARI_URL=http://asterisk:8088/ari
  - ASTERISK_ARI_USERNAME=serphona
  - ASTERISK_ARI_PASSWORD=your_secure_password
  
  # Redis
  - REDIS_URL=redis://redis:6379
  
  # Kafka
  - KAFKA_BROKERS=kafka:9092
  
  # External services
  - TENANT_MANAGER_URL=http://tenant-manager:8081
  - AGENT_ORCHESTRATOR_URL=http://agent-orchestrator:8082
```

### Asterisk Configuration

Crie o diretório `asterisk-config/` com:

**ari.conf**
```ini
[general]
enabled = yes
pretty = yes

[serphona]
type = user
read_only = no
password = your_secure_password
```

**http.conf**
```ini
[general]
enabled = yes
bindaddr = 0.0.0.0
bindport = 8088
```

**extensions.conf**
```ini
[from-trunk]
exten => _X.,1,NoOp(Incoming call to ${EXTEN})
same => n,Stasis(serphona,${EXTEN})
same => n,Hangup()
```

## 🔍 Monitoring

### Prometheus

Acesse: http://localhost:9090

Queries úteis:
```promql
# Total de chamadas
voice_gateway_calls_total

# Chamadas ativas
voice_gateway_calls_active

# Latência STT
rate(voice_gateway_stt_latency_seconds_sum[5m])
```

### Grafana

1. Acesse: http://localhost:3000
2. Login: `admin` / `admin`
3. Add Prometheus datasource: `http://prometheus:9090`
4. Crie dashboards com métricas do voice-gateway

### Logs

```bash
# Voice Gateway logs
docker-compose logs -f voice-gateway

# Todos os serviços
docker-compose logs -f

# Com timestamps
docker-compose logs -f --timestamps

# Últimas 100 linhas
docker-compose logs --tail=100 voice-gateway
```

## 🧪 Testing

### Test ARI Connection

```bash
# Test Asterisk ARI endpoint
curl -u serphona:your_password http://localhost:8088/ari/asterisk/info

# Test voice-gateway API
curl http://localhost:8080/health
curl http://localhost:8080/health/ready
```

### Test Redis

```bash
# Connect to Redis CLI
docker exec -it voice-gateway-redis redis-cli

# Inside redis-cli:
> PING
> KEYS *
> INFO
```

### Test Kafka

```bash
# List topics
docker exec voice-gateway-kafka kafka-topics \
  --list \
  --bootstrap-server localhost:9092

# Consume events
docker exec voice-gateway-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic voice-gateway.call.started \
  --from-beginning
```

## 📊 Production Deployment

### 1. Build para Produção

```bash
# Build image
docker build -t serphona/voice-gateway:1.0.0 .

# Tag para registry
docker tag serphona/voice-gateway:1.0.0 your-registry.com/voice-gateway:1.0.0

# Push para registry
docker push your-registry.com/voice-gateway:1.0.0
```

### 2. Configurar Secrets

Use Docker secrets ou Kubernetes secrets para credenciais:

```bash
# Exemplo com Docker Swarm
echo "your_password" | docker secret create asterisk_password -
```

### 3. Deploy com Docker Swarm

```bash
# Initialize swarm
docker swarm init

# Deploy stack
docker stack deploy -c docker-compose.yml voice-gateway

# Check services
docker service ls
docker service logs voice-gateway_voice-gateway
```

### 4. Health Checks

Health checks já estão configurados no Dockerfile:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
```

## 🐛 Troubleshooting

### Voice Gateway não inicia

```bash
# Ver logs detalhados
docker-compose logs voice-gateway

# Verificar variáveis de ambiente
docker-compose config

# Rebuild sem cache
docker-compose build --no-cache voice-gateway
```

### Asterisk não conecta

```bash
# Verificar se Asterisk está rodando
docker-compose ps asterisk

# Ver logs do Asterisk
docker-compose logs asterisk

# Testar ARI endpoint
curl -u username:password http://localhost:8088/ari/asterisk/info
```

### Redis connection failed

```bash
# Verificar se Redis está rodando
docker-compose ps redis

# Test connection
docker exec voice-gateway-redis redis-cli ping

# Ver logs
docker-compose logs redis
```

### Kafka connection failed

```bash
# Verificar se Kafka está rodando
docker-compose ps kafka zookeeper

# Ver logs
docker-compose logs kafka

# Listar topics
docker exec voice-gateway-kafka kafka-topics --list --bootstrap-server localhost:9092
```

## 🔐 Security Best Practices

### 1. Não commitar secrets

Adicione ao `.gitignore`:
```
.env
.env.local
asterisk-config/
prometheus-data/
grafana-data/
```

### 2. Use secrets management

- Docker Secrets (Swarm)
- Kubernetes Secrets
- HashiCorp Vault
- AWS Secrets Manager

### 3. Network segmentation

```yaml
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true
```

### 4. Limite recursos

```yaml
services:
  voice-gateway:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

## 📈 Scaling

### Horizontal Scaling

```bash
# Scale voice-gateway instances
docker-compose up -d --scale voice-gateway=3

# Com Docker Swarm
docker service scale voice-gateway_voice-gateway=3
```

### Load Balancing

Use nginx ou HAProxy como load balancer:

```nginx
upstream voice_gateway {
    server voice-gateway-1:8080;
    server voice-gateway-2:8080;
    server voice-gateway-3:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://voice_gateway;
    }
}
```

## 🔄 Updates

### Rolling Update

```bash
# Pull nova imagem
docker-compose pull voice-gateway

# Restart com zero-downtime (Swarm)
docker service update --image your-registry.com/voice-gateway:1.0.1 voice-gateway

# Ou com docker-compose
docker-compose up -d voice-gateway
```

## 📝 Maintenance

### Backup

```bash
# Backup Redis data
docker exec voice-gateway-redis redis-cli SAVE
docker cp voice-gateway-redis:/data/dump.rdb ./backups/

# Backup Prometheus data
docker cp voice-gateway-prometheus:/prometheus ./backups/prometheus-$(date +%Y%m%d)
```

### Restore

```bash
# Restore Redis
docker cp ./backups/dump.rdb voice-gateway-redis:/data/
docker-compose restart redis
```

### Cleanup

```bash
# Remove unused images
docker image prune -a

# Remove unused volumes
docker volume prune

# Remove everything stopped
docker system prune -a
```

## 📚 Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Asterisk ARI Documentation](https://wiki.asterisk.org/wiki/display/AST/Asterisk+REST+Interface)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)

## 🆘 Support

Para issues ou dúvidas:
1. Verifique logs: `docker-compose logs`
2. Verifique health: `curl http://localhost:8080/health`
3. Consulte troubleshooting acima
4. Abra issue no repositório

---

**Status**: Production-Ready ✅  
**Last Updated**: 2025-01-03
