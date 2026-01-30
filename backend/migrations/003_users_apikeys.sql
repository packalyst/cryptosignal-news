-- Migration: Create users and api_keys tables for authentication
-- Version: 003
-- Description: Phase 8 - Authentication and Rate Limiting

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    tier VARCHAR(20) DEFAULT 'free' CHECK (tier IN ('free', 'pro', 'enterprise')),
    api_calls_today INTEGER DEFAULT 0,
    api_calls_month INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create api_keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    name VARCHAR(100) DEFAULT 'API Key',
    is_active BOOLEAN DEFAULT true,
    last_used TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_tier ON users(tier);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- Indexes for api_keys table
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys(is_active);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to auto-update updated_at on users table
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE users IS 'User accounts for authentication and rate limiting';
COMMENT ON TABLE api_keys IS 'API keys for programmatic access';
COMMENT ON COLUMN users.tier IS 'Subscription tier: free, pro, or enterprise';
COMMENT ON COLUMN users.api_calls_today IS 'Number of API calls made today (reset at midnight UTC)';
COMMENT ON COLUMN users.api_calls_month IS 'Number of API calls made this month (reset on 1st)';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA-256 hash of the API key (key itself is never stored)';
COMMENT ON COLUMN api_keys.key_prefix IS 'First part of key for identification (e.g., csn_live_abc12345)';
