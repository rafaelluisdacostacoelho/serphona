-- Drop triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables
DROP TABLE IF EXISTS oauth_states;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
