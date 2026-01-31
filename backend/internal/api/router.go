package api

import (
	"time"

	"github.com/go-chi/chi/v5"

	"cryptosignal-news/backend/internal/ai"
	"cryptosignal-news/backend/internal/api/handlers"
	"cryptosignal-news/backend/internal/auth"
	"cryptosignal-news/backend/internal/cache"
	"cryptosignal-news/backend/internal/config"
	"cryptosignal-news/backend/internal/database"
	"cryptosignal-news/backend/internal/middleware"
	"cryptosignal-news/backend/internal/repository"
	"cryptosignal-news/backend/internal/service"
)

// NewRouter creates and configures the main router
func NewRouter(cfg *config.Config, db *database.DB, redisCache *cache.Redis) *chi.Mux {
	r := chi.NewRouter()

	// Initialize repositories
	articleRepo := repository.NewArticleRepository(db)
	sourceRepo := repository.NewSourceRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Initialize auth services (needed for rate limiter)
	jwtService := auth.NewJWTService(cfg.JWTSecret, 24*time.Hour, cfg.JWTRefreshGracePeriod)
	apiKeyService := auth.NewAPIKeyService(db, cfg.MaxAPIKeysPerUser)
	authMiddleware := auth.NewAuthMiddleware(jwtService, apiKeyService)

	// Create tier-based rate limiter
	tierRateLimiter := middleware.NewTierRateLimiter(cfg)

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Timing)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SecurityHeadersWithConfig(cfg))
	r.Use(middleware.CORSWithOrigins(cfg.CORSOrigins))
	r.Use(authMiddleware.OptionalAuth)                        // Check auth for rate limiting (doesn't require auth)
	r.Use(middleware.TierRateLimit(cfg, tierRateLimiter))

	// Initialize services
	// When translation is enabled, exclude articles that haven't been translated yet
	newsService := service.NewNewsService(articleRepo, redisCache, cfg.TranslationEnabled)
	sourceService := service.NewSourceService(sourceRepo, redisCache)

	// Initialize AI services with configurable models
	aiCache := ai.NewAICache(redisCache)
	groqClient := ai.NewGroqClient(cfg.GroqAPIKey)
	sentimentService := ai.NewSentimentService(groqClient, aiCache, cfg.ModelSentiment)
	summaryService := ai.NewSummaryService(groqClient, aiCache, cfg.ModelSummary)
	signalsService := ai.NewSignalsService(groqClient, aiCache, cfg.ModelSummary)

	// Initialize handlers
	healthHandler := handlers.NewHealthChecker(db, redisCache)
	newsHandler := handlers.NewNewsHandler(newsService)
	sourceHandler := handlers.NewSourceHandler(sourceService)
	aiHandler := handlers.NewAIHandler(sentimentService, summaryService, signalsService, newsService)
	authHandler := handlers.NewAuthHandler(userRepo, jwtService, apiKeyService)
	statusHandler := handlers.NewStatusHandler(db, redisCache, articleRepo, cfg)

	// Health endpoints
	r.Get("/health", healthHandler.Health)
	r.Get("/health/live", handlers.LivenessProbe)
	r.Get("/health/ready", healthHandler.ReadinessProbe)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth endpoints
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.RefreshToken)

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

		// Status endpoint
		r.Get("/status", statusHandler.GetStatus)

		// Protected user endpoints (require authentication)
		r.Route("/user", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Get("/me", authHandler.GetCurrentUser)
			r.Post("/api-keys", authHandler.CreateAPIKey)
			r.Get("/api-keys", authHandler.ListAPIKeys)
			r.Delete("/api-keys/{keyID}", authHandler.RevokeAPIKey)
		})
	})

	return r
}
