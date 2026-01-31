package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cryptosignal-news/backend/internal/api"
	"cryptosignal-news/backend/internal/cache"
	"cryptosignal-news/backend/internal/config"
	"cryptosignal-news/backend/internal/database"
)

func main() {
	// Load configuration
	cfg := config.Load()

	log.Printf("[main] Starting CryptoSignal News API (env=%s)", cfg.Env)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
	dbCfg := database.DefaultConfig(cfg.DatabaseURL)
	db, err := database.New(ctx, dbCfg)
	if err != nil {
		log.Fatalf("[main] Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Connect to Redis
	redisCache, err := cache.NewRedisFromURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("[main] Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	// Create router
	router := api.NewRouter(cfg, db, redisCache)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("[main] Server listening on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[main] Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[main] Shutting down server...")

	// Give outstanding requests time to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[main] Server forced to shutdown: %v", err)
	}

	log.Println("[main] Server stopped")
}
