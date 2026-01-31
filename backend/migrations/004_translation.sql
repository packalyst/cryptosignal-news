-- CryptoSignal News - Translation Support
-- Migration: 004_translation.sql
-- Description: Adds translation status and original content fields for multi-language support

-- Add translation fields to articles
ALTER TABLE articles ADD COLUMN IF NOT EXISTS original_title TEXT;
ALTER TABLE articles ADD COLUMN IF NOT EXISTS original_description TEXT;
ALTER TABLE articles ADD COLUMN IF NOT EXISTS original_language VARCHAR(10);
ALTER TABLE articles ADD COLUMN IF NOT EXISTS translation_status VARCHAR(20) DEFAULT 'none';
-- translation_status: 'none' (English, no translation needed), 'pending', 'completed', 'failed'

-- Index for finding articles that need translation
CREATE INDEX IF NOT EXISTS idx_articles_translation_status ON articles(translation_status);
CREATE INDEX IF NOT EXISTS idx_articles_translation_pending ON articles(translation_status, created_at)
    WHERE translation_status = 'pending';

-- Composite index for translation worker queries
CREATE INDEX IF NOT EXISTS idx_articles_needs_translation ON articles(translation_status, id)
    WHERE translation_status = 'pending';
