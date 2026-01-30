package api

import (
	"github.com/go-chi/chi/v5"

	"github.com/cryptosignal-news/backend/internal/ai"
	"github.com/cryptosignal-news/backend/internal/api/handlers"
	"github.com/cryptosignal-news/backend/internal/cache"
	"github.com/cryptosignal-news/backend/internal/config"
	"github.com/cryptosignal-news/backend/internal/database"
	"github.com/cryptosignal-news/backend/internal/middleware"
	"github.com/cryptosignal-news/backend/internal/repository"
	"github.com/cryptosignal-news/backend/internal/service"
)

// NewRouter creates and configures the main router
func NewRouter(cfg *config.Config, db *database.DB, redisCache *cache.Redis) *chi.Mux {
	r := chi.NewRouter()

	// Create rate limiter
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitPerMinute, 60*1000000000) // 1 minute in nanoseconds

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Timing)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CORSWithOrigins(cfg.CORSOrigins))
	r.Use(middleware.RateLimit(rateLimiter))

	// Initialize repositories
	articleRepo := repository.NewArticleRepository(db)
	sourceRepo := repository.NewSourceRepository(db)

	// Initialize services
	newsService := service.NewNewsService(articleRepo, redisCache)
	sourceService := service.NewSourceService(sourceRepo, redisCache)

	// Initialize AI services
	aiCache := ai.NewAICache(redisCache)
	groqClient := ai.NewGroqClient(cfg.GroqAPIKey)
	sentimentService := ai.NewSentimentService(groqClient, aiCache)
	summaryService := ai.NewSummaryService(groqClient, aiCache)
	signalsService := ai.NewSignalsService(groqClient, aiCache)

	// Initialize handlers
	healthHandler := handlers.NewHealthChecker(db, redisCache)
	newsHandler := handlers.NewNewsHandler(newsService)
	sourceHandler := handlers.NewSourceHandler(sourceService)
	aiHandler := handlers.NewAIHandler(sentimentService, summaryService, signalsService, newsService)

	// Health endpoints
	r.Get("/health", healthHandler.Health)
	r.Get("/health/live", handlers.LivenessProbe)
	r.Get("/health/ready", healthHandler.ReadinessProbe)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public news endpoints
		r.Get("/news", newsHandler.ListNews)
		r.Get("/news/breaking", newsHandler.BreakingNews)
		r.Get("/news/search", newsHandler.SearchNews)
		r.Get("/news/{id}", newsHandler.GetArticle)
		r.Get("/news/coin/{symbol}", newsHandler.NewsByCoin)

		// Public source endpoints
		r.Get("/sources", sourceHandler.ListSources)
		r.Get("/categories", sourceHandler.ListCategories)

		// AI endpoints
		r.Get("/ai/sentiment", aiHandler.GetSentiment)
		r.Get("/ai/summary", aiHandler.GetSummary)
		r.Get("/ai/signals", aiHandler.GetSignals)
	})

	return r
}
