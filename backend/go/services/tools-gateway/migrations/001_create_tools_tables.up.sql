-- Create tools table
CREATE TABLE IF NOT EXISTS tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    
    -- HTTP Configuration
    method VARCHAR(10) NOT NULL,
    base_url TEXT NOT NULL,
    endpoint_path TEXT NOT NULL,
    headers JSONB DEFAULT '{}',
    
    -- Authentication
    auth_type VARCHAR(50) NOT NULL,
    auth_config JSONB DEFAULT '{}',
    
    -- Schemas
    input_schema JSONB NOT NULL,
    output_schema JSONB NOT NULL,
    
    -- Configuration
    timeout_seconds INTEGER DEFAULT 30,
    max_retries INTEGER DEFAULT 3,
    retry_delay_seconds INTEGER DEFAULT 1,
    
    -- Rate Limiting
    rate_limit_per_minute INTEGER DEFAULT 60,
    rate_limit_per_hour INTEGER DEFAULT 1000,
    
    -- Billing
    credit_cost INTEGER DEFAULT 1,
    
    -- Metadata
    is_active BOOLEAN DEFAULT true,
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,
    
    CONSTRAINT tools_name_key UNIQUE (name)
);

-- Create indexes for tools
CREATE INDEX IF NOT EXISTS idx_tools_category ON tools(category);
CREATE INDEX IF NOT EXISTS idx_tools_is_active ON tools(is_active);
CREATE INDEX IF NOT EXISTS idx_tools_is_public ON tools(is_public);

-- Create tenant_tools table
CREATE TABLE IF NOT EXISTS tenant_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    tool_id UUID NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    
    -- Override configurations
    custom_auth_config JSONB,
    custom_rate_limit_per_minute INTEGER,
    custom_credit_cost INTEGER,
    
    -- Permissions
    is_enabled BOOLEAN DEFAULT true,
    allowed_user_ids UUID[],
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT tenant_tools_tenant_tool_key UNIQUE (tenant_id, tool_id)
);

-- Create indexes for tenant_tools
CREATE INDEX IF NOT EXISTS idx_tenant_tools_tenant_id ON tenant_tools(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_tools_tool_id ON tenant_tools(tool_id);

-- Create tool_executions table
CREATE TABLE IF NOT EXISTS tool_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    tool_id UUID NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    
    -- Request/Response
    input_data JSONB NOT NULL,
    output_data JSONB,
    
    -- Execution Details
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    latency_ms INTEGER,
    
    -- Billing
    credits_consumed INTEGER DEFAULT 0,
    
    -- Metadata
    executed_at TIMESTAMP DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

-- Create indexes for tool_executions
CREATE INDEX IF NOT EXISTS idx_tool_executions_tenant_id ON tool_executions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tool_executions_user_id ON tool_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_tool_executions_tool_id ON tool_executions(tool_id);
CREATE INDEX IF NOT EXISTS idx_tool_executions_executed_at ON tool_executions(executed_at);
CREATE INDEX IF NOT EXISTS idx_tool_executions_status ON tool_executions(status);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_tools_updated_at BEFORE UPDATE ON tools
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tenant_tools_updated_at BEFORE UPDATE ON tenant_tools
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE tools IS 'Registry of available external API tools';
COMMENT ON TABLE tenant_tools IS 'Tenant-specific tool configurations and permissions';
COMMENT ON TABLE tool_executions IS 'Audit log of tool executions for analytics';

COMMENT ON COLUMN tools.auth_type IS 'Authentication type: none, api_key, bearer, basic, oauth2';
COMMENT ON COLUMN tools.auth_config IS 'JSON configuration specific to auth_type';
COMMENT ON COLUMN tools.input_schema IS 'JSON Schema for validating tool input';
COMMENT ON COLUMN tools.output_schema IS 'JSON Schema for validating tool output';
COMMENT ON COLUMN tools.is_public IS 'If true, available to all tenants by default';

COMMENT ON COLUMN tenant_tools.custom_auth_config IS 'Tenant-specific API keys or OAuth tokens';
COMMENT ON COLUMN tenant_tools.allowed_user_ids IS 'If set, only these users can use this tool';

COMMENT ON COLUMN tool_executions.status IS 'Execution status: success, error, timeout, rate_limited, in_progress';
