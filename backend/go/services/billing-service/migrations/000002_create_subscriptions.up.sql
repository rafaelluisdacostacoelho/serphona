-- Create subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    stripe_subscription_id VARCHAR(255) UNIQUE NOT NULL,
    plan_id VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    cancel_at_period_end BOOLEAN DEFAULT false,
    canceled_at TIMESTAMP WITH TIME ZONE,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_id ON subscriptions(customer_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe_subscription_id ON subscriptions(stripe_subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_plan_id ON subscriptions(plan_id);

-- Add comments
COMMENT ON TABLE subscriptions IS 'Customer subscriptions managed via Stripe';
COMMENT ON COLUMN subscriptions.customer_id IS 'Reference to the billing customer';
COMMENT ON COLUMN subscriptions.stripe_subscription_id IS 'Stripe subscription ID';
COMMENT ON COLUMN subscriptions.plan_id IS 'Subscription plan identifier';
COMMENT ON COLUMN subscriptions.status IS 'Current subscription status (active, canceled, etc.)';
COMMENT ON COLUMN subscriptions.cancel_at_period_end IS 'Whether to cancel at the end of the current period';
