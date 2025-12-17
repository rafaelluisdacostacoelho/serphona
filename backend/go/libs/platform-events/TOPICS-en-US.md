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
| `analytics.interaction.logged` | `events.InteractionLoggedEvent` | Defined | interaction_id, tenant_id, user/agent, channel, logged_at, metadata |
| `analytics.metric.recorded` | `events.MetricRecordedEvent` | Defined | metric, value, unit, dimensions, captured_at |
| `analytics.report.generated` | `events.ReportGeneratedEvent` | Defined | report_id, report_type, period_start/end, generated_at, link |
| `analytics.data.exported` | `events.DataExportedEvent` | Defined | export_id, format, destination, size_bytes, exported_at |

## Tooling
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `tool.registered` | `events.ToolRegisteredEvent` | Defined | tool_id, tenant_id, name, version, registered_at |
| `tool.invoked` | `events.ToolInvokedEvent` | Defined | tool_id, tenant_id, action, invoked_at, correlation_id?, payload? |
| `tool.completed` | `events.ToolCompletedEvent` | Defined | tool_id, tenant_id, action, result, duration_ms?, completed_at, correlation_id? |
| `tool.failed` | `events.ToolFailedEvent` | Defined | tool_id, tenant_id, action, error, duration_ms, failed_at |

## System
| Topic | Payload struct | Status | Notes |
| --- | --- | --- | --- |
| `system.health.check` | `events.SystemHealthCheckEvent` | Defined | service, status, checked_at, details |
| `system.error` | `events.SystemErrorEvent` | Defined | service, error, severity?, occurred_at, trace_id?, span_id?, labels? |
| `system.alert` | `events.SystemAlertEvent` | Defined | alert_id, severity, service, message, labels |
| `system.configuration.updated` | `events.ConfigurationUpdatedEvent` | Defined | service, changes, updated_by, updated_at |
