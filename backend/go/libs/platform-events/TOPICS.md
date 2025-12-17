# Platform Events - Topics and Payloads

This table maps each topic constant to the expected payload struct. Topics without a struct are marked **Pending** and need a contract definition.

## Auth
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `auth.user.created` | `events.UserCreatedEvent` | Defined | |
| `auth.user.updated` | `events.UserUpdatedEvent` | Defined | |
| `auth.user.deleted` | `events.UserDeletedEvent` | Defined |  |
| `auth.user.logged_in` | `events.UserLoggedInEvent` | Defined |  |
| `auth.user.logged_out` | `events.UserLoggedOutEvent` | Defined |  |
| `auth.password.changed` | `events.PasswordChangedEvent` | Defined |  |
| `auth.password.reset` | `events.PasswordResetEvent` | Defined |  |

## Tenant
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `tenant.created` | `events.TenantCreatedEvent` | Defined | |
| `tenant.updated` | `events.TenantUpdatedEvent` | Defined | |
| `tenant.deleted` | `events.TenantDeletedEvent` | Defined | tenant_id, deleted_at, reason, deleted_by, hard_delete |
| `tenant.suspended` | `events.TenantSuspendedEvent` | Defined | tenant_id, suspended_at, reason, suspended_by, expires_at |
| `tenant.activated` | `events.TenantActivatedEvent` | Defined | tenant_id, activated_at, activated_by, reason |
| `tenant.member.added` | `events.TenantMemberAddedEvent` | Defined | tenant_id, member_id, email, role, added_at, added_by |
| `tenant.member.removed` | `events.TenantMemberRemovedEvent` | Defined | tenant_id, member_id, removed_at, removed_by, reason |

## Billing
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `billing.subscription.created` | `events.SubscriptionCreatedEvent` | Defined | |
| `billing.subscription.updated` | `events.SubscriptionUpdatedEvent` | Defined | subscription_id, tenant_id, changes, updated_at, updated_by |
| `billing.subscription.cancelled` | `events.SubscriptionCancelledEvent` | Defined | subscription_id, tenant_id, reason, cancelled_at, cancelled_by, refunded |
| `billing.payment.succeeded` | `events.PaymentSucceededEvent` | Defined | |
| `billing.payment.failed` | `events.PaymentFailedEvent` | Defined | payment_id, tenant_id, amount_cents, currency, failure_code, failure_reason, failed_at, retryable |
| `billing.credits.purchased` | `events.CreditsPurchasedEvent` | Defined | |
| `billing.credits.consumed` | `events.CreditsConsumedEvent` | Defined | |
| `billing.invoice.generated` | `events.InvoiceGeneratedEvent` | Defined | invoice_id, tenant_id, period_start, period_end, amount_cents, currency, due_date, link, generated_at |

## Agent
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `agent.created` | `events.AgentCreatedEvent` | Defined | agent_id, tenant_id, name, channel, model, created_at |
| `agent.updated` | `events.AgentUpdatedEvent` | Defined | agent_id, tenant_id, changes, updated_at, updated_by |
| `agent.deleted` | `events.AgentDeletedEvent` | Defined | agent_id, tenant_id, deleted_at, deleted_by, reason |
| `agent.deployed` | `events.AgentDeployedEvent` | Defined | agent_id, tenant_id, environment, version, deployed_at |
| `agent.started` | `events.AgentStartedEvent` | Defined | agent_id, tenant_id, started_at, node, cluster |
| `agent.stopped` | `events.AgentStoppedEvent` | Defined | agent_id, tenant_id, stopped_at, reason, code |
| `agent.conversation.started` | `events.ConversationStartedEvent` | Defined | conversation_id, agent_id, tenant_id, channel, started_at |
| `agent.conversation.ended` | `events.ConversationEndedEvent` | Defined | conversation_id, agent_id, tenant_id, ended_at, duration, reason |
| `agent.message.sent` | `events.MessageSentEvent` | Defined | message_id, conversation_id, agent_id, tenant_id, content, sent_at |
| `agent.message.received` | `events.MessageReceivedEvent` | Defined | message_id, conversation_id, agent_id, tenant_id, content, source, received_at |

## Analytics
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `analytics.interaction.logged` | - | Pending | Needs interaction id, tenant id, user/agent id, channel, timestamp, metadata. |
| `analytics.metric.recorded` | - | Pending | Needs metric name, value, dimensions/tags, captured_at. |
| `analytics.report.generated` | - | Pending | Needs report id, tenant id, type, period, generated_at, link. |
| `analytics.data.exported` | - | Pending | Needs export id, tenant id, format, size, requested_by, exported_at, destination. |

## Tooling
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `tool.registered` | - | Pending | Needs tool id, tenant id, name, version, registered_at, registered_by. |
| `tool.invoked` | `events.ToolInvokedEvent` | Defined | |
| `tool.completed` | `events.ToolCompletedEvent` | Defined | |
| `tool.failed` | - | Pending | Needs tool id, tenant id, action, error, duration, failed_at. |

## System
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `system.health.check` | - | Pending | Needs service name, status, checked_at, details. |
| `system.error` | `events.SystemErrorEvent` | Defined | |
| `system.alert` | - | Pending | Needs alert id, severity, service, message, created_at, labels. |
| `system.configuration.updated` | - | Pending | Needs service, config keys changed, updated_by, updated_at. |
