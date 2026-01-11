-- Drop policies
DROP POLICY IF EXISTS tenant_tools_isolation ON tenant_tools;
DROP POLICY IF EXISTS tenant_tools_service_accounts_all ON tenant_tools;

-- Disable RLS
ALTER TABLE IF EXISTS tenant_tools DISABLE ROW LEVEL SECURITY;

-- Drop triggers
DROP TRIGGER IF EXISTS trg_tenant_tools_updated_at ON tenant_tools;
DROP TRIGGER IF EXISTS trg_tool_versions_updated_at ON tool_versions;
DROP TRIGGER IF EXISTS trg_tools_updated_at ON tools;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column;

-- Drop tables (reverse dependency order)
DROP TABLE IF EXISTS tenant_tools;
DROP TABLE IF EXISTS tool_versions;
DROP TABLE IF EXISTS tools;
