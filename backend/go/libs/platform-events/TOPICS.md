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
| `tenant.deleted` | - | Pending | Define fields (tenant id, deleted_by, reason, deleted_at). |
| `tenant.suspended` | - | Pending | Define fields (tenant id, reason, suspended_by, suspended_at, until). |
| `tenant.activated` | - | Pending | Define fields (tenant id, activated_by, activated_at). |
| `tenant.member.added` | - | Pending | Needs tenant id, user id, role, added_by, added_at. |
| `tenant.member.removed` | - | Pending | Needs tenant id, user id, removed_by, reason, removed_at. |

## Billing
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `billing.subscription.created` | `events.SubscriptionCreatedEvent` | Defined | |
| `billing.subscription.updated` | - | Pending | Define fields (subscription id, tenant id, changes, updated_at, updated_by). |
| `billing.subscription.cancelled` | - | Pending | Needs subscription id, tenant id, reason, cancelled_at, cancelled_by. |
| `billing.payment.succeeded` | `events.PaymentSucceededEvent` | Defined | |
| `billing.payment.failed` | - | Pending | Needs payment id, tenant id, amount, currency, failure reason, failed_at. |
| `billing.credits.purchased` | `events.CreditsPurchasedEvent` | Defined | |
| `billing.credits.consumed` | `events.CreditsConsumedEvent` | Defined | |
| `billing.invoice.generated` | - | Pending | Needs invoice id, tenant id, period, amount, currency, due date, link. |

## Agent
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `agent.created` | `events.AgentCreatedEvent` | Defined | |
| `agent.updated` | - | Pending | Needs agent id, tenant id, changes, updated_at, updated_by. |
| `agent.deleted` | - | Pending | Needs agent id, tenant id, deleted_at, deleted_by. |
| `agent.deployed` | - | Pending | Needs agent id, tenant id, environment, version, deployed_at. |
| `agent.started` | - | Pending | Needs agent id, tenant id, started_at, node/host. |
| `agent.stopped` | - | Pending | Needs agent id, tenant id, stopped_at, reason. |
| `agent.conversation.started` | `events.ConversationStartedEvent` | Defined | |
| `agent.conversation.ended` | `events.ConversationEndedEvent` | Defined | |
| `agent.message.sent` | `events.MessageSentEvent` | Defined | |
| `agent.message.received` | - | Pending | Likely similar to sent, plus source (customer/channel). |

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
