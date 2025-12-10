-- Create wallets table
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    tenant_id UUID UNIQUE NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
    is_locked BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create wallet_transactions table
CREATE TABLE IF NOT EXISTS wallet_transactions (
    id UUID PRIMARY KEY,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    type VARCHAR(20) NOT NULL,
    description TEXT,
    reference VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_wallets_tenant_id ON wallets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_id ON wallet_transactions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_created_at ON wallet_transactions(created_at);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_type ON wallet_transactions(type);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_reference ON wallet_transactions(reference);

-- Add comments
COMMENT ON TABLE wallets IS 'Credit wallets for tenants';
COMMENT ON COLUMN wallets.tenant_id IS 'Reference to the platform tenant';
COMMENT ON COLUMN wallets.balance IS 'Current balance in cents';
COMMENT ON COLUMN wallets.currency IS 'Currency code (ISO 4217)';
COMMENT ON COLUMN wallets.is_locked IS 'Whether the wallet is locked for transactions';

COMMENT ON TABLE wallet_transactions IS 'Transaction history for wallets';
COMMENT ON COLUMN wallet_transactions.wallet_id IS 'Reference to the wallet';
COMMENT ON COLUMN wallet_transactions.amount IS 'Transaction amount in cents';
COMMENT ON COLUMN wallet_transactions.type IS 'Transaction type (credit or debit)';
COMMENT ON COLUMN wallet_transactions.reference IS 'External reference (invoice ID, usage record ID, etc.)';
