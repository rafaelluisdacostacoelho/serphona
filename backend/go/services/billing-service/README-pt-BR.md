# Serviço de Cobrança (Billing Service)

Microsserviço responsável por gerenciar cobranças, assinaturas, faturas e wallet de créditos, integrando-se com a Stripe para processamento de pagamentos.

## Responsabilidades do Serviço

- Integração completa com Stripe (produtos, planos, assinaturas, faturas)
- Gerenciamento de clientes (customers) e vinculação com tenants
- Processamento de webhooks do Stripe para eventos de pagamento
- Gerenciamento de assinaturas (criação, atualização, cancelamento)
- Controle de faturas e histórico de pagamentos
- Sistema de wallet de créditos para uso da plataforma
- Portal de autoatendimento para clientes (Stripe Billing Portal)
- Gerenciamento de sessões de checkout
- Controle de uso e quotas por tenant
- Publicação de eventos de cobrança e pagamentos

## Arquitetura

Este serviço segue a **Arquitetura Hexagonal** (Portas e Adaptadores):

```
┌───────────────────────────────────────────────────────────┐
│                  ADAPTADORES (Condutores)                 │
│     ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│     │  REST API   │  │   Webhook   │  │   Kafka     │     │
│     │  Handler    │  │   Stripe    │  │  Consumer   │     │
│     └──────┬──────┘  └──────┬──────┘  └──────┬──────┘     │
│            │                │                │            │
│            └────────────────┼────────────────┘            │
│                             ▼                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                 CAMADA DE APLICAÇÃO                 │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │  Customer   │  │Subscription │  │   Invoice   │  │  │
│  │  │   Service   │  │   Service   │  │   Service   │  │  │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  │  │
│  │  ┌─────────────┐  ┌─────────────┐                  │  │
│  │  │   Wallet    │  │   Payment   │                  │  │
│  │  │   Service   │  │   Service   │                  │  │
│  │  └──────┬──────┘  └──────┬──────┘                  │  │
│  └─────────┼────────────────┼────────────────┼─────────┘  │
│            │                │                │            │
│            └────────────────┼────────────────┘            │
│                             ▼                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                  CAMADA DE DOMÍNIO                  │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │  Customer   │  │Subscription │  │   Invoice   │  │  │
│  │  │   Entity    │  │   Entity    │  │   Entity    │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │  │
│  │  ┌─────────────┐  ┌─────────────┐                  │  │
│  │  │   Wallet    │  │   Payment   │                  │  │
│  │  │   Entity    │  │   Entity    │                  │  │
│  │  └─────────────┘  └─────────────┘                  │  │
│  │                                                     │  │
│  │                  PORTAS (Interfaces)                │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐          │  │
│  │  │CustomerR │  │StripeAPI │  │EventPub  │          │  │
│  │  └──────────┘  └──────────┘  └──────────┘          │  │
│  └─────────────────────────────────────────────────────┘  │
│                             │                             │
│                             ▼                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                ADAPTADORES (Conduzidos)             │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │ PostgreSQL  │  │   Stripe    │  │    Redis    │  │  │
│  │  │   Repo      │  │    API      │  │    Cache    │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │  │
│  │  ┌─────────────┐                                   │  │
│  │  │   Kafka     │                                   │  │
│  │  │  Publisher  │                                   │  │
│  │  └─────────────┘                                   │  │
│  └─────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────┘
```

## Estrutura de Pastas

