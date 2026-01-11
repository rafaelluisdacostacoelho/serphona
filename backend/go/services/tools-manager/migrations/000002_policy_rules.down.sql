-- Drop RLS policies
DROP POLICY IF EXISTS policy_rules_isolation ON policy_rules;
DROP POLICY IF EXISTS policy_rules_service_accounts_all ON policy_rules;

-- Disable RLS
ALTER TABLE IF EXISTS policy_rules DISABLE ROW LEVEL SECURITY;

-- Drop trigger
DROP TRIGGER IF EXISTS trg_policy_rules_updated_at ON policy_rules;

-- Drop table
DROP TABLE IF EXISTS policy_rules;
