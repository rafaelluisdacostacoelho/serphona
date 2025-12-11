-- Drop trigger
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_agents_name;
DROP INDEX IF EXISTS idx_agents_tenant_id_is_active;
DROP INDEX IF EXISTS idx_agents_is_active;
DROP INDEX IF EXISTS idx_agents_tenant_id;

-- Drop table
DROP TABLE IF EXISTS agents;
