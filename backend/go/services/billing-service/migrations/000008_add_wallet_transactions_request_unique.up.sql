-- Reinforce idempotency on wallet transactions by request_id/reference per wallet
-- New index name avoids clashing with legacy index from earlier migrations
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_reference_unique_v2
    ON wallet_transactions(wallet_id, reference)
    WHERE reference IS NOT NULL;
