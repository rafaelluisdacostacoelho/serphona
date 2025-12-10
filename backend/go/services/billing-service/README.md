# Billing Service

Microservice responsible for managing billing, subscriptions, invoices, and credit wallets, integrating with Stripe for payment processing.

[🇧🇷 Versão em Português](./README-pt-BR.md) | [🇺🇸 English Version](./README-en-US.md)

## Overview

The Billing Service handles all financial operations for the Serphona platform, including:

- **Stripe Integration**: Complete integration with Stripe API for payment processing
- **Customer Management**: Links Stripe customers with platform tenants
- **Subscription Management**: Create, update, and cancel subscriptions
- **Invoice Management**: Track and manage customer invoices
- **Credit Wallet System**: Prepaid credits for platform usage
- **Webhook Processing**: Handles Stripe webhook events
- **Billing Portal**: Self-service portal for customers
- **Usage Tracking**: Monitors and tracks resource consumption

## Key Features

### 🏦 Payment Processing
- Stripe integration for secure payment processing
- Multiple payment methods support
- Automatic invoice generation
- Payment retry logic for failed transactions

### 💳 Subscription Management
- Multiple subscription plans (Free, Starter, Pro, Enterprise)
- Plan upgrades and downgrades
- Prorated billing
- Cancel at period end support

### 💰 Credit Wallet
- Prepaid credit system
- Real-time balance tracking
- Transaction history
- Automatic debit for usage

### 📊 Usage Tracking
- Real-time usage monitoring
- Quota enforcement
- Usage-based billing
- Detailed usage reports

### 🔔 Event Publishing
- Kafka events for billing lifecycle
- Integration with other services
- Audit trail for financial transactions

## Architecture

This service follows **Hexagonal Architecture** (Ports and Adapters) with clean separation between:
- **Domain Layer**: Business logic and entities
- **Application Layer**: Use cases and orchestration
- **Adapter Layer**: External integrations (HTTP, Stripe, Database, Kafka)

## Quick Start

```bash
# Copy environment file
cp .env.example .env

# Configure Stripe keys in .env
# STRIPE_SECRET_KEY=sk_test_...
# STRIPE_WEBHOOK_SECRET=whsec_...

# Start with Docker Compose
docker-compose up -d

# Or run locally
go run cmd/server/main.go
```

## API Endpoints

### Core Operations
- `POST /api/v1/customers` - Create customer
- `POST /api/v1/subscriptions` - Create subscription
- `GET /api/v1/subscriptions` - List subscriptions
- `PUT /api/v1/subscriptions/{id}` - Update subscription
- `DELETE /api/v1/subscriptions/{id}` - Cancel subscription

### Wallet Operations
- `GET /api/v1/wallet` - Get wallet balance
- `POST /api/v1/wallet/topup` - Add credits
- `GET /api/v1/usage` - Get usage stats

### Portal & Checkout
- `POST /api/v1/portal-session` - Create Billing Portal session
- `POST /api/v1/checkout-session` - Create Checkout session

### Webhooks
- `POST /webhooks/stripe` - Stripe webhook endpoint

## Subscription Plans

| Plan | Credits/Month | Features |
|------|---------------|----------|
| **Free** | 100 | Basic features, limited usage |
| **Starter** | 1,000 | Ideal for small businesses |
| **Pro** | 10,000 | Advanced features, higher volume |
| **Enterprise** | 100,000+ | Custom features, SLA, dedicated support |

## Environment Variables

### Required
- `STRIPE_SECRET_KEY` - Stripe secret key
- `STRIPE_WEBHOOK_SECRET` - Webhook signing secret
- `DATABASE_URL` - PostgreSQL connection string

### Optional
- `SERVER_PORT` - HTTP server port (default: 8081)
- `REDIS_URL` - Redis for caching
- `KAFKA_BROKERS` - Event publishing
- `ENABLE_CREDIT_WALLET` - Enable wallet feature (default: true)

## Development

### Prerequisites
- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Stripe account (test mode)

### Local Setup

```bash
# Install dependencies
go mod download

# Run migrations
make migrate-up

# Start server
go run cmd/server/main.go

# Run tests
make test
```

### Stripe Webhook Testing

```bash
# Install Stripe CLI
stripe login

# Forward webhooks to local server
stripe listen --forward-to localhost:8081/webhooks/stripe
```

## Integration

### With Tenant Manager
- Creates Stripe customer when tenant is created
- Syncs subscription status with tenant plan

### With Voice Gateway
- Debits wallet credits for voice calls
- Enforces usage quotas

### With Analytics Service
- Provides billing data for reports
- Tracks revenue metrics

## Monitoring

- **Metrics**: Prometheus metrics at `/metrics`
- **Health**: Health check at `/health`
- **Logging**: Structured JSON logs
- **Alerts**: Payment failures, webhook errors, low balance

## Security

- JWT authentication via auth-gateway
- Stripe webhook signature validation
- HTTPS required in production
- Audit logs for all financial transactions
- PCI DSS compliance (via Stripe)

## Documentation

For detailed documentation, please refer to:
- [Portuguese Documentation](./README-pt-BR.md) - Comprehensive guide in Portuguese
- [English Documentation](./README-en-US.md) - Comprehensive guide in English
- [Stripe API Docs](https://stripe.com/docs/api)
- [Platform Architecture](../../../docs/architecture/README.md)

## Events

### Published Events
- `billing.customer.created`
- `billing.subscription.created`
- `billing.subscription.updated`
- `billing.subscription.cancelled`
- `billing.payment.succeeded`
- `billing.payment.failed`
- `billing.wallet.credited`
- `billing.wallet.debited`

### Consumed Events
- `tenant.created` - Creates Stripe customer
- `usage.reported` - Debits wallet

## Support

For issues or questions:
- Open an issue in the repository
- Contact the platform team
- Check documentation for common problems

---

**Version**: 1.0.0  
**License**: Proprietary  
**Maintained by**: Serphona Platform Team
