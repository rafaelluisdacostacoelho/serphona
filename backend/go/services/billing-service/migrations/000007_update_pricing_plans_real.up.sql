-- Update pricing plans with real per-dimension values (cents)
-- Starter remains default; others updated but not default
INSERT INTO pricing_plans AS p (
    plan_id, display_name, is_default, call_cents, minute_cents, message_cents, api_request_cents, storage_gb_cents, effective_from
) VALUES
    ('starter', 'Starter', true, 5, 10, 2, 1, 15, CURRENT_TIMESTAMP),
    ('pro', 'Pro', false, 4, 8, 2, 1, 12, CURRENT_TIMESTAMP),
    ('professional', 'Professional', false, 4, 8, 2, 1, 12, CURRENT_TIMESTAMP),
    ('enterprise', 'Enterprise', false, 3, 6, 2, 1, 10, CURRENT_TIMESTAMP)
ON CONFLICT (plan_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    is_default = CASE WHEN EXCLUDED.plan_id = 'starter' THEN true ELSE false END,
    call_cents = EXCLUDED.call_cents,
    minute_cents = EXCLUDED.minute_cents,
    message_cents = EXCLUDED.message_cents,
    api_request_cents = EXCLUDED.api_request_cents,
    storage_gb_cents = EXCLUDED.storage_gb_cents,
    effective_from = CURRENT_TIMESTAMP;

-- Ensure only one default remains
UPDATE pricing_plans SET is_default = false WHERE plan_id <> 'starter' AND is_default = true;
