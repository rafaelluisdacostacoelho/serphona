-- Create pricing_plans table to store per-plan pricing in cents
CREATE TABLE IF NOT EXISTS pricing_plans (
    plan_id VARCHAR(100) PRIMARY KEY,
    display_name VARCHAR(255),
    is_default BOOLEAN NOT NULL DEFAULT false,
    call_cents BIGINT NOT NULL DEFAULT 0 CHECK (call_cents >= 0),
    minute_cents BIGINT NOT NULL DEFAULT 0 CHECK (minute_cents >= 0),
    message_cents BIGINT NOT NULL DEFAULT 0 CHECK (message_cents >= 0),
    api_request_cents BIGINT NOT NULL DEFAULT 0 CHECK (api_request_cents >= 0),
    storage_gb_cents BIGINT NOT NULL DEFAULT 0 CHECK (storage_gb_cents >= 0),
    effective_from TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Enforce a single default plan
CREATE UNIQUE INDEX IF NOT EXISTS idx_pricing_plans_default ON pricing_plans((is_default)) WHERE is_default = true;

-- Optimize lookups by effective date
CREATE INDEX IF NOT EXISTS idx_pricing_plans_effective_from ON pricing_plans(effective_from DESC);

-- Seed baseline plans (can be overridden via future migrations or admin UI)
INSERT INTO pricing_plans (
    plan_id, display_name, is_default, call_cents, minute_cents, message_cents, api_request_cents, storage_gb_cents
) VALUES
    ('starter', 'Starter', true, 1, 2, 1, 1, 5),
    ('professional', 'Professional', false, 1, 2, 1, 1, 4),
    ('enterprise', 'Enterprise', false, 1, 2, 1, 1, 3)
ON CONFLICT (plan_id) DO NOTHING;

-- Add comments
COMMENT ON TABLE pricing_plans IS 'Per-plan pricing in cents for usage dimensions';
COMMENT ON COLUMN pricing_plans.plan_id IS 'Plan identifier used by subscriptions and events';
COMMENT ON COLUMN pricing_plans.is_default IS 'Whether this plan is the default when none is provided';
COMMENT ON COLUMN pricing_plans.effective_from IS 'When this pricing becomes active';
