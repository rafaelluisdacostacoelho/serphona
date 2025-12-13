# Serphona - Docker Compose

Docker Compose para desenvolvimento e testes locais da plataforma Serphona completa.

## 🚀 Quick Start

```bash
# Subir toda a infraestrutura e serviços
docker-compose up -d

# Ver logs
docker-compose logs -f

# Parar tudo
docker-compose down

# Parar e remover volumes (reset completo)
docker-compose down -v
```

## 📦 Serviços Incluídos

### Infraestrutura
- **PostgreSQL** (5432) - Database principal
- **Redis** (6379) - Cache e sessões
- **Kafka + Zookeeper** (9092) - Event streaming
- **Asterisk** (8088, 5060) - Telephony (comentado por padrão)

### Backend Go
- **tenant-manager** (8081) - Gestão de tenants e configurações
- **auth-gateway** (8082) - Autenticação e autorização
- **agent-orchestrator** (8083) - Orquestração de agentes IA
- **tools-gateway** (8084) - Gestão de ferramentas/tools
- **voice-gateway** (8085) - Gateway de voz (comentado - requer Asterisk)
- **billing-service** (8086) - Serviço de billing

### Backend Python
- **analytics-processor** - Processamento de analytics

### Frontend
- **console** (3000) - Aplicação principal
- **auth-mfe** (3001) - Micro-frontend de autenticação
- **billing-mfe** (3002) - Micro-frontend de billing
- **website** (3003) - Landing page

## 🔧 Comandos Úteis

### Gerenciar serviços individuais
```bash
# Subir apenas infraestrutura
docker-compose up -d postgres redis kafka zookeeper

# Subir um serviço específico
docker-compose up -d tenant-manager

# Ver logs de um serviço
docker-compose logs -f tenant-manager

# Reiniciar um serviço
docker-compose restart tenant-manager

# Parar um serviço
docker-compose stop tenant-manager
```

### Build e rebuild
```bash
# Build de todos os serviços
docker-compose build

# Build de um serviço específico
docker-compose build tenant-manager

# Rebuild forçado
docker-compose build --no-cache tenant-manager

# Up com rebuild
docker-compose up -d --build
```

### Limpeza
```bash
# Remover containers parados
docker-compose rm

# Limpar volumes (CUIDADO: apaga dados)
docker-compose down -v

# Limpar tudo incluindo imagens
docker-compose down -v --rmi all
```

## 🔍 Verificação de Saúde

### Verificar status
```bash
# Ver status de todos containers
docker-compose ps

# Ver uso de recursos
docker stats $(docker-compose ps -q)
```

### Health Checks
```bash
# PostgreSQL
docker-compose exec postgres pg_isready -U serphona

# Redis
docker-compose exec redis redis-cli ping

# Kafka
docker-compose exec kafka kafka-broker-api-versions --bootstrap-server=localhost:9092
```

### APIs
```bash
# Tenant Manager
curl http://localhost:8081/health

# Auth Gateway  
curl http://localhost:8082/health

# Agent Orchestrator
curl http://localhost:8083/health

# Tools Gateway
curl http://localhost:8084/health

# Billing Service
curl http://localhost:8086/health
```

## 🎯 Cenários Comuns

### Desenvolvimento Backend
```bash
# Subir apenas infraestrutura
docker-compose up -d postgres redis kafka zookeeper

# Rodar serviço Go localmente (fora do container)
cd backend/go/services/tenant-manager
go run cmd/server/main.go
```

### Desenvolvimento Frontend
```bash
# Subir backend completo
docker-compose up -d postgres redis kafka tenant-manager auth-gateway

# Rodar frontend localmente
cd frontend/console
npm run dev
```

### Testes de Integração
```bash
# Subir tudo
docker-compose up -d

# Aguardar containers ficarem healthy
sleep 30

# Executar testes
make test-integration
```

### Voice Gateway (com Asterisk)
```bash
# 1. Descomentar asterisk e voice-gateway no docker-compose.yml

# 2. Configurar Asterisk (se necessário)
mkdir -p asterisk/config
# Adicionar arquivos de configuração

# 3. Subir
docker-compose up -d asterisk voice-gateway
```

## 🐛 Troubleshooting

### Container não sobe
```bash
# Ver logs detalhados
docker-compose logs <service-name>

# Ver últimas 100 linhas
docker-compose logs --tail=100 <service-name>

# Seguir logs em tempo real
docker-compose logs -f <service-name>
```

### Problemas de conectividade
```bash
# Entrar no container
docker-compose exec <service-name> sh

# Testar DNS
nslookup postgres
nslookup redis

# Testar conectividade
curl http://tenant-manager:8080/health
```

### Problemas de porta
```bash
# Ver portas em uso
docker-compose ps

# Verificar conflitos no host
netstat -tulpn | grep LISTEN
```

### Reset completo
```bash
# Parar tudo
docker-compose down -v

# Limpar imagens antigas
docker-compose down -v --rmi all

# Rebuild completo
docker-compose build --no-cache

# Subir novamente
docker-compose up -d
```

## 📝 Variáveis de Ambiente

Você pode sobrescrever variáveis criando um `.env`:

```bash
# .env
POSTGRES_PASSWORD=minha-senha-segura
REDIS_PASSWORD=outra-senha
KAFKA_AUTO_CREATE_TOPICS=false
```

## 🔐 Segurança

⚠️ **ATENÇÃO**: Este docker-compose é para DESENVOLVIMENTO/TESTES apenas!

Para produção:
- Use senhas fortes
- Não exponha portas desnecessárias
- Configure TLS/SSL
- Use secrets management (Vault, etc.)
- Implemente network policies
- Use Kubernetes (veja pasta `k8s/`)

## 📚 Referências

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Serphona Architecture](./docs/architecture/README.md)
- [Kubernetes Deployment](./backend/go/services/voice-gateway/k8s/README.md)
