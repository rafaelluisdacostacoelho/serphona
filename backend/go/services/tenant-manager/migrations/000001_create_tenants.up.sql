-- =============================================================================
-- Migration: 000001_create_tenants
-- Description: Create tenants table and related indexes
-- =============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For text search

-- Create enum types
CREATE TYPE tenant_status AS ENUM ('active', 'suspended', 'pending', 'deleted');
CREATE TYPE tenant_plan AS ENUM ('starter', 'professional', 'enterprise');

-- =============================================================================
-- Tenants Table
-- =============================================================================
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(20),
    status tenant_status NOT NULL DEFAULT 'pending',
    plan tenant_plan NOT NULL DEFAULT 'starter',
    settings JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    stripe_id VARCHAR(100),
    billing_email VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT tenants_name_length CHECK (char_length(name) >= 2),
    CONSTRAINT tenants_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

-- Indexes for common queries
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_plan ON tenants(plan) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_created_at ON tenants(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tenants_stripe_id ON tenants(stripe_id) WHERE stripe_id IS NOT NULL;

-- GIN index for full-text search on name and email
CREATE INDEX idx_tenants_search ON tenants USING gin (
    (name || ' ' || email) gin_trgm_ops
) WHERE deleted_at IS NULL;

-- GIN index for JSONB settings queries
CREATE INDEX idx_tenants_settings ON tenants USING gin (settings);

-- =============================================================================
-- Tenant Quotas Table
-- =============================================================================
CREATE TABLE tenant_quotas (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    max_api_keys INTEGER NOT NULL DEFAULT 10,
    max_users INTEGER NOT NULL DEFAULT 5,
    max_calls_per_month INTEGER NOT NULL DEFAULT 1000,
    max_minutes_per_month INTEGER NOT NULL DEFAULT 5000,
    max_storage_gb INTEGER NOT NULL DEFAULT 10,
    used_calls INTEGER NOT NULL DEFAULT 0,
    used_minutes INTEGER NOT NULL DEFAULT 0,
    used_storage_gb NUMERIC(10,2) NOT NULL DEFAULT 0,
    reset_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (date_trunc('month', NOW()) + INTERVAL '1 month'),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- API Keys Table
-- =============================================================================
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE, -- SHA-256 hash of the key
    key_prefix VARCHAR(8) NOT NULL, -- First 8 chars for identification
    scopes TEXT[] NOT NULL DEFAULT '{}',
    rate_limit INTEGER NOT NULL DEFAULT 1000, -- requests per minute
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT api_keys_name_length CHECK (char_length(name) >= 1)
);

-- Indexes
CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at) WHERE expires_at IS NOT NULL AND revoked_at IS NULL;

-- =============================================================================
-- Tenant Usage History Table (for billing and analytics)
-- =============================================================================
CREATE TABLE tenant_usage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period VARCHAR(7) NOT NULL, -- YYYY-MM format
    total_calls INTEGER NOT NULL DEFAULT 0,
    total_minutes INTEGER NOT NULL DEFAULT 0,
    total_messages INTEGER NOT NULL DEFAULT 0,
    storage_used_gb NUMERIC(10,2) NOT NULL DEFAULT 0,
    api_requests BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Unique constraint for tenant + period
    CONSTRAINT tenant_usage_history_unique UNIQUE (tenant_id, period)
);

CREATE INDEX idx_tenant_usage_history_period ON tenant_usage_history(period DESC);

-- =============================================================================
-- Audit Log Table
-- =============================================================================
CREATE TABLE tenant_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_id VARCHAR(100), -- User ID or system identifier
    actor_type VARCHAR(20) NOT NULL DEFAULT 'user', -- user, system, api
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(100),
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(36),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for audit queries
CREATE INDEX idx_tenant_audit_log_tenant_id ON tenant_audit_log(tenant_id);
CREATE INDEX idx_tenant_audit_log_created_at ON tenant_audit_log(created_at DESC);
CREATE INDEX idx_tenant_audit_log_action ON tenant_audit_log(action);
CREATE INDEX idx_tenant_audit_log_request_id ON tenant_audit_log(request_id) WHERE request_id IS NOT NULL;

