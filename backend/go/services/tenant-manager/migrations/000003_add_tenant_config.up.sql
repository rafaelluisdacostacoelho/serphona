-- =============================================================================
-- Migration: 000003_add_tenant_config
-- Description: Add tenant_configs table for typed configuration blobs
-- =============================================================================

CREATE TABLE IF NOT EXISTS tenant_configs (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    config_type TEXT NOT NULL,
    data JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_by UUID,
    PRIMARY KEY (tenant_id, config_type)
);

CREATE INDEX IF NOT EXISTS idx_tenant_configs_type ON tenant_configs(config_type);
