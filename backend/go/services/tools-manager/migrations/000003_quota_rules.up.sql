-- Quota rules for rate/usage limits with deterministic precedence
CREATE TABLE IF NOT EXISTS quota_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    tool_id UUID REFERENCES tools(id) ON DELETE CASCADE,
    agent_id TEXT,
    env TEXT,
    limit_per_minute INTEGER CHECK (limit_per_minute >= 0),
    limit_per_day INTEGER CHECK (limit_per_day >= 0),
    weight INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quota_rules_tenant ON quota_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_quota_rules_tool ON quota_rules(tool_id);
CREATE INDEX IF NOT EXISTS idx_quota_rules_weight ON quota_rules(weight DESC);

CREATE TRIGGER trg_quota_rules_updated_at BEFORE UPDATE ON quota_rules
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- RLS for quota_rules
ALTER TABLE quota_rules ENABLE ROW LEVEL SECURITY;

CREATE POLICY quota_rules_service_accounts_all ON quota_rules
    FOR ALL
    TO service_account
    USING (TRUE)
    WITH CHECK (TRUE);

CREATE POLICY quota_rules_isolation ON quota_rules
    FOR ALL
    TO application
    USING (
        current_setting('app.current_tenant_id', TRUE) = 'platform'
        OR (
            current_setting('app.current_tenant_id', TRUE) IS NOT NULL
            AND tenant_id = current_setting('app.current_tenant_id')::UUID
        )
    )
    WITH CHECK (
        current_setting('app.current_tenant_id', TRUE) = 'platform'
        OR (
            current_setting('app.current_tenant_id', TRUE) IS NOT NULL
            AND tenant_id = current_setting('app.current_tenant_id')::UUID
        )
    );
