-- Drop subscriptions table
DROP INDEX IF EXISTS idx_subscriptions_plan_id;
DROP INDEX IF EXISTS idx_subscriptions_status;
DROP INDEX IF EXISTS idx_subscriptions_stripe_subscription_id;
DROP INDEX IF EXISTS idx_subscriptions_customer_id;
DROP TABLE IF EXISTS subscriptions;
