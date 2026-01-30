package handlers

import (
	"net/http"

	"github.com/cryptosignal-news/backend/internal/auth"
	"github.com/cryptosignal-news/backend/internal/models"
	"github.com/cryptosignal-news/backend/internal/ratelimit"
	"github.com/cryptosignal-news/backend/internal/repository"
)

// UsageHandler handles usage tracking endpoints
type UsageHandler struct {
	userRepo    *repository.UserRepository
	rateLimiter *ratelimit.RateLimiter
}

// NewUsageHandler creates a new usage handler
func NewUsageHandler(userRepo *repository.UserRepository, rateLimiter *ratelimit.RateLimiter) *UsageHandler {
	return &UsageHandler{
		userRepo:    userRepo,
		rateLimiter: rateLimiter,
	}
}

// UsageStats represents API usage statistics
type UsageStats struct {
	UserID           string `json:"user_id"`
	Tier             string `json:"tier"`
	APICallsToday    int    `json:"api_calls_today"`
	APICallsMonth    int    `json:"api_calls_month"`
	RemainingToday   int    `json:"remaining_today"`   // -1 means unlimited
	RemainingMonth   int    `json:"remaining_month"`   // -1 means unlimited (not used currently)
	LimitPerMinute   int    `json:"limit_per_minute"`
	LimitPerDay      int    `json:"limit_per_day"`     // -1 means unlimited
	RequestsThisMinute int  `json:"requests_this_minute"`
	RemainingThisMinute int `json:"remaining_this_minute"`
}

// GetUsage returns the API usage statistics for the current user
// GET /api/v1/user/usage
func (h *UsageHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	// Get full user data from database
	fullUser, err := h.userRepo.GetByID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to fetch user data")
		return
	}

	// Get rate limit info from Redis
	rateLimitStats, err := h.rateLimiter.GetUsageStats(r.Context(), user.ID, fullUser.Tier)
	if err != nil {
		// Log error but continue with database stats
		rateLimitStats = &ratelimit.UsageStats{
			Tier:                fullUser.Tier,
			LimitPerMinute:      h.rateLimiter.GetLimitForTier(fullUser.Tier).RequestsPerMinute,
			LimitPerDay:         h.rateLimiter.GetLimitForTier(fullUser.Tier).RequestsPerDay,
		}
	}

	// Calculate remaining
	limit := h.rateLimiter.GetLimitForTier(fullUser.Tier)
	remainingToday := limit.RequestsPerDay - fullUser.APICallsToday
	if limit.RequestsPerDay == -1 {
		remainingToday = -1 // Unlimited
	} else if remainingToday < 0 {
		remainingToday = 0
	}

	stats := UsageStats{
		UserID:              fullUser.ID,
		Tier:                fullUser.Tier,
		APICallsToday:       fullUser.APICallsToday,
		APICallsMonth:       fullUser.APICallsMonth,
		RemainingToday:      remainingToday,
		RemainingMonth:      -1, // Not tracked currently
		LimitPerMinute:      limit.RequestsPerMinute,
		LimitPerDay:         limit.RequestsPerDay,
		RequestsThisMinute:  rateLimitStats.RequestsThisMinute,
		RemainingThisMinute: rateLimitStats.RemainingThisMinute,
	}

	writeJSON(w, http.StatusOK, stats)
}

// GetTierInfo returns information about all available tiers
// GET /api/v1/tiers
func (h *UsageHandler) GetTierInfo(w http.ResponseWriter, r *http.Request) {
	limits := h.rateLimiter.GetLimits()

	tiers := make([]map[string]interface{}, 0)

	for _, tier := range []string{models.TierFree, models.TierPro, models.TierEnterprise} {
		limit := limits[tier]
		tierInfo := map[string]interface{}{
			"name":                tier,
			"requests_per_minute": limit.RequestsPerMinute,
			"requests_per_day":    limit.RequestsPerDay,
		}

		// Add features based on tier
		switch tier {
		case models.TierFree:
			tierInfo["features"] = []string{
				"Access to news feed",
				"Basic search",
				"Public endpoints",
			}
		case models.TierPro:
			tierInfo["features"] = []string{
				"All Free features",
				"AI sentiment analysis",
				"Daily summaries",
				"Priority support",
			}
		case models.TierEnterprise:
			tierInfo["features"] = []string{
				"All Pro features",
				"Unlimited API calls",
				"Custom integrations",
				"Dedicated support",
				"SLA guarantee",
			}
		}

		tiers = append(tiers, tierInfo)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tiers": tiers,
	})
}

// GetAnonymousUsage returns usage info for anonymous users (by IP)
// This is useful for showing rate limit info on public endpoints
func (h *UsageHandler) GetAnonymousUsage(w http.ResponseWriter, r *http.Request) {
	// Get IP address
	ip := getClientIP(r)

	// Get rate limit info
	rateLimitStats, err := h.rateLimiter.GetUsageStats(r.Context(), ip, models.TierAnonymous)
	if err != nil {
		limit := h.rateLimiter.GetLimitForTier(models.TierAnonymous)
		rateLimitStats = &ratelimit.UsageStats{
			Tier:           models.TierAnonymous,
			LimitPerMinute: limit.RequestsPerMinute,
			LimitPerDay:    limit.RequestsPerDay,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tier":                   models.TierAnonymous,
		"requests_this_minute":   rateLimitStats.RequestsThisMinute,
		"requests_today":         rateLimitStats.RequestsToday,
		"remaining_this_minute":  rateLimitStats.RemainingThisMinute,
		"remaining_today":        rateLimitStats.RemainingToday,
		"limit_per_minute":       rateLimitStats.LimitPerMinute,
		"limit_per_day":          rateLimitStats.LimitPerDay,
	})
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (common for proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
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
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
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
