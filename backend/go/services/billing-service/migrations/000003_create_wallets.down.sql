-- Drop wallet_transactions table
DROP INDEX IF EXISTS idx_wallet_transactions_reference;
DROP INDEX IF EXISTS idx_wallet_transactions_type;
DROP INDEX IF EXISTS idx_wallet_transactions_created_at;
DROP INDEX IF EXISTS idx_wallet_transactions_wallet_id;
DROP TABLE IF EXISTS wallet_transactions;

-- Drop wallets table
DROP INDEX IF EXISTS idx_wallets_tenant_id;
DROP TABLE IF EXISTS wallets;
