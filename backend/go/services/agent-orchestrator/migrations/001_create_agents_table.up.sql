-- Create agents table
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    system_prompt TEXT NOT NULL,
    model VARCHAR(50) NOT NULL,
    temperature DECIMAL(3, 2) NOT NULL DEFAULT 0.7,
    max_tokens INTEGER NOT NULL DEFAULT 2000,
    tools JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    CONSTRAINT agents_tenant_name_unique UNIQUE (tenant_id, name),
    CONSTRAINT agents_temperature_check CHECK (temperature >= 0 AND temperature <= 2),
    CONSTRAINT agents_max_tokens_check CHECK (max_tokens > 0)
);

-- Create indexes
CREATE INDEX idx_agents_tenant_id ON agents(tenant_id);
CREATE INDEX idx_agents_is_active ON agents(is_active);
CREATE INDEX idx_agents_tenant_id_is_active ON agents(tenant_id, is_active);
CREATE INDEX idx_agents_name ON agents(name);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for updated_at
CREATE TRIGGER update_agents_updated_at
    BEFORE UPDATE ON agents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert default agent for testing
INSERT INTO agents (
    id, tenant_id, name, display_name, description,
    system_prompt, model, temperature, max_tokens, tools, is_active
) VALUES (
    '550e8400-e29b-41d4-a716-446655440000',
    '550e8400-e29b-41d4-a716-446655440001',
    'assistant',
    'AI Assistant',
    'Default AI assistant for general conversations',
    'You are a helpful AI assistant. Be concise, accurate, and friendly in your responses.',
    'gpt-4-turbo',
    0.7,
    2000,
    '[]'::jsonb,
    true
) ON CONFLICT (tenant_id, name) DO NOTHING;
