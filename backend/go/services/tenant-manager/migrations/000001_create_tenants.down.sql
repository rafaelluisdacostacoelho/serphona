-- =============================================================================
-- Down Migration: Drop tenant-manager resources
-- =============================================================================

DROP TRIGGER IF EXISTS generate_tenant_slug_on_insert ON tenants;
DROP FUNCTION IF EXISTS generate_tenant_slug();

DROP TRIGGER IF EXISTS create_tenant_quota_on_insert ON tenants;
DROP FUNCTION IF EXISTS create_default_tenant_quota();

DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TRIGGER IF EXISTS update_tenant_quotas_updated_at ON tenant_quotas;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS tenant_audit_log;
DROP TABLE IF EXISTS tenant_usage_history;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS tenant_quotas;
DROP TABLE IF EXISTS tenants;

DROP TYPE IF EXISTS tenant_status;
DROP TYPE IF EXISTS tenant_plan;
