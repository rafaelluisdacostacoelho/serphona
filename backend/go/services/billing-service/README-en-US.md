# Billing Service

Microservice responsible for managing billing, subscriptions, invoices, and credit wallets, integrating with Stripe for payment processing.

## Service Responsibilities

- Complete Stripe integration (products, plans, subscriptions, invoices)
- Customer management and tenant linking
- Stripe webhook processing for payment events
- Subscription management (create, update, cancel)
- Invoice and payment history management
- Credit wallet system for platform usage
- Self-service customer portal (Stripe Billing Portal)
- Checkout session management
- Usage and quota control per tenant
- Publishing billing and payment events

## Architecture

This service follows **Hexagonal Architecture** (Ports and Adapters):

```
┌───────────────────────────────────────────────────────────┐
│                   ADAPTERS (Driving)                      │
│     ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│     │  REST API   │  │   Webhook   │  │   Kafka     │     │
│     │  Handler    │  │   Stripe    │  │  Consumer   │     │
│     └──────┬──────┘  └──────┬──────┘  └──────┬──────┘     │
│            │                │                │            │
│            └────────────────┼────────────────┘            │
│                             ▼                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                 APPLICATION LAYER                   │  │
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
│  │                   DOMAIN LAYER                      │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │  │
│  │  │  Customer   │  │Subscription │  │   Invoice   │  │  │
│  │  │   Entity    │  │   Entity    │  │   Entity    │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  │  │
│  │  ┌─────────────┐  ┌─────────────┐                  │  │
│  │  │   Wallet    │  │   Payment   │                  │  │
│  │  │   Entity    │  │   Entity    │                  │  │
│  │  └─────────────┘  └─────────────┘                  │  │
│  │                                                     │  │
│  │                   PORTS (Interfaces)                │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐          │  │
│  │  │CustomerR │  │StripeAPI │  │EventPub  │          │  │
│  │  └──────────┘  └──────────┘  └──────────┘          │  │
│  └─────────────────────────────────────────────────────┘  │
│                             │                             │
│                             ▼                             │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                 ADAPTERS (Driven)                   │  │
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

## Folder Structure

```
billing-service/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── domain/                     # Domain layer (business logic)
│   │   ├── customer/
│   │   │   ├── entity.go           # Customer entity
│   │   │   ├── repository.go       # Repository interface (port)
│   │   │   ├── service.go          # Domain service
│   │   │   └── errors.go           # Domain-specific errors
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
│   │       └── events.go           # Domain events
│   ├── application/                # Application layer (use cases)
│   │   ├── customer/
│   │   │   ├── service.go          # Application service
│   │   │   ├── dto.go              # DTOs
│   │   │   └── commands.go         # Command/Query objects
│   │   ├── subscription/
│   │   │   └── service.go
│   │   ├── invoice/
│   │   │   └── service.go
│   │   └── wallet/
│   │       └── service.go
│   ├── adapter/                    # Adapters (infrastructure)
│   │   ├── http/                   # HTTP adapter (REST API)
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
│   │   ├── stripe/                 # Stripe adapter
│   │   │   ├── client.go
│   │   │   ├── customer.go
│   │   │   ├── subscription.go
│   │   │   ├── invoice.go
│   │   │   └── webhook.go
│   │   ├── postgres/               # PostgreSQL adapter
│   │   │   ├── customer_repo.go
│   │   │   ├── subscription_repo.go
│   │   │   └── wallet_repo.go
│   │   ├── redis/                  # Redis adapter
│   │   │   └── cache.go
│   │   └── kafka/                  # Kafka adapter
│   │       └── publisher.go
│   └── config/                     # Configuration
│       └── config.go
├── pkg/                            # Shared packages
│   ├── logger/
│   │   └── logger.go
│   └── errors/
│       └── errors.go
├── migrations/                     # Database migrations
│   ├── 000001_create_customers.up.sql
│   ├── 000001_create_customers.down.sql
│   ├── 000002_create_subscriptions.up.sql
│   ├── 000002_create_subscriptions.down.sql
│   ├── 000003_create_wallets.up.sql
│   ├── 000003_create_wallets.down.sql
│   ├── 000004_create_pricing_plans.up.sql
│   ├── 000004_create_pricing_plans.down.sql
│   ├── 000005_add_wallet_transactions_reference_unique.up.sql
│   ├── 000005_add_wallet_transactions_reference_unique.down.sql
│   ├── 000006_seed_test_wallets_optional.up.sql
│   ├── 000006_seed_test_wallets_optional.down.sql
│   ├── 000007_update_pricing_plans_real.up.sql
│   ├── 000007_update_pricing_plans_real.down.sql
│   ├── 000008_add_wallet_transactions_request_unique.up.sql
│   └── 000008_add_wallet_transactions_request_unique.down.sql
├── scripts/
│   ├── migrate.sh
│   └── setup-stripe.sh
├── .env.example
├── Dockerfile
├── Makefile
├── go.mod
└── go.sum
```

## Quick Start

```bash
# Copy configuration file
cp .env.example .env

