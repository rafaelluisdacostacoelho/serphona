\if :{?seed_test_wallets}
\else
\set seed_test_wallets false
\endif

\if :seed_test_wallets
DELETE FROM wallet_transactions
WHERE wallet_id IN ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222');

DELETE FROM wallets
WHERE tenant_id IN ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb');
\endif
