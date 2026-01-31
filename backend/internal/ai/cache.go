package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cryptosignal-news/backend/internal/cache"
)

const (
	// SentimentCacheTTL is the TTL for sentiment analysis cache
	SentimentCacheTTL = 10 * time.Minute

	// SummaryCacheTTL is the TTL for daily summary cache
	SummaryCacheTTL = 1 * time.Hour

	// SignalsCacheTTL is the TTL for trading signals cache
	SignalsCacheTTL = 30 * time.Minute

	// CoinSentimentCacheTTL is the TTL for coin-specific sentiment cache
	CoinSentimentCacheTTL = 15 * time.Minute

	// CacheKeyPrefix is the prefix for all AI cache keys
	CacheKeyPrefix = "ai:"
)

// AICache wraps the cache.Redis for AI-specific caching
type AICache struct {
	redis *cache.Redis
}

// NewAICache creates a new AI cache wrapper
func NewAICache(redis *cache.Redis) *AICache {
	return &AICache{redis: redis}
}

// sentimentCacheKey generates a cache key for article sentiment
func sentimentCacheKey(articleID int64) string {
	return fmt.Sprintf("%ssentiment:article:%d", CacheKeyPrefix, articleID)
}

// coinSentimentCacheKey generates a cache key for coin sentiment
func coinSentimentCacheKey(symbol string) string {
	return fmt.Sprintf("%ssentiment:coin:%s", CacheKeyPrefix, symbol)
}

// summaryCacheKey generates a cache key for daily summary
func summaryCacheKey() string {
	return fmt.Sprintf("%ssummary:daily", CacheKeyPrefix)
}

// signalsCacheKey generates a cache key for trading signals
func signalsCacheKey() string {
	return fmt.Sprintf("%ssignals:current", CacheKeyPrefix)
}

// GetSentiment retrieves cached sentiment result
func (c *AICache) GetSentiment(ctx context.Context, articleID int64) (*SentimentResult, error) {
	key := sentimentCacheKey(articleID)
	data, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, nil // Cache miss, not an error
	}

	var result SentimentResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached sentiment: %w", err)
	}

	return &result, nil
}

// SetSentiment caches a sentiment result
func (c *AICache) SetSentiment(ctx context.Context, articleID int64, result *SentimentResult) error {
	key := sentimentCacheKey(articleID)
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal sentiment: %w", err)
	}

	if err := c.redis.Set(ctx, key, string(data), SentimentCacheTTL); err != nil {
		return fmt.Errorf("failed to cache sentiment: %w", err)
	}

	return nil
}

// GetCoinSentiment retrieves cached coin sentiment
func (c *AICache) GetCoinSentiment(ctx context.Context, symbol string) (*CoinSentiment, error) {
	key := coinSentimentCacheKey(symbol)
	data, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, nil // Cache miss, not an error
	}

	var result CoinSentiment
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached coin sentiment: %w", err)
	}

	return &result, nil
}

// SetCoinSentiment caches a coin sentiment result
func (c *AICache) SetCoinSentiment(ctx context.Context, symbol string, result *CoinSentiment) error {
	key := coinSentimentCacheKey(symbol)
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal coin sentiment: %w", err)
	}

	if err := c.redis.Set(ctx, key, string(data), CoinSentimentCacheTTL); err != nil {
		return fmt.Errorf("failed to cache coin sentiment: %w", err)
	}

	return nil
}

// GetSummary retrieves cached daily summary
func (c *AICache) GetSummary(ctx context.Context) (*MarketSummary, error) {
	key := summaryCacheKey()
	data, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, nil // Cache miss, not an error
	}

	var result MarketSummary
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached summary: %w", err)
	}

	return &result, nil
}

// SetSummary caches a daily summary
func (c *AICache) SetSummary(ctx context.Context, result *MarketSummary) error {
	key := summaryCacheKey()
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	if err := c.redis.Set(ctx, key, string(data), SummaryCacheTTL); err != nil {
		return fmt.Errorf("failed to cache summary: %w", err)
	}

	return nil
}

// GetSignals retrieves cached trading signals
func (c *AICache) GetSignals(ctx context.Context) (*SignalsResult, error) {
	key := signalsCacheKey()
	data, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, nil // Cache miss, not an error
	}

	var result SignalsResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached signals: %w", err)
	}

	return &result, nil
}

// SetSignals caches trading signals
func (c *AICache) SetSignals(ctx context.Context, result *SignalsResult) error {
	key := signalsCacheKey()
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal signals: %w", err)
	}

	if err := c.redis.Set(ctx, key, string(data), SignalsCacheTTL); err != nil {
		return fmt.Errorf("failed to cache signals: %w", err)
	}

	return nil
}

// InvalidateSentiment removes cached sentiment for an article
func (c *AICache) InvalidateSentiment(ctx context.Context, articleID int64) error {
	key := sentimentCacheKey(articleID)
	return c.redis.Delete(ctx, key)
}

// InvalidateCoinSentiment removes cached sentiment for a coin
func (c *AICache) InvalidateCoinSentiment(ctx context.Context, symbol string) error {
	key := coinSentimentCacheKey(symbol)
	return c.redis.Delete(ctx, key)
}

// InvalidateSummary removes cached daily summary
func (c *AICache) InvalidateSummary(ctx context.Context) error {
	key := summaryCacheKey()
	return c.redis.Delete(ctx, key)
}

// InvalidateSignals removes cached trading signals
func (c *AICache) InvalidateSignals(ctx context.Context) error {
	key := signalsCacheKey()
	return c.redis.Delete(ctx, key)
}
