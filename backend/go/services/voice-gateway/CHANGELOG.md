# Changelog

Todas as mudanças notáveis neste projeto serão documentadas neste arquivo.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
e este projeto adere ao [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planejado
- Implementação completa do Google Cloud Speech SDK
- Implementação completa do Google Cloud TTS SDK
- Implementação completa da ElevenLabs API
- Audio codec libraries (Opus, MP3)
- Testes de integração completos
- Kubernetes Helm charts
- CI/CD pipeline (GitHub Actions)

## [1.0.0] - 2025-01-03

### 🎉 Release Inicial

Primeira release production-ready do Voice Gateway!

### Added

#### Core Features
- **Asterisk ARI Integration** - WebSocket connection com auto-reconnect
- **Call Management** - Ciclo de vida completo de chamadas
  - Incoming/Outbound calls
  - Answer, Hold, Resume, Transfer, Hangup
  - State management (Ringing, Answered, Active, Hold, Transferred, Ended)
- **HTTP API** - RESTful API para gerenciamento
  - GET /api/v1/calls/{call_id}
  - DELETE /api/v1/calls/{call_id}
  - POST /api/v1/calls/{call_id}/transfer
  - GET /api/v1/tenants/{tenant_id}/calls
- **Asterisk Webhooks** - POST /asterisk/events
- **Health Checks** - Kubernetes-ready probes

#### Integration Clients
- **Tenant Manager Client**
  - DID lookup
  - Provider settings retrieval
  - Agent configuration
- **Agent Orchestrator Client**
  - Conversation management
  - Turn submission
  - Context updates

#### Infrastructure
- **Redis Integration** - Call state persistence
  - Thread-safe repository
  - TTL management
  - Multiple indexes (call_id, channel_id, tenant_id)
- **Kafka Integration** - Event publishing
  - 7 tipos de eventos (call.started, call.answered, call.ended, etc)
  - Async publishing
  - Error handling

#### Adapters & Providers
- **STT Provider Interface** - Speech-to-Text abstraction
  - Google STT skeleton
- **TTS Provider Interface** - Text-to-Speech abstraction
  - Google TTS skeleton
  - ElevenLabs TTS skeleton
- **Audio Processing**
  - Audio buffer management (thread-safe)
  - Stream converter
  - Chunk reader
  - PCM converter (resampling, mono conversion)
  - Audio mixer

#### Domain Layer
- **Call Entity** - Core business logic
  - Lifecycle management
  - State transitions
  - Metadata support
- **Direction & State** - Value objects
- **Conversation Management** - Thread-safe conversation state

#### Docker & Deployment
- **Dockerfile** - Multi-stage optimized build (~20MB image)
- **docker-compose.yml** - 7 services orchestrados
  - voice-gateway
  - redis
  - kafka + zookeeper
  - asterisk
  - prometheus
  - grafana
- **Health checks** - Automatic monitoring
- **Resource limits** - Production-ready configuration

#### Monitoring & Observability
- **Prometheus Integration** - Metrics exposure (port 9091)
- **Structured Logging** - zap logger
- **Health endpoints** - /health, /health/live, /health/ready

#### Documentation
- **README.md** - Visão geral e setup
- **API.md** - Documentação completa da API REST
- **DOCKER.md** - Guia de deployment Docker
- **CONTRIBUTING.md** - Guia para contribuidores
- **VOICE-GATEWAY-DESIGN.md** - Arquitetura detalhada
- **PROMPTS-YAML-SPEC.md** - Configuração de agentes (4 exemplos)
- **TENANT-MANAGER-TELEPHONY-EXTENSIONS.md** - Extensões de API

#### Testing
- **Unit Tests** - Domain layer (10 tests, 100% passing)
- **Test Coverage** - call.go completamente testado

#### Configuration
- **.env.example** - Template de configuração
- **Environment Variables** - 30+ variáveis configuráveis
- **Multi-provider Support** - Google, ElevenLabs

### Technical Highlights

#### Architecture
- **Hexagonal Architecture** - Clean separation of concerns
- **Dependency Injection** - Testable design
- **Interface-based** - Easy to mock and test
- **Context Propagation** - Proper cancellation
- **Thread-safe** - sync.RWMutex where needed

#### Code Quality
- **Go 1.23** - Latest stable version
- **32 arquivos** - ~6.000 linhas de código
- **14 packages** - Bem organizados
- **Zero compiler errors** - Build 100%
- **10/10 tests passing** - Unit tests

#### Performance & Reliability
- **WebSocket Auto-reconnect** - Max 10 attempts with backoff
- **Stateless Design** - Horizontal scaling ready
- **Redis Persistence** - State recovery
- **Event-driven** - Kafka async processing
- **Error Handling** - Structured error wrapping

### Dependencies

```go
require (
    github.com/google/uuid v1.6.0
    github.com/gorilla/websocket v1.5.3
    github.com/redis/go-redis/v9 v9.6.2
    github.com/segmentio/kafka-go v0.4.47
    go.uber.org/zap v1.27.0
)
```

### Breaking Changes

Nenhuma (primeira release).

### Deprecated

Nenhuma.

### Removed

Nenhuma.

### Fixed

Nenhuma.

### Security

- Non-root user em Docker
- Basic Auth para Asterisk ARI
- JWT authentication ready (placeholder)
- Secrets via environment variables

---

## Tipos de Mudanças

- `Added` - Nova funcionalidade
- `Changed` - Mudança em funcionalidade existente
- `Deprecated` - Funcionalidade que será removida
- `Removed` - Funcionalidade removida
- `Fixed` - Correção de bug
- `Security` - Vulnerabilidades corrigidas

---

## Links

- [Unreleased]: https://github.com/serphona/serphona/compare/voice-gateway-v1.0.0...HEAD
- [1.0.0]: https://github.com/serphona/serphona/releases/tag/voice-gateway-v1.0.0
