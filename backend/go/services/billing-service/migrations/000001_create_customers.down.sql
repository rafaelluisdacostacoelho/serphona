-- Drop customers table
DROP INDEX IF EXISTS idx_customers_email;
DROP INDEX IF EXISTS idx_customers_stripe_customer_id;
DROP INDEX IF EXISTS idx_customers_tenant_id;
DROP TABLE IF EXISTS customers;
