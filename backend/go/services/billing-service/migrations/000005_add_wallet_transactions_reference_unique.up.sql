-- Enforce idempotency for wallet transactions with non-null references
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_reference_unique
    ON wallet_transactions(wallet_id, reference)
    WHERE reference IS NOT NULL;