```
billing-service/
├── cmd/
│   └── server/
│       └── main.go                 # Ponto de entrada da aplicação
├── internal/
│   ├── domain/                     # Camada de domínio (lógica de negócio)
│   │   ├── customer/
│   │   │   ├── entity.go           # Entidade Customer
│   │   │   ├── repository.go       # Interface do repositório (porta)
│   │   │   ├── service.go          # Serviço de domínio
│   │   │   └── errors.go           # Erros específicos do domínio
│   │   ├── subscription/
│   │   │   ├── entity.go
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   ├── invoice/
│   │   │   ├── entity.go
│   │   │   └── service.go
│   │   ├── wallet/
│   │   │   ├── entity.go
│   │   │   ├── repository.go
│   │   │   └── service.go
│   │   └── events/
│   │       └── events.go           # Eventos de domínio
│   ├── application/                # Camada de aplicação (casos de uso)
│   │   ├── customer/
│   │   │   ├── service.go          # Serviço de aplicação
│   │   │   ├── dto.go              # DTOs
│   │   │   └── commands.go         # Command/Query objects
│   │   ├── subscription/
│   │   │   └── service.go
│   │   ├── invoice/
│   │   │   └── service.go
│   │   └── wallet/
│   │       └── service.go
│   ├── adapter/                    # Adaptadores (infraestrutura)
│   │   ├── http/                   # Adaptador HTTP (REST API)
│   │   │   ├── router.go
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go
│   │   │   │   ├── tenant.go
│   │   │   │   └── logging.go
│   │   │   ├── handler/
│   │   │   │   ├── customer.go
│   │   │   │   ├── subscription.go
│   │   │   │   ├── invoice.go
│   │   │   │   ├── wallet.go
│   │   │   │   ├── webhook.go
│   │   │   │   └── health.go
│   │   │   └── dto/
│   │   │       ├── request.go
│   │   │       └── response.go
│   │   ├── stripe/                 # Adaptador Stripe
│   │   │   ├── client.go
│   │   │   ├── customer.go
│   │   │   ├── subscription.go
│   │   │   ├── invoice.go
│   │   │   └── webhook.go
│   │   ├── postgres/               # Adaptador PostgreSQL
│   │   │   ├── customer_repo.go
│   │   │   ├── subscription_repo.go
│   │   │   └── wallet_repo.go
│   │   ├── redis/                  # Adaptador Redis
│   │   │   └── cache.go
│   │   └── kafka/                  # Adaptador Kafka
│   │       └── publisher.go
│   └── config/                     # Configuração
│       └── config.go
├── pkg/                            # Pacotes compartilhados
│   ├── logger/
│   │   └── logger.go
│   └── errors/
│       └── errors.go
├── migrations/                     # Migrações de banco de dados
│   ├── 000001_create_customers.up.sql
│   ├── 000001_create_customers.down.sql
│   ├── 000002_create_subscriptions.up.sql
│   ├── 000002_create_subscriptions.down.sql
│   ├── 000003_create_wallets.up.sql
│   └── 000003_create_wallets.down.sql
├── scripts/
│   ├── migrate.sh
│   └── setup-stripe.sh
├── .env.example
├── Dockerfile
├── Makefile
├── go.mod
└── go.sum
```

## Início Rápido

```bash
# Copiar arquivo de configuração
cp .env.example .env

# Editar .env e adicionar suas chaves Stripe
# STRIPE_SECRET_KEY=sk_test_...
# STRIPE_WEBHOOK_SECRET=whsec_...

# Executar com Docker Compose
docker-compose up -d

# Executar localmente
export $(cat .env | xargs)
go run cmd/server/main.go

# Executar testes
make test

# Executar migrações
make migrate-up
```

## Endpoints da API

### Customers

| Método | Caminho | Descrição |
|--------|---------|-----------|
| POST | /api/v1/customers | Criar novo customer (vinculado a tenant) |
| GET | /api/v1/customers/{id} | Obter customer por ID |

### Assinaturas

| Método | Caminho | Descrição |
|--------|---------|-----------|
| GET | /api/v1/subscriptions | Listar assinaturas do tenant |
| POST | /api/v1/subscriptions | Criar nova assinatura |
| GET | /api/v1/subscriptions/{id} | Obter assinatura por ID |
| PUT | /api/v1/subscriptions/{id} | Atualizar assinatura |
| DELETE | /api/v1/subscriptions/{id} | Cancelar assinatura |

### Faturas

| Método | Caminho | Descrição |
|--------|---------|-----------|
| GET | /api/v1/invoices | Listar faturas |
| GET | /api/v1/invoices/{id} | Obter fatura por ID |

### Planos e Produtos

| Método | Caminho | Descrição |
|--------|---------|-----------|
| GET | /api/v1/plans | Listar planos disponíveis |
| GET | /api/v1/products | Listar produtos disponíveis |

### Wallet e Uso

| Método | Caminho | Descrição |
|--------|---------|-----------|
| GET | /api/v1/usage | Obter uso atual do tenant |
| GET | /api/v1/wallet | Obter saldo da wallet |
| POST | /api/v1/wallet/topup | Adicionar créditos à wallet |

