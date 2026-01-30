package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cryptosignal-news/backend/internal/api/response"
	"github.com/cryptosignal-news/backend/internal/auth"
	"github.com/cryptosignal-news/backend/internal/config"
)

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*clientRequests
	limit    int           // requests per window
	window   time.Duration // time window
}

type clientRequests struct {
	count     int
	resetTime time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientRequests),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup removes expired entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, client := range rl.requests {
			if now.After(client.resetTime) {
				delete(rl.requests, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.requests[ip]

	if !exists || now.After(client.resetTime) {
		// New window
		rl.requests[ip] = &clientRequests{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	if client.count >= rl.limit {
		return false
	}

	client.count++
	return true
}

// RemainingRequests returns the number of remaining requests for an IP
func (rl *RateLimiter) RemainingRequests(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	client, exists := rl.requests[ip]
	if !exists {
		return rl.limit
	}

	if time.Now().After(client.resetTime) {
		return rl.limit
	}

	remaining := rl.limit - client.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RateLimit creates a middleware that limits requests by IP address
// Default: 10 requests per minute for free tier
func RateLimit(limiter *RateLimiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			if !limiter.Allow(ip) {
				w.Header().Set("X-RateLimit-Limit", "10")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				response.TooManyRequests(w, "Rate limit exceeded. Please try again later.")
				return
			}

			remaining := limiter.RemainingRequests(ip)
			w.Header().Set("X-RateLimit-Limit", "10")
			w.Header().Set("X-RateLimit-Remaining", string(rune('0'+remaining)))

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// RemoteAddr is in the form "IP:port"
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

// DefaultRateLimiter creates a default rate limiter with 10 requests per minute
func DefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(10, time.Minute)
}

// TierRateLimiter implements tier-based rate limiting
type TierRateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*clientRequests
	cfg      *config.Config
	window   time.Duration
}

// NewTierRateLimiter creates a tier-aware rate limiter
func NewTierRateLimiter(cfg *config.Config) *TierRateLimiter {
	trl := &TierRateLimiter{
		requests: make(map[string]*clientRequests),
		cfg:      cfg,
		window:   time.Minute,
	}

	// Start cleanup goroutine
	go trl.cleanup()

	return trl
}

func (trl *TierRateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		trl.mu.Lock()
		now := time.Now()
		for key, client := range trl.requests {
			if now.After(client.resetTime) {
				delete(trl.requests, key)
			}
		}
		trl.mu.Unlock()
	}
}

// getLimitForTier returns the rate limit for a given tier
func (trl *TierRateLimiter) getLimitForTier(tier string) int {
	switch tier {
	case "enterprise":
		return trl.cfg.RateLimitEnterprise
	case "pro":
		return trl.cfg.RateLimitPro
	case "free":
		return trl.cfg.RateLimitFree
	default:
		return trl.cfg.RateLimitAnonymous
	}
}

// Allow checks if a request should be allowed based on identifier and tier
func (trl *TierRateLimiter) Allow(identifier string, tier string) (bool, int, int) {
	limit := trl.getLimitForTier(tier)

	trl.mu.Lock()
	defer trl.mu.Unlock()

	now := time.Now()
	client, exists := trl.requests[identifier]

	if !exists || now.After(client.resetTime) {
		trl.requests[identifier] = &clientRequests{
			count:     1,
			resetTime: now.Add(trl.window),
		}
		return true, limit, limit - 1
	}

	if client.count >= limit {
		return false, limit, 0
	}

	client.count++
	remaining := limit - client.count
	if remaining < 0 {
		remaining = 0
	}
	return true, limit, remaining
}

// TierRateLimit creates a middleware that limits requests by user tier
func TierRateLimit(cfg *config.Config, limiter *TierRateLimiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if rate limiting is disabled
			if !cfg.RateLimitEnabled {
				next.ServeHTTP(w, r)
				return
			}

			// Determine identifier and tier
			var identifier string
			var tier string

			user := auth.GetUser(r.Context())
			if user != nil {
				// Authenticated user - use user ID and their tier
				identifier = "user:" + user.ID
				tier = user.Tier
			} else {
				// Anonymous - use IP address
				identifier = "ip:" + getClientIP(r)
				tier = "anonymous"
			}

			allowed, limit, remaining := limiter.Allow(identifier, tier)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			if !allowed {
				w.Header().Set("Retry-After", "60")
				response.TooManyRequests(w, "Rate limit exceeded. Please try again later.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