-- =============================================================================
-- Functions and Triggers
-- =============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for tenants table
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger for tenant_quotas table
CREATE TRIGGER update_tenant_quotas_updated_at
    BEFORE UPDATE ON tenant_quotas
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to create default quota on tenant creation
CREATE OR REPLACE FUNCTION create_default_tenant_quota()
RETURNS TRIGGER AS $$
DECLARE
    quota_calls INTEGER;
    quota_minutes INTEGER;
    quota_storage INTEGER;
    quota_users INTEGER;
    quota_api_keys INTEGER;
BEGIN
    -- Set quotas based on plan
    CASE NEW.plan
        WHEN 'starter' THEN
            quota_calls := 1000;
            quota_minutes := 5000;
            quota_storage := 10;
            quota_users := 5;
            quota_api_keys := 5;
        WHEN 'professional' THEN
            quota_calls := 10000;
            quota_minutes := 50000;
            quota_storage := 100;
            quota_users := 25;
            quota_api_keys := 20;
        WHEN 'enterprise' THEN
            quota_calls := 100000;
            quota_minutes := 500000;
            quota_storage := 1000;
            quota_users := 100;
            quota_api_keys := 100;
        ELSE
            quota_calls := 1000;
            quota_minutes := 5000;
            quota_storage := 10;
            quota_users := 5;
            quota_api_keys := 5;
    END CASE;

    INSERT INTO tenant_quotas (
        tenant_id, max_api_keys, max_users, max_calls_per_month,
        max_minutes_per_month, max_storage_gb
    ) VALUES (
        NEW.id, quota_api_keys, quota_users, quota_calls,
        quota_minutes, quota_storage
    );
    
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to create quota on tenant insert
CREATE TRIGGER create_tenant_quota_on_insert
    AFTER INSERT ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION create_default_tenant_quota();

-- Function to generate URL-friendly slug from name
CREATE OR REPLACE FUNCTION generate_tenant_slug()
RETURNS TRIGGER AS $$
DECLARE
    base_slug TEXT;
    final_slug TEXT;
    counter INTEGER := 0;
BEGIN
    -- Generate base slug from name
    base_slug := lower(regexp_replace(NEW.name, '[^a-zA-Z0-9]+', '-', 'g'));
    base_slug := trim(both '-' from base_slug);
    final_slug := base_slug;
    
    -- Check for uniqueness and append counter if needed
    WHILE EXISTS (SELECT 1 FROM tenants WHERE slug = final_slug AND id != COALESCE(NEW.id, uuid_nil())) LOOP
        counter := counter + 1;
        final_slug := base_slug || '-' || counter;
    END LOOP;
    
    NEW.slug := final_slug;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to auto-generate slug
CREATE TRIGGER generate_tenant_slug_on_insert
    BEFORE INSERT ON tenants
    FOR EACH ROW
    WHEN (NEW.slug IS NULL OR NEW.slug = '')
    EXECUTE FUNCTION generate_tenant_slug();

-- =============================================================================
-- Row Level Security (RLS) - Multi-tenant isolation
-- =============================================================================

-- Enable RLS on tables
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_quotas ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_usage_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_audit_log ENABLE ROW LEVEL SECURITY;

-- Policy for service accounts (bypass RLS)
CREATE POLICY service_account_all ON tenants
    FOR ALL
    TO service_account
    USING (true)
    WITH CHECK (true);

-- Policies for tenant isolation (application-level)
-- These will use the current tenant_id set via SET LOCAL
CREATE POLICY tenant_isolation ON tenants
    FOR ALL
    TO application
    USING (id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_quotas ON tenant_quotas
    FOR ALL
    TO application
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_api_keys ON api_keys
    FOR ALL
    TO application
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- =============================================================================
-- Comments
-- =============================================================================
COMMENT ON TABLE tenants IS 'Multi-tenant organizations using the Voice of Customer platform';
COMMENT ON TABLE tenant_quotas IS 'Usage quotas and limits for each tenant';
COMMENT ON TABLE api_keys IS 'API keys for programmatic access to tenant resources';
COMMENT ON TABLE tenant_usage_history IS 'Historical usage data for billing and analytics';
COMMENT ON TABLE tenant_audit_log IS 'Audit trail for tenant resource changes';

COMMENT ON COLUMN tenants.settings IS 'JSONB containing telephony, AI agent, notification, and security settings';
COMMENT ON COLUMN tenants.metadata IS 'JSONB containing industry, company size, website, address, and custom fields';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA-256 hash of the API key for secure storage';
COMMENT ON COLUMN api_keys.key_prefix IS 'First 8 characters of the key for identification without exposing the full key';
