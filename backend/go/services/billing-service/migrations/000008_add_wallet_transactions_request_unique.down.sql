-- Drop the additional idempotency index (original index may remain from prior migrations)
DROP INDEX IF EXISTS idx_wallet_transactions_wallet_reference_unique_v2;
