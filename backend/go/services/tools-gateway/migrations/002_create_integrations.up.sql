-- Create integrations table
CREATE TABLE IF NOT EXISTS integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    provider VARCHAR(100),
    type VARCHAR(50) NOT NULL,
    base_url TEXT NOT NULL,
    
    -- Authentication
    auth_type VARCHAR(50) NOT NULL,
    auth_config JSONB DEFAULT '{}',
    
    -- OAuth 2.0 config
    oauth2_config JSONB,
    
    -- Protocol configs
    graphql_config JSONB,
    soap_config JSONB,
    
    -- Headers and settings
    default_headers JSONB DEFAULT '{}',
    settings JSONB DEFAULT '{}',
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Indexes
    CONSTRAINT integrations_name_key CHECK (char_length(name) > 0),
    CONSTRAINT integrations_base_url_key CHECK (char_length(base_url) > 0)
);

-- Indexes for integrations
CREATE INDEX idx_integrations_tenant_id ON integrations(tenant_id);
CREATE INDEX idx_integrations_provider ON integrations(provider);
CREATE INDEX idx_integrations_type ON integrations(type);
CREATE INDEX idx_integrations_is_active ON integrations(is_active);
CREATE UNIQUE INDEX idx_integrations_tenant_provider ON integrations(tenant_id, provider) WHERE provider IS NOT NULL;

-- Create oauth_tokens table
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    user_id UUID,
    
    -- Tokens (should be encrypted at application level)
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_type VARCHAR(50) DEFAULT 'Bearer',
    
    -- Expiration
    expires_at TIMESTAMP WITH TIME ZONE,
    refresh_at TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    scopes JSONB DEFAULT '[]',
    extra JSONB DEFAULT '{}',
    
    -- Status
    is_valid BOOLEAN DEFAULT true,
    is_revoked BOOLEAN DEFAULT false,
    revoked_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for oauth_tokens
CREATE INDEX idx_oauth_tokens_integration_id ON oauth_tokens(integration_id);
CREATE INDEX idx_oauth_tokens_tenant_id ON oauth_tokens(tenant_id);
CREATE INDEX idx_oauth_tokens_user_id ON oauth_tokens(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_oauth_tokens_integration_tenant ON oauth_tokens(integration_id, tenant_id);
CREATE INDEX idx_oauth_tokens_integration_tenant_user ON oauth_tokens(integration_id, tenant_id, user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_oauth_tokens_expires_at ON oauth_tokens(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_oauth_tokens_refresh_at ON oauth_tokens(refresh_at) WHERE refresh_at IS NOT NULL;
CREATE INDEX idx_oauth_tokens_is_valid ON oauth_tokens(is_valid);

-- Create oauth_states table
CREATE TABLE IF NOT EXISTS oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    state VARCHAR(255) NOT NULL UNIQUE,
    code_verifier VARCHAR(255),
    redirect_uri TEXT NOT NULL,
    scopes JSONB DEFAULT '[]',
    extra_params JSONB DEFAULT '{}',
    
    -- Status
    is_used BOOLEAN DEFAULT false,
    used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for oauth_states
CREATE UNIQUE INDEX idx_oauth_states_state ON oauth_states(state);
CREATE INDEX idx_oauth_states_integration_id ON oauth_states(integration_id);
CREATE INDEX idx_oauth_states_tenant_id ON oauth_states(tenant_id);
CREATE INDEX idx_oauth_states_user_id ON oauth_states(user_id);
CREATE INDEX idx_oauth_states_expires_at ON oauth_states(expires_at);
CREATE INDEX idx_oauth_states_is_used ON oauth_states(is_used);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add triggers
CREATE TRIGGER update_integrations_updated_at BEFORE UPDATE ON integrations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_oauth_tokens_updated_at BEFORE UPDATE ON oauth_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE integrations IS 'External API integrations (REST, GraphQL, SOAP)';
COMMENT ON TABLE oauth_tokens IS 'OAuth 2.0 access and refresh tokens';
COMMENT ON TABLE oauth_states IS 'OAuth 2.0 state for CSRF protection';

COMMENT ON COLUMN integrations.oauth2_config IS 'OAuth 2.0 configuration (client_id, client_secret, auth_url, token_url, etc)';
COMMENT ON COLUMN integrations.graphql_config IS 'GraphQL-specific configuration (endpoint, batching, etc)';
COMMENT ON COLUMN integrations.soap_config IS 'SOAP/WebService configuration (wsdl_url, namespace, version)';

COMMENT ON COLUMN oauth_tokens.access_token IS 'OAuth access token (should be encrypted)';
COMMENT ON COLUMN oauth_tokens.refresh_token IS 'OAuth refresh token (should be encrypted)';
COMMENT ON COLUMN oauth_tokens.refresh_at IS 'When to proactively refresh (before expires_at)';

COMMENT ON COLUMN oauth_states.code_verifier IS 'PKCE code verifier for enhanced security';
