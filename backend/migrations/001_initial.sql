-- CryptoSignal News - Initial Database Schema
-- Migration: 001_initial.sql
-- Description: Creates all core tables for the news aggregation service

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================================
-- SOURCES TABLE
-- Stores RSS feed sources and their metadata
-- ============================================================================
CREATE TABLE IF NOT EXISTS sources (
    id SERIAL PRIMARY KEY,
    key VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    rss_url TEXT NOT NULL,
    website_url TEXT,
    category VARCHAR(50) DEFAULT 'dedicated',
    language VARCHAR(10) DEFAULT 'en',
    is_enabled BOOLEAN DEFAULT TRUE,
    reliability_score DECIMAL(3, 2) DEFAULT 0.80,
    last_fetch_at TIMESTAMP WITH TIME ZONE,
    error_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for sources
CREATE INDEX IF NOT EXISTS idx_sources_key ON sources(key);
CREATE INDEX IF NOT EXISTS idx_sources_is_enabled ON sources(is_enabled);
CREATE INDEX IF NOT EXISTS idx_sources_category ON sources(category);
CREATE INDEX IF NOT EXISTS idx_sources_language ON sources(language);
CREATE INDEX IF NOT EXISTS idx_sources_last_fetch_at ON sources(last_fetch_at);

-- ============================================================================
-- ARTICLES TABLE
-- Stores fetched news articles with sentiment analysis
-- ============================================================================
CREATE TABLE IF NOT EXISTS articles (
    id BIGSERIAL PRIMARY KEY,
    source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    guid VARCHAR(512) NOT NULL,
    title TEXT NOT NULL,
    link TEXT NOT NULL,
    description TEXT,
    pub_date TIMESTAMP WITH TIME ZONE NOT NULL,
    categories TEXT[] DEFAULT '{}',
    sentiment VARCHAR(20),
    sentiment_score DECIMAL(4, 3),
    mentioned_coins TEXT[] DEFAULT '{}',
    is_breaking BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Prevent duplicate articles from the same source
    UNIQUE(source_id, guid)
);

-- Indexes for articles
CREATE INDEX IF NOT EXISTS idx_articles_source_id ON articles(source_id);
CREATE INDEX IF NOT EXISTS idx_articles_pub_date ON articles(pub_date DESC);
CREATE INDEX IF NOT EXISTS idx_articles_sentiment ON articles(sentiment);
CREATE INDEX IF NOT EXISTS idx_articles_is_breaking ON articles(is_breaking);
CREATE INDEX IF NOT EXISTS idx_articles_created_at ON articles(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_articles_mentioned_coins ON articles USING GIN(mentioned_coins);
CREATE INDEX IF NOT EXISTS idx_articles_categories ON articles USING GIN(categories);

-- Full-text search index on title and description
CREATE INDEX IF NOT EXISTS idx_articles_search ON articles
    USING GIN(to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, '')));

-- Composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_articles_source_pubdate ON articles(source_id, pub_date DESC);

-- ============================================================================
-- USERS TABLE
-- Stores user accounts and API access information
-- ============================================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    tier VARCHAR(20) DEFAULT 'free',
    api_calls_today INTEGER DEFAULT 0,
    api_calls_month INTEGER DEFAULT 0,
    api_calls_reset_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() + INTERVAL '1 day',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for users
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_tier ON users(tier);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- ============================================================================
-- API_KEYS TABLE
-- Stores API keys for user authentication
-- ============================================================================
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(16) NOT NULL,
    name VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for api_keys
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys(is_active);

-- ============================================================================
-- ALERTS TABLE
-- Stores user-configured alerts for news events
-- ============================================================================
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    conditions JSONB NOT NULL DEFAULT '{}',
    channels TEXT[] DEFAULT '{"email"}',
    webhook_url TEXT,
    is_enabled BOOLEAN DEFAULT TRUE,
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for alerts
CREATE INDEX IF NOT EXISTS idx_alerts_user_id ON alerts(user_id);
CREATE INDEX IF NOT EXISTS idx_alerts_type ON alerts(type);
CREATE INDEX IF NOT EXISTS idx_alerts_is_enabled ON alerts(is_enabled);
CREATE INDEX IF NOT EXISTS idx_alerts_conditions ON alerts USING GIN(conditions);

