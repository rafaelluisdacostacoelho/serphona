# Platform Events - Tópicos e Payloads

Tabela que relaciona cada tópico ao struct de payload esperado. Tópicos sem struct estão marcados como **Pendente** e precisam de contrato.

## Auth
| Tópico | Struct de payload | Status | Notas |
| --- | --- | --- | --- |
| `auth.user.created` | `events.UserCreatedEvent` | Definido | |
| `auth.user.updated` | `events.UserUpdatedEvent` | Definido | |
| `auth.user.deleted` | `events.UserDeletedEvent` | Definido |  |
| `auth.user.logged_in` | `events.UserLoggedInEvent` | Definido |  |
| `auth.user.logged_out` | `events.UserLoggedOutEvent` | Definido |  |
| `auth.password.changed` | `events.PasswordChangedEvent` | Definido |  |
| `auth.password.reset` | `events.PasswordResetEvent` | Definido |  |

## Tenant
| Tópico | Struct de payload | Status | Notas |
| --- | --- | --- | --- |
| `tenant.created` | `events.TenantCreatedEvent` | Definido | |
| `tenant.updated` | `events.TenantUpdatedEvent` | Definido | |
| `tenant.deleted` | `events.TenantDeletedEvent` | Definido | tenant_id, deleted_at, reason, deleted_by, hard_delete |
| `tenant.suspended` | `events.TenantSuspendedEvent` | Definido | tenant_id, suspended_at, reason, suspended_by, expires_at |
| `tenant.activated` | `events.TenantActivatedEvent` | Definido | tenant_id, activated_at, activated_by, reason |
| `tenant.member.added` | `events.TenantMemberAddedEvent` | Definido | tenant_id, member_id, email, role, added_at, added_by |
| `tenant.member.removed` | `events.TenantMemberRemovedEvent` | Definido | tenant_id, member_id, removed_at, removed_by, reason |

## Billing
| Tópico | Struct de payload | Status | Notas |
| --- | --- | --- | --- |
| `billing.subscription.created` | `events.SubscriptionCreatedEvent` | Definido | |
| `billing.subscription.updated` | `events.SubscriptionUpdatedEvent` | Definido | subscription_id, tenant_id, changes, updated_at, updated_by |
| `billing.subscription.cancelled` | `events.SubscriptionCancelledEvent` | Definido | subscription_id, tenant_id, reason, cancelled_at, cancelled_by, refunded |
| `billing.payment.succeeded` | `events.PaymentSucceededEvent` | Definido | |
| `billing.payment.failed` | `events.PaymentFailedEvent` | Definido | payment_id, tenant_id, amount_cents, currency, failure_code, failure_reason, failed_at, retryable |
| `billing.credits.purchased` | `events.CreditsPurchasedEvent` | Definido | |
| `billing.credits.consumed` | `events.CreditsConsumedEvent` | Definido | |
| `billing.invoice.generated` | `events.InvoiceGeneratedEvent` | Definido | invoice_id, tenant_id, period_start, period_end, amount_cents, currency, due_date, link, generated_at |

## Agent
| Tópico | Struct de payload | Status | Notas |
| --- | --- | --- | --- |
| `agent.created` | `events.AgentCreatedEvent` | Definido | agent_id, tenant_id, name, channel, model, created_at |
| `agent.updated` | `events.AgentUpdatedEvent` | Definido | agent_id, tenant_id, changes, updated_at, updated_by |
| `agent.deleted` | `events.AgentDeletedEvent` | Definido | agent_id, tenant_id, deleted_at, deleted_by, reason |
| `agent.deployed` | `events.AgentDeployedEvent` | Definido | agent_id, tenant_id, environment, version, deployed_at |
| `agent.started` | `events.AgentStartedEvent` | Definido | agent_id, tenant_id, started_at, node, cluster |
| `agent.stopped` | `events.AgentStoppedEvent` | Definido | agent_id, tenant_id, stopped_at, reason, code |
| `agent.conversation.started` | `events.ConversationStartedEvent` | Definido | conversation_id, agent_id, tenant_id, channel, started_at |
| `agent.conversation.ended` | `events.ConversationEndedEvent` | Definido | conversation_id, agent_id, tenant_id, ended_at, duration, reason |
| `agent.message.sent` | `events.MessageSentEvent` | Definido | message_id, conversation_id, agent_id, tenant_id, content, sent_at |
| `agent.message.received` | `events.MessageReceivedEvent` | Definido | message_id, conversation_id, agent_id, tenant_id, content, source, received_at |

## Analytics
| Tópico | Struct de payload | Status | Notas |
| --- | --- | --- | --- |
| `analytics.interaction.logged` | - | Pendente | Needs interaction id, tenant id, user/agent id, channel, timestamp, metadata. |
| `analytics.metric.recorded` | - | Pendente | Needs metric name, value, dimensions/tags, captured_at. |
| `analytics.report.generated` | - | Pendente | Needs report id, tenant id, type, period, generated_at, link. |
| `analytics.data.exported` | - | Pendente | Needs export id, tenant id, format, size, requested_by, exported_at, destination. |

## Tooling
| Tópico | Struct de payload | Status | Notas |
| --- | --- | --- | --- |
| `tool.registered` | - | Pendente | Needs tool id, tenant id, name, version, registered_at, registered_by. |
| `tool.invoked` | `events.ToolInvokedEvent` | Definido | |
| `tool.completed` | `events.ToolCompletedEvent` | Definido | |
| `tool.failed` | - | Pendente | Needs tool id, tenant id, action, error, duration, failed_at. |

## System
| Tópico | Struct de payload | Status | Notas |
| --- | --- | --- | --- |
| `system.health.check` | - | Pendente | Needs service name, status, checked_at, details. |
| `system.error` | `events.SystemErrorEvent` | Definido | |
| `system.alert` | - | Pendente | Needs alert id, severity, service, message, created_at, labels. |
| `system.configuration.updated` | - | Pendente | Needs service, config keys changed, updated_by, updated_at. |
