package handlers

import (
	"context"
	"net/http"
	"time"

	"cryptosignal-news/backend/internal/api/response"
	"cryptosignal-news/backend/internal/cache"
	"cryptosignal-news/backend/internal/config"
	"cryptosignal-news/backend/internal/database"
	"cryptosignal-news/backend/internal/repository"
)

// StatusHandler handles status API endpoints
type StatusHandler struct {
	db          *database.DB
	cache       *cache.Redis
	articleRepo *repository.ArticleRepository
	cfg         *config.Config
	startTime   time.Time
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(db *database.DB, cache *cache.Redis, articleRepo *repository.ArticleRepository, cfg *config.Config) *StatusHandler {
	return &StatusHandler{
		db:          db,
		cache:       cache,
		articleRepo: articleRepo,
		cfg:         cfg,
		startTime:   time.Now(),
	}
}

// TranslationStatusResponse represents translation status info
type TranslationStatusResponse struct {
	Enabled        bool           `json:"enabled"`
	TargetLanguage string         `json:"target_language"`
	Model          string         `json:"model"`
	Interval       string         `json:"interval"`
	BatchSize      int            `json:"batch_size"`
	Stats          *TranslationStatsResponse `json:"stats"`
}

// TranslationStatsResponse represents translation statistics
type TranslationStatsResponse struct {
	TotalArticles int            `json:"total_articles"`
	Completed     int            `json:"completed"`
	Pending       int            `json:"pending"`
	Failed        int            `json:"failed"`
	NoTranslation int            `json:"no_translation_needed"`
	ByLanguage    map[string]int `json:"by_language"`
}

// AIStatusResponse represents AI service status
type AIStatusResponse struct {
	Enabled        bool   `json:"enabled"`
	SentimentModel string `json:"sentiment_model"`
	SummaryModel   string `json:"summary_model"`
}

// SystemStatusResponse represents the full system status
type SystemStatusResponse struct {
	Status      string                    `json:"status"`
	Uptime      string                    `json:"uptime"`
	Environment string                    `json:"environment"`
	Timestamp   string                    `json:"timestamp"`
	Services    ServiceStatusResponse     `json:"services"`
	Translation TranslationStatusResponse `json:"translation"`
	AI          AIStatusResponse          `json:"ai"`
}

// ServiceStatusResponse represents service health
type ServiceStatusResponse struct {
	Database string `json:"database"`
	Redis    string `json:"redis"`
}

// GetStatus handles GET /api/v1/status
func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Check services health
	services := ServiceStatusResponse{
		Database: "healthy",
		Redis:    "healthy",
	}
	overallStatus := "healthy"

	if err := h.db.Ping(ctx); err != nil {
		services.Database = "unhealthy"
		overallStatus = "degraded"
	}

	if err := h.cache.Health(ctx); err != nil {
		services.Redis = "unhealthy"
		overallStatus = "degraded"
	}

	// Get translation stats
	var translationStats *TranslationStatsResponse
	if repoStats, err := h.articleRepo.GetTranslationStats(ctx); err == nil {
		translationStats = &TranslationStatsResponse{
			TotalArticles: repoStats.TotalArticles,
			Completed:     repoStats.ByStatus["completed"],
			Pending:       repoStats.ByStatus["pending"],
			Failed:        repoStats.ByStatus["failed"],
			NoTranslation: repoStats.ByStatus["none"],
			ByLanguage:    repoStats.ByLanguage,
		}
	}

	// Build response
	resp := SystemStatusResponse{
		Status:      overallStatus,
		Uptime:      time.Since(h.startTime).Round(time.Second).String(),
		Environment: h.cfg.Env,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Services:    services,
		Translation: TranslationStatusResponse{
			Enabled:        h.cfg.TranslationEnabled,
			TargetLanguage: h.cfg.TranslationTargetLanguage,
			Model:          h.cfg.ModelTranslation,
			Interval:       h.cfg.TranslationInterval.String(),
			BatchSize:      h.cfg.TranslationBatchSize,
			Stats:          translationStats,
		},
		AI: AIStatusResponse{
			Enabled:        h.cfg.GroqAPIKey != "",
			SentimentModel: h.cfg.ModelSentiment,
			SummaryModel:   h.cfg.ModelSummary,
		},
	}

	response.Success(w, resp)
}