# Edit .env and add your Stripe keys
# STRIPE_SECRET_KEY=sk_test_...
# STRIPE_WEBHOOK_SECRET=whsec_...

# Run with Docker Compose
docker-compose up -d

# Run locally
export $(cat .env | xargs)
go run cmd/server/main.go

# Run tests
make test

# Run migrations
make migrate-up

# Optionally seed test wallets (non-production)
SEED_TEST_WALLETS=true make migrate-up
```

## API Endpoints

### Customers

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/customers | Create new customer (linked to tenant) |
| GET | /api/v1/customers/{id} | Get customer by ID |

### Subscriptions

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/subscriptions | List tenant subscriptions |
| POST | /api/v1/subscriptions | Create new subscription |
| GET | /api/v1/subscriptions/{id} | Get subscription by ID |
| PUT | /api/v1/subscriptions/{id} | Update subscription |
| DELETE | /api/v1/subscriptions/{id} | Cancel subscription |

### Invoices

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/invoices | List invoices |
| GET | /api/v1/invoices/{id} | Get invoice by ID |

### Plans and Products

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/plans | List available plans |
| GET | /api/v1/products | List available products |

### Wallet and Usage

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/v1/usage | Get current tenant usage |
| GET | /api/v1/wallet | Get wallet balance |
| POST | /api/v1/wallet/topup | Add credits to wallet |

### Stripe Portal and Checkout

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/v1/portal-session | Create Billing Portal session |
| POST | /api/v1/checkout-session | Create Checkout session |

### Webhooks

| Method | Path | Description |
|--------|------|-------------|
| POST | /webhooks/stripe | Receive Stripe webhooks |

### Health and Metrics

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |
| GET | /metrics | Prometheus metrics |

## Subscription Plans

| Plan | Credits/Month | Price (example) | Features |
|------|---------------|-----------------|----------|
| Free | 100 | $0 | Basic features, limited |
| Starter | 1,000 | $49 | Ideal for small businesses |
| Pro | 10,000 | $199 | Advanced features, higher volume |
| Enterprise | 100,000+ | Contact us | Custom features, SLA |

## Environment Variables

### Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| SERVER_HOST | Server host | 0.0.0.0 |
| SERVER_PORT | Server port | 8081 |
| ENV | Environment (development/production) | development |

### Database

| Variable | Description | Default |
|----------|-------------|---------|
| DATABASE_URL | PostgreSQL connection URL (required) | - |
| DB_MAX_OPEN_CONNS | Max open connections | 25 |
| DB_MAX_IDLE_CONNS | Max idle connections | 5 |
| DB_CONN_MAX_LIFETIME | Connection max lifetime (e.g., 5m) | 5m |

### Stripe

| Variable | Description | Required |
|----------|-------------|----------|
| STRIPE_SECRET_KEY | Stripe secret key | Yes |
| STRIPE_PUBLISHABLE_KEY | Stripe publishable key | Yes |
| STRIPE_WEBHOOK_SECRET | Webhook secret | Yes |
| STRIPE_API_VERSION | Stripe API version | No |

### Redis

| Variable | Description | Default |
|----------|-------------|---------|
| REDIS_URL | Redis connection URL | redis://localhost:6379 |
| REDIS_HOST | Redis host | localhost |
| REDIS_PORT | Redis port | 6379 |
| REDIS_DB | Redis database | 1 |

### Kafka

| Variable | Description | Default |
|----------|-------------|---------|
| KAFKA_BROKERS | Kafka brokers | localhost:9092 |
| KAFKA_GROUP_ID | Consumer group ID | billing-service |
| KAFKA_TOPICS | Topics for publishing events | billing.events,payments.webhooks |
| KAFKA_DLQ_TOPIC | DLQ topic for failed usage messages | usage.reported.dlq |
| KAFKA_MAX_RETRIES | Retries before sending to DLQ | 3 |
| KAFKA_RETRY_BACKOFF_MS | Backoff (ms) between retries | 2000 |

### Wallet

| Variable | Description | Default |
|----------|-------------|---------|
| WALLET_DEFAULT_CURRENCY | Default currency | BRL |
| WALLET_INITIAL_CREDITS | Initial credits | 100 |
| WALLET_MIN_TOPUP_AMOUNT | Minimum top-up amount | 1000 (cents) |
| WALLET_MAX_TOPUP_AMOUNT | Maximum top-up amount | 1000000 (cents) |

### Pricing

| Variable | Description | Default |
|----------|-------------|---------|
| PRICING_DEFAULT_PLAN | Plan used when an event does not provide a plan id | starter |
| PRICING_PLANS_JSON | JSON map of plan ids to per-dimension pricing in cents; overrides defaults | empty (uses built-ins) |

Defaults baked into the service: `starter` (call 1, minute 2, message 1, api_request 1, storage_gb 5), `professional` (storage_gb 4), and `enterprise` (storage_gb 3). Provide overrides via `PRICING_PLANS_JSON`, for example `{"starter":{"call_cents":1,"minute_cents":2,"message_cents":1,"api_request_cents":1,"storage_gb_cents":5}}`.

### Feature Flags

| Variable | Description | Default |
|----------|-------------|---------|
| ENABLE_CREDIT_WALLET | Enable wallet system | true |
| ENABLE_SUBSCRIPTIONS | Enable subscriptions | true |
| ENABLE_INVOICING | Enable invoicing | false |
| ENABLE_PAYMENT_METHODS | Enable payment methods | true |

## Kafka Events

### Published Events

- `billing.customer.created` - Customer created
- `billing.subscription.created` - Subscription created
- `billing.subscription.updated` - Subscription updated
- `billing.subscription.cancelled` - Subscription cancelled
- `billing.payment.succeeded` - Payment successful
- `billing.payment.failed` - Payment failed
- `billing.invoice.created` - Invoice created
- `billing.wallet.credited` - Credits added to wallet
- `billing.wallet.debited` - Credits debited from wallet

### Consumed Events

- `tenant.created` - New tenant created (create Stripe customer)
- `usage.reported` - Usage reported (debit wallet)

See schema: [docs/usage.reported.schema.json](docs/usage.reported.schema.json)

## Stripe Webhooks

The service processes the following Stripe events:

- `customer.subscription.created` - New subscription created
- `customer.subscription.updated` - Subscription updated
- `customer.subscription.deleted` - Subscription cancelled
- `invoice.payment_succeeded` - Invoice payment successful
- `invoice.payment_failed` - Invoice payment failed
- `customer.created` - Customer created
- `payment_method.attached` - Payment method added
- `payment_method.detached` - Payment method removed

## Development

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- Kafka 3.0+
- Stripe account (test mode)

### Local Setup

1. Clone the repository
2. Copy `.env.example` to `.env`
3. Configure your Stripe keys in `.env`
4. Start dependencies:
   ```bash
   docker-compose up -d postgres redis kafka
   ```
5. Run migrations:
   ```bash
   make migrate-up
   ```
6. Start the server:
   ```bash
   go run cmd/server/main.go
   ```

### Setting up Local Stripe Webhooks

To test webhooks locally, use Stripe CLI:

```bash
# Install Stripe CLI
# https://stripe.com/docs/stripe-cli