-- ============================================================================
-- ALERT_HISTORY TABLE
-- Stores history of triggered alerts
-- ============================================================================
CREATE TABLE IF NOT EXISTS alert_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alert_id UUID NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    article_id BIGINT REFERENCES articles(id) ON DELETE SET NULL,
    triggered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    notification_sent BOOLEAN DEFAULT FALSE,
    notification_error TEXT
);

-- Indexes for alert_history
CREATE INDEX IF NOT EXISTS idx_alert_history_alert_id ON alert_history(alert_id);
CREATE INDEX IF NOT EXISTS idx_alert_history_triggered_at ON alert_history(triggered_at DESC);

-- ============================================================================
-- FETCH_LOGS TABLE
-- Stores logs of RSS feed fetch operations
-- ============================================================================
CREATE TABLE IF NOT EXISTS fetch_logs (
    id BIGSERIAL PRIMARY KEY,
    source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL,
    articles_fetched INTEGER DEFAULT 0,
    articles_new INTEGER DEFAULT 0,
    error_message TEXT,
    duration_ms INTEGER
);

-- Indexes for fetch_logs
CREATE INDEX IF NOT EXISTS idx_fetch_logs_source_id ON fetch_logs(source_id);
CREATE INDEX IF NOT EXISTS idx_fetch_logs_started_at ON fetch_logs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_fetch_logs_status ON fetch_logs(status);

-- ============================================================================
-- SENTIMENT_CACHE TABLE
-- Caches sentiment analysis results to reduce API calls
-- ============================================================================
CREATE TABLE IF NOT EXISTS sentiment_cache (
    id BIGSERIAL PRIMARY KEY,
    content_hash VARCHAR(64) UNIQUE NOT NULL,
    sentiment VARCHAR(20) NOT NULL,
    sentiment_score DECIMAL(4, 3) NOT NULL,
    coins TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() + INTERVAL '7 days'
);

-- Indexes for sentiment_cache
CREATE INDEX IF NOT EXISTS idx_sentiment_cache_content_hash ON sentiment_cache(content_hash);
CREATE INDEX IF NOT EXISTS idx_sentiment_cache_expires_at ON sentiment_cache(expires_at);

-- ============================================================================
-- FUNCTIONS AND TRIGGERS
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_sources_updated_at
    BEFORE UPDATE ON sources
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SEED DATA - Initial Sources
-- ============================================================================
INSERT INTO sources (key, name, rss_url, website_url, category, language, reliability_score) VALUES
    ('coindesk', 'CoinDesk', 'https://www.coindesk.com/arc/outboundfeeds/rss/', 'https://www.coindesk.com', 'dedicated', 'en', 0.90),
    ('cointelegraph', 'CoinTelegraph', 'https://cointelegraph.com/rss', 'https://cointelegraph.com', 'dedicated', 'en', 0.88),
    ('decrypt', 'Decrypt', 'https://decrypt.co/feed', 'https://decrypt.co', 'dedicated', 'en', 0.85),
    ('theblock', 'The Block', 'https://www.theblock.co/rss.xml', 'https://www.theblock.co', 'dedicated', 'en', 0.87),
    ('bitcoinmagazine', 'Bitcoin Magazine', 'https://bitcoinmagazine.com/feed', 'https://bitcoinmagazine.com', 'dedicated', 'en', 0.86),
    ('cryptoslate', 'CryptoSlate', 'https://cryptoslate.com/feed/', 'https://cryptoslate.com', 'dedicated', 'en', 0.82),
    ('newsbtc', 'NewsBTC', 'https://www.newsbtc.com/feed/', 'https://www.newsbtc.com', 'dedicated', 'en', 0.78),
    ('bitcoinist', 'Bitcoinist', 'https://bitcoinist.com/feed/', 'https://bitcoinist.com', 'dedicated', 'en', 0.77),
    ('cryptonews', 'Crypto.news', 'https://crypto.news/feed/', 'https://crypto.news', 'dedicated', 'en', 0.80),
    ('blockonomi', 'Blockonomi', 'https://blockonomi.com/feed/', 'https://blockonomi.com', 'dedicated', 'en', 0.79)
ON CONFLICT (key) DO NOTHING;

-- ============================================================================
-- GRANTS (adjust as needed for your setup)
-- ============================================================================
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO your_app_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO your_app_user;
