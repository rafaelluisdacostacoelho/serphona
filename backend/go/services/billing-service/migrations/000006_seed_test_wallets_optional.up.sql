\if :{?seed_test_wallets}
\else
\set seed_test_wallets false
\endif

\if :seed_test_wallets
-- Seed test wallets (safe to re-run)
INSERT INTO wallets (id, tenant_id, balance, currency, is_locked)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 10000, 'USD', false),
    ('22222222-2222-2222-2222-222222222222', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 5000, 'USD', false)
ON CONFLICT (tenant_id) DO NOTHING;

INSERT INTO wallet_transactions (id, wallet_id, amount, type, description, reference, metadata)
VALUES
    ('33333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 10000, 'credit', 'seed credits', 'seed-wallet-a', '{}'::jsonb),
    ('44444444-4444-4444-4444-444444444444', '22222222-2222-2222-2222-222222222222', 5000, 'credit', 'seed credits', 'seed-wallet-b', '{}'::jsonb)
ON CONFLICT (wallet_id, reference) DO NOTHING;
\endif
