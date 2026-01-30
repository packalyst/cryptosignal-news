package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// checkMinuteLimit checks if the request is within the per-minute limit using sliding window
func (r *RateLimiter) checkMinuteLimit(ctx context.Context, key string, limit int) (bool, int, error) {
	return r.checkSlidingWindowLimit(ctx, key, limit, time.Minute)
}

// checkDayLimit checks if the request is within the per-day limit using sliding window
func (r *RateLimiter) checkDayLimit(ctx context.Context, key string, limit int) (bool, int, error) {
	return r.checkSlidingWindowLimit(ctx, key, limit, 24*time.Hour)
}

// getMinuteRemaining returns the remaining requests for the current minute
func (r *RateLimiter) getMinuteRemaining(ctx context.Context, key string, limit int) (bool, int, error) {
	return r.getSlidingWindowRemaining(ctx, key, limit, time.Minute)
}

// getDayRemaining returns the remaining requests for the current day
func (r *RateLimiter) getDayRemaining(ctx context.Context, key string, limit int) (bool, int, error) {
	return r.getSlidingWindowRemaining(ctx, key, limit, 24*time.Hour)
}

// checkSlidingWindowLimit implements the sliding window rate limiting algorithm
// using Redis sorted sets. Each request is stored with its timestamp as the score.
func (r *RateLimiter) checkSlidingWindowLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	now := time.Now()
	nowUnixMicro := now.UnixMicro()
	windowStart := now.Add(-window).UnixMicro()

	client := r.cache.Client()
	pipe := client.Pipeline()

	// Remove entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(windowStart, 10))

	// Count current entries in window
	countCmd := pipe.ZCard(ctx, key)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, 0, fmt.Errorf("failed to execute rate limit check: %w", err)
	}

	count := countCmd.Val()
	remaining := limit - int(count)

	// Check if we're at the limit
	if int(count) >= limit {
		return false, 0, nil
	}

	// Add new entry with current timestamp as score
	// Using microseconds to ensure uniqueness even for rapid requests
	err = client.ZAdd(ctx, key, redis.Z{
		Score:  float64(nowUnixMicro),
		Member: strconv.FormatInt(nowUnixMicro, 10),
	}).Err()
	if err != nil {
		return false, remaining, fmt.Errorf("failed to add rate limit entry: %w", err)
	}

	// Set expiration on the key to clean up old keys
	err = client.Expire(ctx, key, window+time.Second).Err()
	if err != nil {
		// Non-fatal, just log
		// The key will eventually be cleaned up by Redis if memory pressure occurs
	}

	return true, remaining - 1, nil
}

// getSlidingWindowRemaining returns the remaining requests without adding a new entry
func (r *RateLimiter) getSlidingWindowRemaining(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	now := time.Now()
	windowStart := now.Add(-window).UnixMicro()

	client := r.cache.Client()

	// Count entries within the window
	count, err := client.ZCount(ctx, key, strconv.FormatInt(windowStart, 10), "+inf").Result()
	if err != nil && err != redis.Nil {
		return false, limit, fmt.Errorf("failed to get rate limit count: %w", err)
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return int(count) < limit, remaining, nil
}

// ResetLimit resets the rate limit for an identifier
func (r *RateLimiter) ResetLimit(ctx context.Context, identifier string) error {
	client := r.cache.Client()

	minuteKey := fmt.Sprintf("ratelimit:minute:%s", identifier)
	dayKey := fmt.Sprintf("ratelimit:day:%s", identifier)

	err := client.Del(ctx, minuteKey, dayKey).Err()
	if err != nil {
		return fmt.Errorf("failed to reset rate limit: %w", err)
	}

	return nil
}

// GetUsageStats returns detailed usage statistics for an identifier
func (r *RateLimiter) GetUsageStats(ctx context.Context, identifier string, tier string) (*UsageStats, error) {
	limit := r.GetLimitForTier(tier)

	minuteKey := fmt.Sprintf("ratelimit:minute:%s", identifier)
	dayKey := fmt.Sprintf("ratelimit:day:%s", identifier)

	client := r.cache.Client()

	// Get minute count
	now := time.Now()
	minuteWindowStart := now.Add(-time.Minute).UnixMicro()
	minuteCount, err := client.ZCount(ctx, minuteKey, strconv.FormatInt(minuteWindowStart, 10), "+inf").Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get minute count: %w", err)
	}

	// Get day count
	dayWindowStart := now.Add(-24 * time.Hour).UnixMicro()
	dayCount, err := client.ZCount(ctx, dayKey, strconv.FormatInt(dayWindowStart, 10), "+inf").Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get day count: %w", err)
	}

	minuteRemaining := limit.RequestsPerMinute - int(minuteCount)
	if minuteRemaining < 0 {
		minuteRemaining = 0
	}

	dayRemaining := limit.RequestsPerDay - int(dayCount)
	if limit.RequestsPerDay == -1 {
		dayRemaining = -1 // Unlimited
	} else if dayRemaining < 0 {
		dayRemaining = 0
	}

	return &UsageStats{
		Tier:                tier,
		RequestsThisMinute:  int(minuteCount),
		RequestsToday:       int(dayCount),
		RemainingThisMinute: minuteRemaining,
		RemainingToday:      dayRemaining,
		LimitPerMinute:      limit.RequestsPerMinute,
		LimitPerDay:         limit.RequestsPerDay,
		ResetMinute:         now.Truncate(time.Minute).Add(time.Minute).Unix(),
		ResetDay:            now.Truncate(24 * time.Hour).Add(24 * time.Hour).Unix(),
	}, nil
}

// UsageStats contains detailed usage statistics
type UsageStats struct {
	Tier                string `json:"tier"`
	RequestsThisMinute  int    `json:"requests_this_minute"`
	RequestsToday       int    `json:"requests_today"`
	RemainingThisMinute int    `json:"remaining_this_minute"`
	RemainingToday      int    `json:"remaining_today"` // -1 means unlimited
	LimitPerMinute      int    `json:"limit_per_minute"`
	LimitPerDay         int    `json:"limit_per_day"` // -1 means unlimited
	ResetMinute         int64  `json:"reset_minute"`  // Unix timestamp
	ResetDay            int64  `json:"reset_day"`     // Unix timestamp
}
