# Agent Orchestrator - Planning Document

> 🤖 Orquestrador central de agentes LLM com roteamento inteligente e gerenciamento de contexto

## Índice

1. [Visão Geral](#visão-geral)
2. [Arquitetura](#arquitetura)
3. [Domínio](#domínio)
4. [Casos de Uso](#casos-de-uso)
5. [Integrações](#integrações)
6. [Implementação](#implementação)
7. [Roadmap](#roadmap)

---

## Visão Geral

### Propósito

O **Agent Orchestrator** é o cérebro da plataforma Serphona, responsável por:

- 🎯 **Roteamento Inteligente**: Direcionar mensagens para o agente mais apropriado
- 🔄 **Delegação**: Permitir que agentes deleguem tarefas entre si
- 💬 **Sessões**: Gerenciar contexto de conversação multi-turn
- 🧠 **Context Management**: Manter histórico e estado em Redis
- 🔧 **Tool Integration**: Orquestrar chamadas ao Tools Gateway
- 📊 **Event Publishing**: Emitir eventos para analytics via Kafka

### Contexto no Sistema

```
┌────────────────────────────────────────────────────────────┐
│                     Serphona Platform                      │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  ┌─────────────┐      ┌──────────────────┐                 │
│  │   Voice     │─────>│     Agent        │                 │
│  │   Gateway   │      │  Orchestrator    │                 │
│  └─────────────┘      └──────────────────┘                 │
│                              │                             │
│                              ├────────────┐                │
│                              │            │                │
│                              ▼            ▼                │
│                       ┌───────────┐ ┌─────────────┐        │
│                       │   Tools   │ │   OpenAI/   │        │
│                       │  Gateway  │ │  Anthropic  │        │
│                       └───────────┘ └─────────────┘        │
│                              │                             │
│                              ▼                             │
│                       ┌──────────────┐                     │
│                       │   Analytics  │                     │
│                       │  (Kafka)     │                     │
│                       └──────────────┘                     │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### Responsabilidades Principais

1. **Session Management**
   - Criar/gerenciar sessões de conversação
   - Manter contexto multi-turn
   - Persistir histórico em Redis
   - TTL configurável

2. **Agent Routing**
   - Analisar intent da mensagem
   - Selecionar agente apropriado
   - Fallback para agente genérico
   - Load balancing entre modelos

3. **Tool Orchestration**
   - Identificar quando ferramentas são necessárias
   - Executar tools via Tools Gateway
   - Processar resultados de ferramentas
   - Retry logic em falhas

4. **Delegation**
   - Permitir agentes delegarem tarefas
   - Manter chain of delegation
   - Timeout em delegações longas

5. **Event Publishing**
   - Publicar eventos de execução
   - Métricas de latência
   - Consumo de tokens
   - Erros e timeouts

---

## Arquitetura

### Clean Architecture (4 Camadas)

```
┌────────────────────────────────────────────────┐
│            Presentation Layer                  │
│  (HTTP Handlers, WebSocket, DTOs)              │
├────────────────────────────────────────────────┤
│            Use Case Layer                      │
│  (Session, Agent Routing, Tool Orchestration)  │
├────────────────────────────────────────────────┤
│            Domain Layer                        │
│  (Entities, Repository Interfaces)             │
├────────────────────────────────────────────────┤
│            Infrastructure Layer                │
│  (Redis, LLM Clients, Kafka, Tools Gateway)    │
└────────────────────────────────────────────────┘
```

### Componentes Principais

#### 1. Session Manager
- Criar sessões
- Armazenar contexto
- Recuperar histórico
- Limpar sessões expiradas

#### 2. Agent Router
- Intent recognition
- Agent selection
- Load balancing
- Fallback logic

#### 3. Tool Orchestrator
- Function calling detection
- Tool execution via gateway
- Result formatting
- Error handling

#### 4. Message Processor
- Processar mensagens
- Manter contexto
- Chamar agentes
- Retornar respostas

#### 5. LLM Client Pool
- OpenAI client
- Anthropic client
- Gemini client (futuro)
- Rate limiting

---

## Domínio

### Entities

#### Session
```go
type Session struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    UserID      uuid.UUID
    ChannelType string   // voice, text, api
    ChannelID   string   // phone number, chat id, etc
    Status      string   // active, ended, timeout
    Context     SessionContext
    CreatedAt   time.Time
    UpdatedAt   time.Time
    ExpiresAt   time.Time
}

type SessionContext struct {
    Messages    []Message
    Variables   map[string]interface{}
    CurrentAgent string
    DelegationChain []string
}
```

#### Message
```go
type Message struct {
    ID          uuid.UUID
    SessionID   uuid.UUID
    Role        string   // user, assistant, system, tool
    Content     string
    ToolCalls   []ToolCall
    Metadata    MessageMetadata
    CreatedAt   time.Time
}

type ToolCall struct {
    ID          string
    ToolName    string
    Arguments   json.RawMessage
    Result      json.RawMessage
    Status      string
    Error       string
}

type MessageMetadata struct {
    Model       string
    Tokens      int
    LatencyMS   int64
    Agent       string
    Cost        float64
}
```

#### Agent
```go
type Agent struct {
    ID            uuid.UUID
    Name          string
    DisplayName   string
    Description   string
    SystemPrompt  string
    Model         string   // gpt-4, claude-3, etc
    Temperature   float64
    MaxTokens     int
    Tools         []string // tool names from tools-gateway
    IsActive      bool
    CreatedAt     time.Time
}
```

#### AgentExecution
```go
type AgentExecution struct {
    ID            uuid.UUID
    SessionID     uuid.UUID
    AgentID       uuid.UUID
    MessageID     uuid.UUID
    InputTokens   int
    OutputTokens  int
    TotalTokens   int
    LatencyMS     int64
    Cost          float64
    Status        string   // success, error, timeout
    ErrorMessage  string
    ExecutedAt    time.Time
}
```

### Repository Interfaces

#### SessionRepository
```go
type SessionRepository interface {
    Create(ctx, *Session) error
    FindByID(ctx, id) (*Session, error)
    Update(ctx, *Session) error
    Delete(ctx, id) error
    FindByTenant(ctx, tenantID) ([]*Session, error)
    FindActive(ctx) ([]*Session, error)
}
```

#### AgentRepository
```go
type AgentRepository interface {
    Create(ctx, *Agent) error
    FindByID(ctx, id) (*Agent, error)
    FindByName(ctx, name) (*Agent, error)
    FindAll(ctx) ([]*Agent, error)
    Update(ctx, *Agent) error
    Delete(ctx, id) error
}
```

#### MessageRepository
```go
type MessageRepository interface {
    Create(ctx, *Message) error
    FindBySessionID(ctx, sessionID, limit, offset) ([]*Message, error)
    Count(ctx, sessionID) (int64, error)
}
```

---

## Casos de Uso

### 1. SessionService

**Responsabilidades**:
- Gerenciar ciclo de vida de sessões
- Manter contexto no Redis
- Implementar TTL
- Limpeza de sessões expiradas

**Métodos**:
```go
type SessionService interface {
    CreateSession(ctx, tenantID, userID, channelType) (*Session, error)
    GetSession(ctx, sessionID) (*Session, error)
    UpdateSession(ctx, *Session) error
    EndSession(ctx, sessionID) error
    AddMessage(ctx, sessionID, *Message) error
    GetMessages(ctx, sessionID, limit, offset) ([]*Message, error)
}
```

### 2. AgentRoutingService

**Responsabilidades**:
- Analisar mensagem e determinar intent
- Selecionar agente apropriado
- Load balancing entre modelos
- Fallback logic

**Métodos**:
```go
type AgentRoutingService interface {
    RouteMessage(ctx, *Message, *Session) (*Agent, error)
    SelectAgent(ctx, intent string) (*Agent, error)
    GetDefaultAgent(ctx) (*Agent, error)
}
```

### 3. MessageProcessingService

**Responsabilidades**:
- Processar mensagens end-to-end
- Chamar LLM via client pool
- Detectar e executar tool calls
- Manter contexto de sessão

**Métodos**:
```go
type MessageProcessingService interface {
    ProcessMessage(ctx, *ProcessMessageRequest) (*ProcessMessageResponse, error)
    StreamMessage(ctx, *ProcessMessageRequest) (<-chan MessageChunk, error)
}

type ProcessMessageRequest struct {
    SessionID uuid.UUID
    Content   string
    UserID    uuid.UUID
}

type ProcessMessageResponse struct {
    MessageID uuid.UUID
    Content   string
    ToolCalls []ToolCall
    Metadata  MessageMetadata
}
```

### 4. ToolOrchestrationService

**Responsabilidades**:
- Detectar function calling no LLM response
- Executar tools via Tools Gateway
- Formatar resultados para LLM
- Handle errors e retries

**Métodos**:
```go
type ToolOrchestrationService interface {
    ExecuteToolCall(ctx, *ToolCall, sessionID) (*ToolResult, error)
    ExecuteMultipleTools(ctx, []ToolCall, sessionID) ([]ToolResult, error)
    FormatToolResult(result) string
}
```

### 5. LLMClientPool

**Responsabilidades**:
- Gerenciar clients OpenAI, Anthropic, etc
- Rate limiting
- Token counting
- Cost tracking

**Métodos**:
```go
type LLMClientPool interface {
    GetClient(model string) (LLMClient, error)
    Chat(ctx, *ChatRequest) (*ChatResponse, error)
    StreamChat(ctx, *ChatRequest) (<-chan ChatChunk, error)
}

type LLMClient interface {
    Chat(ctx, messages, config) (*Response, error)
    Stream(ctx, messages, config) (<-chan Chunk, error)
    CountTokens(messages) int
}
```

---

## Integrações

### 1. Redis (Session Store)

**Uso**:
- Armazenar contexto de sessão
- TTL automático
- Pub/Sub para real-time updates

**Schema**:
```
session:{session_id}           -> JSON da Session
session:{session_id}:messages  -> Lista de messages
session:{session_id}:context   -> Context variables
```

**TTL**: 24 horas (configurável)

### 2. Tools Gateway

**Endpoints usados**:
```
GET  /api/v1/tools
POST /api/v1/tools/:id/execute
```

**Fluxo**:
1. LLM retorna function call
2. Orchestrator identifica tool
3. Chama Tools Gateway com parâmetros
4. Recebe resultado
5. Formata para LLM
6. LLM processa resultado

### 3. OpenAI API

**Modelos suportados**:
- gpt-4-turbo
- gpt-4
- gpt-3.5-turbo

**Features**:
- Function calling
- Streaming
- Token counting
- Cost calculation

### 4. Anthropic API

**Modelos suportados**:
- claude-3-opus
- claude-3-sonnet
- claude-3-haiku

**Features**:
- Tool use
- Streaming
- Token counting

### 5. Kafka (Event Publishing)

**Topics**:
```
agent-executions     -> AgentExecutionEvent
tool-calls           -> ToolCallEvent
session-events       -> SessionEvent
errors               -> ErrorEvent
```

---

## Implementação

### Fase 1: MVP (Core Functionality)

#### Sprint 1: Infrastructure
- [ ] Setup Redis client
- [ ] Setup Kafka producer
- [ ] Create database schema (agents, agent_executions)
- [ ] Migrations

#### Sprint 2: Domain Layer
- [ ] Session entity
- [ ] Message entity
- [ ] Agent entity
- [ ] Repository interfaces

#### Sprint 3: Session Management
- [ ] SessionService implementation
- [ ] Redis session store
- [ ] TTL management
- [ ] Message history

#### Sprint 4: Agent Management
- [ ] AgentService CRUD
- [ ] Agent configuration
- [ ] System prompts
- [ ] Tool associations

#### Sprint 5: LLM Integration
- [ ] OpenAI client
- [ ] Anthropic client
- [ ] Client pool
- [ ] Token counting
- [ ] Cost calculation

#### Sprint 6: Message Processing
- [ ] MessageProcessingService
- [ ] Context management
- [ ] LLM invocation
- [ ] Response handling

#### Sprint 7: Tool Orchestration
- [ ] Function calling detection
- [ ] Tools Gateway client
- [ ] Tool execution
- [ ] Result formatting

#### Sprint 8: API Layer
- [ ] HTTP handlers
- [ ] DTOs
- [ ] Routes
- [ ] Error handling

#### Sprint 9: Testing & Documentation
- [ ] Unit tests
- [ ] Integration tests
- [ ] README
- [ ] API documentation

### Fase 2: Advanced Features

- [ ] Agent routing (intent recognition)
- [ ] Delegation between agents
- [ ] Streaming responses (WebSocket/SSE)
- [ ] Multi-language support
- [ ] Voice integration
- [ ] Context variables
- [ ] Conditional logic

### Fase 3: Production

- [ ] Rate limiting (Redis)
- [ ] Circuit breakers
- [ ] Observability (Prometheus)
- [ ] Distributed tracing
- [ ] Load testing
- [ ] Performance optimization

---

## Roadmap

### Q1 2026
- ✅ Planning document
- ⏳ Fase 1: MVP implementation
- ⏳ Basic session management
- ⏳ OpenAI integration
- ⏳ Tool orchestration

### Q2 2026
- Agent routing
- Anthropic integration
- Streaming support
- Advanced context management

### Q3 2026
- Multi-model support
- Delegation features
- Voice integration
- Analytics dashboard

### Q4 2026
- AI-powered routing
- Auto-scaling
- Cost optimization
- Enterprise features

---

## Decisões Técnicas

### Por que Redis?
- Fast in-memory storage
- TTL nativo
- Pub/Sub para real-time
- Amplamente suportado

### Por que não PostgreSQL para sessões?
- Sessões são temporárias (24h)
- Leitura/escrita muito frequente
- Redis é mais performático
- Menos carga no DB principal

### Streaming vs Request/Response?
- Ambos suportados
- Streaming para UX melhor
- Request/Response mais simples
- Client escolhe

### Function Calling vs Prompt Engineering?
- Function calling mais confiável
- Menos parsing manual
- Melhor structured output
- Suportado por OpenAI e Anthropic

---

## Estimativas

### Linhas de Código
- Domain: ~500 linhas
- Use Cases: ~1,500 linhas
- Infrastructure: ~1,000 linhas
- Presentation: ~800 linhas
- **Total**: ~3,800 linhas

### Tempo de Desenvolvimento
- Fase 1 (MVP): 4-6 semanas
- Fase 2 (Advanced): 3-4 semanas
- Fase 3 (Production): 2-3 semanas
- **Total**: 9-13 semanas

### Complexidade
- **Alta**: LLM integration, streaming, real-time
- **Média**: Session management, tool orchestration
- **Baixa**: CRUD operations

---

## Referências

- [OpenAI API Documentation](https://platform.openai.com/docs/api-reference)
- [Anthropic API Documentation](https://docs.anthropic.com/)
- [Redis Documentation](https://redis.io/docs/)
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [LangChain Architecture](https://python.langchain.com/docs/get_started/introduction)

---

## Próximos Passos

1. ✅ Criar este documento de planejamento
2. ⏳ Setup inicial (Redis, Kafka clients)
3. ⏳ Implementar domain layer
4. ⏳ Implementar session management
5. ⏳ Integrar OpenAI
6. ⏳ Implementar tool orchestration
7. ⏳ Criar API REST
8. ⏳ Testes e documentação

---

**Status**: 📝 Planning Phase  
**Prioridade**: Alta  
**Complexidade**: Alta  
**Dependencies**: Tools Gateway (✅ Completo)
