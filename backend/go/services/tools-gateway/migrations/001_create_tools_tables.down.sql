-- Drop triggers
DROP TRIGGER IF EXISTS update_tenant_tools_updated_at ON tenant_tools;
DROP TRIGGER IF EXISTS update_tools_updated_at ON tools;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (in reverse order due to foreign keys)
DROP TABLE IF EXISTS tool_executions;
DROP TABLE IF EXISTS tenant_tools;
DROP TABLE IF EXISTS tools;