### Stripe Portal e Checkout

| Método | Caminho | Descrição |
|--------|---------|-----------|
| POST | /api/v1/portal-session | Criar sessão do Billing Portal |
| POST | /api/v1/checkout-session | Criar sessão de Checkout |

### Webhooks

| Método | Caminho | Descrição |
|--------|---------|-----------|
| POST | /webhooks/stripe | Receber webhooks do Stripe |

### Health e Métricas

| Método | Caminho | Descrição |
|--------|---------|-----------|
| GET | /health | Verificação de saúde |
| GET | /metrics | Métricas Prometheus |

## Planos de Assinatura

| Plano | Créditos/Mês | Preço (exemplo) | Recursos |
|-------|--------------|-----------------|----------|
| Free | 100 | R$ 0 | Recursos básicos, limitado |
| Starter | 1,000 | R$ 49 | Ideal para pequenas empresas |
| Pro | 10,000 | R$ 199 | Recursos avançados, maior volume |
| Enterprise | 100,000+ | Sob consulta | Recursos personalizados, SLA |

## Variáveis de Ambiente

### Configuração do Servidor

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| SERVER_HOST | Host do servidor | 0.0.0.0 |
| SERVER_PORT | Porta do servidor | 8081 |
| ENV | Ambiente (development/production) | development |

### Banco de Dados

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| DATABASE_URL | URL de conexão PostgreSQL | - |
| DB_HOST | Host do PostgreSQL | localhost |
| DB_PORT | Porta do PostgreSQL | 5432 |
| DB_USER | Usuário do banco | postgres |
| DB_PASSWORD | Senha do banco | postgres |
| DB_NAME | Nome do banco | serphona_billing |

### Stripe

| Variável | Descrição | Obrigatório |
|----------|-----------|-------------|
| STRIPE_SECRET_KEY | Chave secreta da Stripe | Sim |
| STRIPE_PUBLISHABLE_KEY | Chave pública da Stripe | Sim |
| STRIPE_WEBHOOK_SECRET | Segredo do webhook | Sim |
| STRIPE_API_VERSION | Versão da API Stripe | Não |

### Redis

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| REDIS_URL | URL de conexão Redis | redis://localhost:6379 |
| REDIS_HOST | Host do Redis | localhost |
| REDIS_PORT | Porta do Redis | 6379 |
| REDIS_DB | Database do Redis | 1 |

### Kafka

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| KAFKA_BROKERS | Brokers Kafka | localhost:9092 |
| KAFKA_GROUP_ID | ID do grupo consumidor | billing-service |
| KAFKA_TOPICS | Tópicos para publicar eventos | billing.events,payments.webhooks |

### Wallet

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| WALLET_DEFAULT_CURRENCY | Moeda padrão | BRL |
| WALLET_INITIAL_CREDITS | Créditos iniciais | 100 |
| WALLET_MIN_TOPUP_AMOUNT | Valor mínimo de recarga | 1000 (centavos) |
| WALLET_MAX_TOPUP_AMOUNT | Valor máximo de recarga | 1000000 (centavos) |

### Feature Flags

| Variável | Descrição | Padrão |
|----------|-----------|--------|
| ENABLE_CREDIT_WALLET | Habilitar sistema de wallet | true |
| ENABLE_SUBSCRIPTIONS | Habilitar assinaturas | true |
| ENABLE_INVOICING | Habilitar faturamento | false |
| ENABLE_PAYMENT_METHODS | Habilitar métodos de pagamento | true |

## Eventos Kafka

### Eventos Publicados

- `billing.customer.created` - Customer criado
- `billing.subscription.created` - Assinatura criada
- `billing.subscription.updated` - Assinatura atualizada
- `billing.subscription.cancelled` - Assinatura cancelada
- `billing.payment.succeeded` - Pagamento bem-sucedido
- `billing.payment.failed` - Pagamento falhou
- `billing.invoice.created` - Fatura criada
- `billing.wallet.credited` - Créditos adicionados à wallet
- `billing.wallet.debited` - Créditos debitados da wallet

### Eventos Consumidos

- `tenant.created` - Novo tenant criado (criar customer no Stripe)
- `usage.reported` - Uso reportado (debitar wallet)

