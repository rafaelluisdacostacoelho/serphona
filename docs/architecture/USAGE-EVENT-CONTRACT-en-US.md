# Usage Event Contract (usage.reported)

Purpose: align tenant-manager emitters with billing-service consumers for usage and quota/overage handling.

## Topic and Routing
- Kafka topic: `usage.reported`
- Key: `tenant_id` (string UUID); ensures per-tenant ordering
- Value format: JSON (below); future Avro/Protobuf allowed if schema registry is introduced

## Payload
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| tenant_id | string (UUID) | yes | Tenant identifier; also the partition key |
| period | string (YYYY-MM) | yes | Billing/usage bucket (UTC) |
| occurred_at | string (RFC3339) | yes | When the usage occurred (UTC) |
| source | string | yes | Producer of the usage (e.g., `tenant-manager`, `voice-gateway`, `analytics-processor`) |
| calls | integer >= 0 | yes | Delta calls for this message |
| minutes | integer >= 0 | yes | Delta minutes for this message |
| messages | integer >= 0 | yes | Delta messages (if applicable) |
| storage_gb | number >= 0 | yes | Delta storage GB (can be fractional) |
| api_requests | integer >= 0 | yes | Delta API requests |
| plan | string | no | Plan code at time of usage |
| subscription_id | string | no | Billing subscription ID (if available) |
| request_id | string | yes | Idempotency key for this usage record |
| trace_id | string | no | Trace/span correlation |

Notes:
- All counters are **deltas**, never totals.
- Emit one message per logical operation that consumes quota; deduplicate on `request_id`.
- Producer must never send negative deltas.

## Example
```json
{
  "tenant_id": "9c1c3e79-7e5d-4e30-8a6e-4f3bb7cb4c0f",
  "period": "2025-12",
  "occurred_at": "2025-12-31T23:59:12Z",
  "source": "tenant-manager",
  "calls": 2,
  "minutes": 6,
  "messages": 0,
  "storage_gb": 0.0,
  "api_requests": 3,
  "plan": "starter",
  "subscription_id": "sub_123",
  "request_id": "req-7f5f8d1e-321e-4c71-9a3f-62b6f0e9c111",
  "trace_id": "2a4b5c6d7e8f9a0b"
}
```

## Reset and Period Semantics
- Billing buckets are monthly, keyed by `period=YYYY-MM` in UTC.
- Quota reset happens at the boundary of the next period; producers must set `period` based on occurrence time, not send time.

## Validation Rules (consumer expectations)
- Reject/ignore messages with missing `tenant_id`, `period`, `occurred_at`, or negative deltas.
- Idempotency: duplicate `request_id` for the same `tenant_id` should be treated as a no-op.

## Open Questions to Close
- Should `messages`/`api_requests` be optional per plan? (default is zero allowed)
- Do we need per-product dimensions (e.g., `transcription_seconds`, `storage_write_gb`)?
- Should overage be billed immediately per message or batched per period?
