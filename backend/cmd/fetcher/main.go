package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cryptosignal-news/backend/internal/cache"
	"github.com/cryptosignal-news/backend/internal/config"
	"github.com/cryptosignal-news/backend/internal/database"
	"github.com/cryptosignal-news/backend/internal/fetcher"
	"github.com/cryptosignal-news/backend/internal/sources"
)

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting CryptoSignal News Fetcher Worker...")

	// Load configuration
	cfg := config.Load()
	log.Printf("Environment: %s", cfg.Env)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
	dbCfg := database.DefaultConfig(cfg.DatabaseURL)
	db, err := database.New(ctx, dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to PostgreSQL")

	// Sync sources from Go code to database
	if err := syncSources(ctx, db); err != nil {
		log.Printf("Warning: Failed to sync sources: %v", err)
	}

	// Connect to Redis
	redis, err := cache.NewRedisFromURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()
	log.Println("Connected to Redis")

	// Create fetcher with configuration
	fetcherCfg := &fetcher.Config{
		WorkerCount:   getEnvInt("FETCHER_WORKERS", 50),
		Timeout:       getEnvDuration("FETCHER_TIMEOUT", 10*time.Second),
		MaxArticleAge: getEnvDuration("FETCHER_MAX_AGE", 7*24*time.Hour),
	}
	log.Printf("Fetcher config: workers=%d, timeout=%v, max_age=%v",
		fetcherCfg.WorkerCount, fetcherCfg.Timeout, fetcherCfg.MaxArticleAge)

	f := fetcher.New(db, redis, fetcherCfg)

	// Create scheduler
	schedulerCfg := &fetcher.SchedulerConfig{
		Interval: getEnvDuration("FETCH_INTERVAL", 3*time.Minute),
	}
	log.Printf("Scheduler config: interval=%v", schedulerCfg.Interval)

	scheduler := fetcher.NewScheduler(f, schedulerCfg)

	// Set up graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start scheduler in goroutine
	go func() {
		scheduler.Start(ctx)
	}()

	log.Println("Fetcher worker started successfully")
	log.Printf("Fetching feeds every %v", schedulerCfg.Interval)

	// Wait for shutdown signal
	sig := <-shutdown
	log.Printf("Received signal: %v", sig)

	// Initiate graceful shutdown
	log.Println("Initiating graceful shutdown...")

	// Stop the scheduler
	scheduler.Stop()

	// Cancel context to stop any in-flight operations
	cancel()

	// Give some time for cleanup
	time.Sleep(2 * time.Second)

	log.Println("Fetcher worker stopped")
}

// getEnvInt gets an integer environment variable with a default value
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var result int
		if _, err := fmt.Sscanf(val, "%d", &result); err == nil {
			return result
		}
	}
	return defaultVal
}

// getEnvDuration gets a duration environment variable with a default value
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

// syncSources inserts all sources from Go code into database (if not exists)
func syncSources(ctx context.Context, db *database.DB) error {
	allSources := sources.GetAllFeedSources()
	log.Printf("Syncing %d sources from Go code to database...", len(allSources))

	inserted := 0
	for _, src := range allSources {
		_, err := db.Exec(ctx, `
			INSERT INTO sources (key, name, rss_url, website_url, category, language, is_enabled, reliability_score)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (key) DO NOTHING
		`, src.Key, src.Name, src.RSSURL, src.WebsiteURL, src.Category, src.Language, src.IsEnabled, 0.80)
		if err != nil {
			log.Printf("Warning: Failed to insert source %s: %v", src.Key, err)
			continue
		}
		inserted++
	}

	log.Printf("Sources sync complete: %d sources available", inserted)
	return nil
}
