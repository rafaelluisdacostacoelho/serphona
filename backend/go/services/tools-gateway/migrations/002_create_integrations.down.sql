-- Drop triggers
DROP TRIGGER IF EXISTS update_oauth_tokens_updated_at ON oauth_tokens;
DROP TRIGGER IF EXISTS update_integrations_updated_at ON integrations;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (in reverse order due to foreign keys)
DROP TABLE IF EXISTS oauth_states;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS integrations;
