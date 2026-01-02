-- Drop idempotency index on wallet transactions references
DROP INDEX IF EXISTS idx_wallet_transactions_wallet_reference_unique;
