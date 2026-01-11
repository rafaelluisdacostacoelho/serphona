-- Policy rules for allow/deny with deterministic precedence
CREATE TABLE IF NOT EXISTS policy_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    tool_id UUID REFERENCES tools(id) ON DELETE CASCADE,
    agent_id TEXT,
    env TEXT,
    effect TEXT NOT NULL CHECK (effect IN ('allow','deny')),
    scopes TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    roles TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    weight INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_policy_rules_tenant ON policy_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_policy_rules_tool ON policy_rules(tool_id);
CREATE INDEX IF NOT EXISTS idx_policy_rules_effect ON policy_rules(effect);
CREATE INDEX IF NOT EXISTS idx_policy_rules_weight ON policy_rules(weight DESC);

CREATE TRIGGER trg_policy_rules_updated_at BEFORE UPDATE ON policy_rules
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- RLS for policy_rules
ALTER TABLE policy_rules ENABLE ROW LEVEL SECURITY;

CREATE POLICY policy_rules_service_accounts_all ON policy_rules
    FOR ALL
    TO service_account
    USING (TRUE)
    WITH CHECK (TRUE);

CREATE POLICY policy_rules_isolation ON policy_rules
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