## Webhooks do Stripe

O serviço processa os seguintes eventos do Stripe:

- `customer.subscription.created` - Nova assinatura criada
- `customer.subscription.updated` - Assinatura atualizada
- `customer.subscription.deleted` - Assinatura cancelada
- `invoice.payment_succeeded` - Pagamento de fatura bem-sucedido
- `invoice.payment_failed` - Falha no pagamento de fatura
- `customer.created` - Customer criado
- `payment_method.attached` - Método de pagamento adicionado
- `payment_method.detached` - Método de pagamento removido

## Desenvolvimento

### Pré-requisitos

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- Kafka 3.0+
- Conta Stripe (modo teste)

### Configuração Local

1. Clone o repositório
2. Copie `.env.example` para `.env`
3. Configure suas chaves da Stripe no `.env`
4. Inicie as dependências:
   ```bash
   docker-compose up -d postgres redis kafka
   ```
5. Execute as migrações:
   ```bash
   make migrate-up
   ```
6. Inicie o servidor:
   ```bash
   go run cmd/server/main.go
   ```

### Configurando Webhooks Stripe Local

Para testar webhooks localmente, use o Stripe CLI:

```bash
# Instalar Stripe CLI
# https://stripe.com/docs/stripe-cli

# Login
stripe login

# Encaminhar webhooks para seu servidor local
stripe listen --forward-to localhost:8081/webhooks/stripe

# Use o webhook secret fornecido no .env
```

### Testes

```bash
# Executar todos os testes
make test

# Testes com cobertura
make test-coverage

# Testes de integração
make test-integration
```

## Modelo de Dados

### Customer

```sql
customers (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  stripe_customer_id VARCHAR(255) UNIQUE,
  email VARCHAR(255),
  name VARCHAR(255),
  metadata JSONB,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)
```

### Subscription

```sql
subscriptions (
  id UUID PRIMARY KEY,
  customer_id UUID REFERENCES customers(id),
  stripe_subscription_id VARCHAR(255) UNIQUE,
  plan_id VARCHAR(100),
  status VARCHAR(50),
  current_period_start TIMESTAMP,
  current_period_end TIMESTAMP,
  cancel_at_period_end BOOLEAN,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)
```

### Wallet

```sql
wallets (
  id UUID PRIMARY KEY,
  tenant_id UUID UNIQUE NOT NULL,
  balance BIGINT DEFAULT 0,
  currency VARCHAR(3) DEFAULT 'BRL',
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)

wallet_transactions (
  id UUID PRIMARY KEY,
  wallet_id UUID REFERENCES wallets(id),
  amount BIGINT,
  type VARCHAR(20), -- credit, debit
  description TEXT,
  metadata JSONB,
  created_at TIMESTAMP
)
```

## Integração com Outros Serviços

### Tenant Manager

- Recebe eventos `tenant.created` para criar customer no Stripe
- Notifica sobre mudanças de plano/assinatura

### Voice Gateway

- Consome créditos da wallet para chamadas de voz
- Reporta uso via eventos Kafka

### Analytics Service

- Fornece dados de faturamento e uso
- Gera relatórios de receita

## Segurança

- Autenticação via JWT (tokens do auth-gateway)
- Validação de assinatura dos webhooks Stripe
- Chaves de API do Stripe armazenadas como variáveis de ambiente
- Comunicação HTTPS obrigatória em produção
- Logs de auditoria para transações financeiras

## Monitoramento

- Métricas Prometheus em `/metrics`
- Health checks em `/health`
- Logs estruturados (JSON)
- Tracing com Jaeger (opcional)
- Alertas para:
  - Falhas de pagamento
  - Erros de webhook
  - Saldo baixo na wallet
  - Anomalias de uso

## Documentação Relacionada

- [Arquitetura da Plataforma](../../../docs/architecture/README.md)
- [Estratégia de Precificação](../../../infra/PRICING_STRATEGY_AWS.md)
- [Estimativa de Custos](../../../infra/COST_ESTIMATION.md)
- [Documentação da Stripe API](https://stripe.com/docs/api)

## Suporte e Contribuição

Para reportar bugs ou solicitar funcionalidades, abra uma issue no repositório.

---

**Versão**: 1.0.0  
**Licença**: Proprietary