# Login
stripe login

# Forward webhooks to your local server
stripe listen --forward-to localhost:8081/webhooks/stripe

# Use the provided webhook secret in .env
```

### Testing

```bash
# Run all tests
make test

# Tests with coverage
make test-coverage

# Integration tests
make test-integration
```

## Data Model

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

## Integration with Other Services

### Tenant Manager

- Receives `tenant.created` events to create Stripe customer
- Notifies about plan/subscription changes

### Voice Gateway

- Consumes wallet credits for voice calls
- Reports usage via Kafka events

### Analytics Service

- Provides billing and usage data
- Generates revenue reports

## Security

- JWT authentication (tokens from auth-gateway)
- Stripe webhook signature validation
- Stripe API keys stored as environment variables
- HTTPS required in production
- Audit logs for financial transactions

## Monitoring

- Prometheus metrics at `/metrics`
- Health checks at `/health`
- Structured logging (JSON)
- Tracing with Jaeger (optional)
- Alerts for:
  - Payment failures
  - Webhook errors
  - Low wallet balance
  - Usage anomalies

## Related Documentation

- [Platform Architecture](../../../docs/architecture/README.md)
- [Pricing Strategy](../../../infra/PRICING_STRATEGY_AWS.md)
- [Cost Estimation](../../../infra/COST_ESTIMATION.md)
- [Stripe API Documentation](https://stripe.com/docs/api)

## Support and Contribution

To report bugs or request features, open an issue in the repository.

---

**Version**: 1.0.0  
**License**: Proprietary
