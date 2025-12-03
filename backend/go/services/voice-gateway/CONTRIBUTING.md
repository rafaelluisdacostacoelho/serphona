# Contributing to Voice Gateway

Obrigado por considerar contribuir para o Voice Gateway! Este documento fornece diretrizes e melhores práticas para contribuições.

## 📋 Índice

- [Código de Conduta](#código-de-conduta)
- [Como Contribuir](#como-contribuir)
- [Desenvolvimento Local](#desenvolvimento-local)
- [Pull Requests](#pull-requests)
- [Padrões de Código](#padrões-de-código)
- [Testes](#testes)
- [Documentação](#documentação)

## 📜 Código de Conduta

Este projeto adere ao [Código de Conduta do Contributor Covenant](https://www.contributor-covenant.org/). Ao participar, você concorda em seguir esses termos.

## 🤝 Como Contribuir

### Reportando Bugs

Se você encontrou um bug, por favor:

1. Verifique se o bug já não foi reportado em [Issues](https://github.com/serphona/serphona/issues)
2. Crie uma nova issue com:
   - Descrição clara do problema
   - Passos para reproduzir
   - Comportamento esperado vs atual
   - Versão do software
   - Logs relevantes

**Template de Bug Report:**

```markdown
## Descrição
[Descrição clara e concisa do bug]

## Reproduzir
1. Vá para '...'
2. Execute '...'
3. Veja erro

## Comportamento Esperado
[O que deveria acontecer]

## Comportamento Atual
[O que está acontecendo]

## Ambiente
- OS: [e.g. Ubuntu 22.04]
- Go Version: [e.g. 1.23]
- Voice Gateway Version: [e.g. 1.0.0]

## Logs
```
[Cole logs relevantes aqui]
```
```

### Sugerindo Melhorias

Para sugerir novas funcionalidades:

1. Verifique se a funcionalidade já não foi sugerida
2. Crie uma issue com:
   - Descrição da funcionalidade
   - Casos de uso
   - Benefícios esperados
   - Possíveis implementações

## 💻 Desenvolvimento Local

### Pré-requisitos

- Go 1.23+
- Docker & Docker Compose
- Make
- Git

### Setup

```bash
# Clone o repositório
git clone https://github.com/serphona/serphona.git
cd serphona/backend/go/services/voice-gateway

# Instale dependências
go mod download

# Configure ambiente
cp .env.example .env
# Edite .env com suas configurações

# Inicie dependências (Redis, Kafka, etc)
docker-compose up -d redis kafka zookeeper

# Execute o serviço
go run cmd/server/main.go
```

### Comandos Úteis

```bash
# Build
make build

# Run
make run

# Tests
make test

# Lint
make lint

# Format code
make fmt

# Docker build
make docker-build

# Docker run
make docker-run
```

## 🔀 Pull Requests

### Processo

1. **Fork** o repositório
2. **Crie um branch** para sua feature/fix:
   ```bash
   git checkout -b feature/nova-funcionalidade
   # ou
   git checkout -b fix/correcao-bug
   ```

3. **Faça commits** seguindo o padrão:
   ```bash
   git commit -m "feat: adiciona suporte para TTS streaming"
   git commit -m "fix: corrige memory leak no event loop"
   git commit -m "docs: atualiza README com exemplos"
   ```

4. **Push** para seu fork:
   ```bash
   git push origin feature/nova-funcionalidade
   ```

5. **Abra um Pull Request** no repositório original

### Padrão de Commits

Seguimos o [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: Nova funcionalidade
- `fix`: Correção de bug
- `docs`: Documentação
- `style`: Formatação (sem mudança de código)
- `refactor`: Refatoração
- `test`: Testes
- `chore`: Manutenção

**Exemplos:**

```bash
feat(asterisk): implementa reconnection automática no ARI client
fix(redis): corrige race condition no call state repository
docs(api): adiciona exemplos de uso para transfer endpoint
test(domain): adiciona testes para call lifecycle
refactor(http): extrai handler comum para responses
```

### Checklist do PR

Antes de enviar seu PR, verifique:

- [ ] Código compila sem erros
- [ ] Testes passam (`make test`)
- [ ] Código formatado (`make fmt`)
- [ ] Lint sem erros (`make lint`)
- [ ] Documentação atualizada
- [ ] CHANGELOG.md atualizado
- [ ] Commit messages seguem padrão
- [ ] PR tem descrição clara

## 🎨 Padrões de Código

### Arquitetura

Seguimos **Hexagonal Architecture (Ports and Adapters)**:

```
voice-gateway/
├── cmd/              # Entry points
├── internal/
│   ├── domain/      # Core business logic (sem dependências externas)
│   ├── application/ # Use cases / orchestration
│   └── adapter/     # External integrations (HTTP, DB, etc)
```

### Go Style Guide

Seguimos [Effective Go](https://golang.org/doc/effective_go.html) e [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments):

**Boas Práticas:**

```go
// ✅ BOM: Nomes descritivos
func (s *Service) HandleIncomingCall(ctx context.Context, channelID string) (*Call, error) {
    // Implementation
}

// ❌ RUIM: Nomes genéricos
func (s *Service) Handle(ctx context.Context, id string) (*Call, error) {
    // Implementation
}

// ✅ BOM: Retorno de erro específico
return nil, fmt.Errorf("failed to answer channel %s: %w", channelID, err)

// ❌ RUIM: Erro genérico
return nil, err

// ✅ BOM: Context propagation
func (c *Client) GetChannel(ctx context.Context, id string) (*Channel, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    // ...
}

// ❌ RUIM: Sem context
func (c *Client) GetChannel(id string) (*Channel, error) {
    req, _ := http.NewRequest("GET", url, nil)
    // ...
}
```

### Naming Conventions

```go
// Interfaces: sufixo com verbo/ação
type CallRepository interface {
    Save(ctx context.Context, call *Call) error
    Find(ctx context.Context, id uuid.UUID) (*Call, error)
}

// Structs: substantivos
type Call struct {
    ID       uuid.UUID
    TenantID uuid.UUID
    // ...
}

// Methods: verbos
func (c *Call) Answer() {
    // ...
}

// Packages: minúsculas, singular
package call

// Constants: PascalCase ou SCREAMING_SNAKE_CASE
const MaxRetries = 3
const DEFAULT_TIMEOUT = 30 * time.Second
```

### Error Handling

```go
// Sempre wrap errors com contexto
if err != nil {
    return fmt.Errorf("failed to connect to ARI: %w", err)
}

// Use custom error types para casos específicos
type ValidationError struct {
    Field string
    Value interface{}
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("invalid value for %s: %v", e.Field, e.Value)
}

// Logging estruturado
logger.Error("failed to process event",
    zap.String("event_type", eventType),
    zap.Error(err),
)
```

## 🧪 Testes

### Estrutura

```
internal/
├── domain/
│   ├── call/
│   │   ├── call.go
│   │   └── call_test.go     # Testes unitários
├── application/
│   ├── call/
│   │   ├── service.go
│   │   └── service_test.go  # Testes de use cases
└── adapter/
    ├── http/
    │   ├── handler/
    │   │   ├── call_handler.go
    │   │   └── call_handler_test.go  # Testes de integração
```

### Escrevendo Testes

```go
func TestCall_Answer(t *testing.T) {
    // Arrange
    call := NewCall(uuid.New(), DirectionInbound, "+5511999887766", "+5511988776655")

    // Act
    call.Answer()

    // Assert
    if call.State != StateAnswered {
        t.Errorf("Expected state %s, got %s", StateAnswered, call.State)
    }
    if call.AnsweredAt == nil {
        t.Error("AnsweredAt should be set")
    }
}

// Table-driven tests
func TestCallState_Transitions(t *testing.T) {
    tests := []struct {
        name          string
        initialState  State
        action        func(*Call)
        expectedState State
    }{
        {
            name:          "Ringing to Answered",
            initialState:  StateRinging,
            action:        func(c *Call) { c.Answer() },
            expectedState: StateAnswered,
        },
        // More cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            call := NewCall(uuid.New(), DirectionInbound, "+5511", "+5522")
            call.State = tt.initialState
            tt.action(call)
            if call.State != tt.expectedState {
                t.Errorf("Expected %s, got %s", tt.expectedState, call.State)
            }
        })
    }
}
```

### Rodando Testes

```bash
# Todos os testes
make test

# Com coverage
make test-coverage

# Específico
go test ./internal/domain/call/... -v

# Com race detector
go test -race ./...
```

### Cobertura

Mantenha cobertura mínima de **80%**:

```bash
make test-coverage
open coverage.html
```

## 📚 Documentação

### O que Documentar

1. **README.md**: Overview, setup, usage
2. **API.md**: Documentação completa da API
3. **DOCKER.md**: Guia de deployment
4. **Código**: Comentários em funções públicas

### GoDoc

```go
// Package call contains the call domain model and business logic.
//
// This package provides core entities and value objects for managing
// phone calls in the voice gateway system.
package call

// Call represents a phone call in the system.
//
// A call goes through multiple states: ringing -> answered -> active -> ended.
// It maintains metadata about the conversation and integrates with external
// services like Asterisk, Redis, and Kafka.
type Call struct {
    // ...
}

// NewCall creates a new Call instance.
//
// The call is initialized in the "ringing" state and assigned a unique ID.
// All timestamps are in UTC.
func NewCall(tenantID uuid.UUID, direction Direction, callerNumber, calleeNumber string) *Call {
    // ...
}
```

### Atualizando Documentação

Ao adicionar/modificar features:

1. Atualize README.md se necessário
2. Atualize API.md para novos endpoints
3. Adicione exemplos de uso
4. Atualize CHANGELOG.md

## 🏅 Reconhecimento

Contribuidores serão listados em `CONTRIBUTORS.md`.

## 📞 Contato

- **Issues**: https://github.com/serphona/serphona/issues
- **Discussions**: https://github.com/serphona/serphona/discussions
- **Email**: dev@serphona.com

---

**Obrigado por contribuir! 🎉**
