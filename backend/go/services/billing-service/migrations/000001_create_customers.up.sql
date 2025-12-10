-- Create customers table
CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    stripe_customer_id VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    phone VARCHAR(50),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_customers_tenant_id ON customers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_customers_stripe_customer_id ON customers(stripe_customer_id);
CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email);

-- Add comments
COMMENT ON TABLE customers IS 'Billing customers linked to platform tenants';
COMMENT ON COLUMN customers.tenant_id IS 'Reference to the platform tenant';
COMMENT ON COLUMN customers.stripe_customer_id IS 'Stripe customer ID for payment processing';
COMMENT ON COLUMN customers.metadata IS 'Additional customer metadata in JSON format';
