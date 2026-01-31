package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"cryptosignal-news/backend/internal/auth"
	"cryptosignal-news/backend/internal/cache"
	"cryptosignal-news/backend/internal/models"
)

// Limit defines rate limits for a tier
type Limit struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	RequestsPerDay    int `json:"requests_per_day"` // -1 means unlimited
}

// DefaultLimits defines the default rate limits per tier
var DefaultLimits = map[string]Limit{
	models.TierFree:       {RequestsPerMinute: 10, RequestsPerDay: 500},
	models.TierPro:        {RequestsPerMinute: 60, RequestsPerDay: 10000},
	models.TierEnterprise: {RequestsPerMinute: 300, RequestsPerDay: -1}, // -1 = unlimited
	models.TierAnonymous:  {RequestsPerMinute: 5, RequestsPerDay: 100},
}

// RateLimitInfo contains rate limit information for a response
type RateLimitInfo struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"` // Unix timestamp
}

// RateLimiter handles rate limiting using Redis
type RateLimiter struct {
	cache  *cache.Redis
	limits map[string]Limit
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cache *cache.Redis) *RateLimiter {
	return &RateLimiter{
		cache:  cache,
		limits: DefaultLimits,
	}
}

// NewRateLimiterWithLimits creates a rate limiter with custom limits
func NewRateLimiterWithLimits(cache *cache.Redis, limits map[string]Limit) *RateLimiter {
	return &RateLimiter{
		cache:  cache,
		limits: limits,
	}
}

// Allow checks if a request should be allowed based on rate limits
func (r *RateLimiter) Allow(ctx context.Context, identifier string, tier string) (bool, error) {
	limit, ok := r.limits[tier]
	if !ok {
		limit = r.limits[models.TierAnonymous]
	}

	// Check per-minute limit
	minuteKey := fmt.Sprintf("ratelimit:minute:%s", identifier)
	allowed, _, err := r.checkMinuteLimit(ctx, minuteKey, limit.RequestsPerMinute)
	if err != nil {
		return false, err
	}
	if !allowed {
		return false, nil
	}

	// Check per-day limit (if not unlimited)
	if limit.RequestsPerDay > 0 {
		dayKey := fmt.Sprintf("ratelimit:day:%s", identifier)
		allowed, _, err = r.checkDayLimit(ctx, dayKey, limit.RequestsPerDay)
		if err != nil {
			return false, err
		}
		if !allowed {
			return false, nil
		}
	}

	return true, nil
}

// GetRemaining returns the remaining requests for an identifier
func (r *RateLimiter) GetRemaining(ctx context.Context, identifier string, tier string) (*RateLimitInfo, error) {
	limit, ok := r.limits[tier]
	if !ok {
		limit = r.limits[models.TierAnonymous]
	}

	minuteKey := fmt.Sprintf("ratelimit:minute:%s", identifier)
	_, minuteRemaining, err := r.getMinuteRemaining(ctx, minuteKey, limit.RequestsPerMinute)
	if err != nil {
		return nil, err
	}

	dayKey := fmt.Sprintf("ratelimit:day:%s", identifier)
	dayRemaining := limit.RequestsPerDay
	if limit.RequestsPerDay > 0 {
		_, dayRemaining, err = r.getDayRemaining(ctx, dayKey, limit.RequestsPerDay)
		if err != nil {
			return nil, err
		}
	}

	// Use the more restrictive remaining count
	remaining := minuteRemaining
	if limit.RequestsPerDay > 0 && dayRemaining < remaining {
		remaining = dayRemaining
	}

	// Calculate reset time (end of current minute)
	now := time.Now()
	reset := now.Truncate(time.Minute).Add(time.Minute).Unix()

	return &RateLimitInfo{
		Limit:     limit.RequestsPerDay,
		Remaining: remaining,
		Reset:     reset,
	}, nil
}

// Middleware returns HTTP middleware that enforces rate limits
func (r *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		// Get identifier and tier
		identifier, tier := r.getIdentifierAndTier(req)

		// Check rate limit
		allowed, err := r.Allow(ctx, identifier, tier)
		if err != nil {
			// Log error but allow request on rate limiter failure
			// This prevents the rate limiter from blocking all requests if Redis is down
			next.ServeHTTP(w, req)
			return
		}

		// Get rate limit info for headers
		info, err := r.GetRemaining(ctx, identifier, tier)
		if err == nil {
			r.setRateLimitHeaders(w, info)
		}

		if !allowed {
			r.writeRateLimitExceeded(w, info)
			return
		}

		next.ServeHTTP(w, req)
	})
}

// getIdentifierAndTier extracts the identifier and tier from the request
func (r *RateLimiter) getIdentifierAndTier(req *http.Request) (string, string) {
	// Check if user is authenticated
	user := auth.GetUser(req.Context())
	if user != nil {
		return user.ID, user.Tier
	}

	// Fall back to IP address for anonymous users
	ip := getClientIP(req)
	return ip, models.TierAnonymous
}

// setRateLimitHeaders sets rate limit headers on the response
func (r *RateLimiter) setRateLimitHeaders(w http.ResponseWriter, info *RateLimitInfo) {
	if info == nil {
		return
	}
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.Reset, 10))
}

// writeRateLimitExceeded writes a rate limit exceeded response
func (r *RateLimiter) writeRateLimitExceeded(w http.ResponseWriter, info *RateLimitInfo) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", strconv.FormatInt(info.Reset-time.Now().Unix(), 10))
	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]interface{}{
		"error":   "rate_limit_exceeded",
		"message": "You have exceeded your rate limit. Please try again later.",
		"retry_after": info.Reset - time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(response)
}

// getClientIP extracts the client IP from the request
func getClientIP(req *http.Request) string {
	// Check X-Forwarded-For header (common for proxies/load balancers)
	xff := req.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the list
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	xri := req.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := req.RemoteAddr
	// Remove port if present
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			return ip[:i]
		}
		if ip[i] == ']' {
			// IPv6 address
			break
		}
	}
	return ip
}

// GetLimits returns the configured limits
func (r *RateLimiter) GetLimits() map[string]Limit {
	return r.limits
}

// GetLimitForTier returns the limit for a specific tier
func (r *RateLimiter) GetLimitForTier(tier string) Limit {
	limit, ok := r.limits[tier]
	if !ok {
		return r.limits[models.TierAnonymous]
	}
	return limit
}
