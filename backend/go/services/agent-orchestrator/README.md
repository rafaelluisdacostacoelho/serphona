# Agent Orchestrator Service

Enterprise-grade microservice for orchestrating conversational AI agents with LLM integration, tool calling, and comprehensive resilience patterns.

## 🚀 Features

### Core Capabilities
- **Session Management**: Redis-backed conversation sessions with automatic expiration
- **Agent Configuration**: PostgreSQL-based agent persistence with customizable prompts and models
- **LLM Integration**: OpenAI GPT-4/3.5-turbo support with streaming responses
- **Tool Calling**: Function calling integration with Tools Gateway
- **Multi-tenant**: Complete tenant isolation and management

### Resilience & Performance
- **Rate Limiting**: Token bucket algorithm (IP, tenant, user-based)
- **Circuit Breaker**: Automatic fault isolation for external dependencies
- **Graceful Degradation**: Fallback mechanisms for service failures
- **Health Checks**: Kubernetes-ready liveness and readiness probes

### Observability
- **Metrics**: Prometheus-ready instrumentation (HTTP, LLM, Tools, Tokens)
- **Tracing**: OpenTelemetry-ready distributed tracing
- **Logging**: Structured logging with request IDs
- **API Documentation**: OpenAPI 3.0 specification

### DevOps
- **Docker**: Multi-stage builds, optimized for size (~15MB)
- **Kubernetes**: Production-ready manifests with auto-scaling
- **CI/CD**: GitHub Actions pipeline (test, lint, build, deploy)
- **Service Mesh**: Istio integration ready

## 📋 Prerequisites

- Go 1.21+
- Redis 7+
- PostgreSQL 14+ (optional, for agent persistence)
- Docker & Kubernetes (for deployment)
- OpenAI API key

## 🛠️ Installation

### Local Development

```bash
# Clone repository
git clone https://github.com/rafaelluisdacostacoelho/serphona.git
cd serphona/backend/go/services/agent-orchestrator

# Install dependencies
go mod download

# Set environment variables
export REDIS_ADDR=localhost:6379
export DATABASE_URL=postgresql://user:pass@localhost:5432/agent_orchestrator
export OPENAI_API_KEY=sk-your-api-key
export TOOLS_GATEWAY_URL=http://localhost:8085

# Run service
go run cmd/server/main.go
```

### Docker

```bash
# Build image
docker build -t serphona/agent-orchestrator:latest .

# Run container
docker run -p 8080:8080 \
  -e REDIS_ADDR=redis:6379 \
  -e OPENAI_API_KEY=sk-your-api-key \
  serphona/agent-orchestrator:latest
```

### Kubernetes

```bash
# Create namespace
kubectl create namespace serphona

# Apply secrets (edit values first)
cp k8s/secret.example.yaml k8s/secret.yaml
kubectl apply -f k8s/secret.yaml

# Deploy service
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

# Check status
kubectl get pods -n serphona
kubectl logs -f deployment/agent-orchestrator -n serphona
```

## 📚 API Documentation

Full OpenAPI 3.0 specification available at `api/openapi.yaml`.

### Endpoints

#### Sessions
- `POST /api/v1/sessions` - Create new session
- `GET /api/v1/sessions/:id` - Get session
- `DELETE /api/v1/sessions/:id` - End session
- `POST /api/v1/sessions/:id/messages` - Send message
- `GET /api/v1/sessions/:id/messages` - Get messages

#### Agents
- `POST /api/v1/agents` - Create agent
- `GET /api/v1/agents` - List agents
- `GET /api/v1/agents/:id` - Get agent
- `PUT /api/v1/agents/:id` - Update agent
- `DELETE /api/v1/agents/:id` - Delete agent
- `POST /api/v1/agents/:id/activate` - Activate agent
- `POST /api/v1/agents/:id/deactivate` - Deactivate agent

#### Health
- `GET /health` - Health check
- `GET /ready` - Readiness check

### Example: Create Session and Send Message

```bash
# Create session
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "660e8400-e29b-41d4-a716-446655440001",
    "channel_type": "text"
  }'

# Send message
curl -X POST http://localhost:8080/api/v1/sessions/{session_id}/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Find me flights from NYC to LAX tomorrow",
    "user_id": "660e8400-e29b-41d4-a716-446655440001"
  }'
```

## 🏗️ Architecture

```
agent-orchestrator/
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── adapter/
│   │   └── http/        # HTTP handlers
│   ├── domain/          # Domain models & interfaces
│   │   ├── model/
│   │   └── repository/
│   ├── infrastructure/  # External integrations
│   │   ├── events/      # Kafka publisher
│   │   ├── http/        # Tools Gateway client
│   │   ├── llm/         # OpenAI client pool
│   │   ├── metrics/     # Prometheus metrics
│   │   ├── middleware/  # Rate limiter
│   │   ├── repository/  # Redis & PostgreSQL
│   │   ├── resilience/  # Circuit breaker
│   │   └── tracing/     # OpenTelemetry tracer
│   └── usecase/         # Business logic
├── api/                 # OpenAPI specification
├── k8s/                 # Kubernetes manifests
└── Dockerfile
```

## 🔧 Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_ADDR` | HTTP server address | `:8080` |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | - |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `OPENAI_API_KEY` | OpenAI API key | **required** |
| `TOOLS_GATEWAY_URL` | Tools Gateway URL | - |
| `GIN_MODE` | Gin mode (debug/release) | `debug` |

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -v ./internal/usecase/...
```

## 📈 Monitoring

### Metrics (Prometheus)
When Prometheus library is integrated:
- `agent_orchestrator_http_requests_total`
- `agent_orchestrator_llm_calls_total`
- `agent_orchestrator_tokens_used_total`
- `agent_orchestrator_sessions_created_total`

### Tracing (Jaeger)
Distributed tracing spans:
- `ProcessMessage`
- `LLMCall`
- `ToolExecution`

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## 📄 License

This project is part of the Serphona platform.

## 🔗 Links

- **Repository**: https://github.com/rafaelluisdacostacoelho/serphona
- **Documentation**: [/docs](../../docs/)
- **API Spec**: [/api/openapi.yaml](api/openapi.yaml)
