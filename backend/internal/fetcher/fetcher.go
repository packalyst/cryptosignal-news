package fetcher

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cryptosignal-news/backend/internal/cache"
	"cryptosignal-news/backend/internal/database"
	"cryptosignal-news/backend/internal/models"
	"cryptosignal-news/backend/internal/parser"
	"cryptosignal-news/backend/internal/repository"
	"cryptosignal-news/backend/internal/sources"
)

// Fetcher orchestrates the fetching of RSS feeds
type Fetcher struct {
	db             *database.DB
	cache          *cache.Redis
	parser         *parser.FeedParser
	cleaner        *parser.Cleaner
	enricher       *Enricher
	articleRepo    *repository.ArticleRepository
	sourceRepo     *repository.SourceRepository
	workerPool     *WorkerPool
	timeout        time.Duration
	maxArticleAge  time.Duration
	targetLanguage string // Target language for translations (empty = no translation)
}

// Config holds fetcher configuration
type Config struct {
	WorkerCount    int
	Timeout        time.Duration
	MaxArticleAge  time.Duration
	TargetLanguage string // Target language for translations (e.g., "en", "ro"). Empty = no translation.
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		WorkerCount:    50,
		Timeout:        10 * time.Second,
		MaxArticleAge:  7 * 24 * time.Hour, // 7 days
		TargetLanguage: "",                 // No translation by default
	}
}

// FetchResult contains the results of a fetch operation
type FetchResult struct {
	TotalSources    int
	SuccessfulFeeds int
	FailedFeeds     int
	TotalArticles   int
	NewArticles     int
	Duration        time.Duration
	Errors          []FetchError
}

// FetchError represents an error from a specific source
type FetchError struct {
	SourceID  int
	SourceKey string
	Error     error
}

// New creates a new Fetcher
func New(db *database.DB, cache *cache.Redis, cfg *Config) *Fetcher {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Fetcher{
		db:             db,
		cache:          cache,
		parser:         parser.NewFeedParser(),
		cleaner:        parser.NewCleaner(),
		enricher:       NewEnricher(),
		articleRepo:    repository.NewArticleRepository(db),
		sourceRepo:     repository.NewSourceRepository(db),
		workerPool:     NewWorkerPool(cfg.WorkerCount),
		timeout:        cfg.Timeout,
		maxArticleAge:  cfg.MaxArticleAge,
		targetLanguage: strings.ToLower(cfg.TargetLanguage),
	}
}

// FetchAll fetches all enabled sources concurrently
func (f *Fetcher) FetchAll(ctx context.Context) (*FetchResult, error) {
	start := time.Now()

	// Get all enabled sources from database
	dbSources, err := f.sourceRepo.GetEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled sources: %w", err)
	}

	if len(dbSources) == 0 {
		log.Println("[fetcher] No enabled sources found")
		return &FetchResult{}, nil
	}

	log.Printf("[fetcher] Starting fetch for %d sources", len(dbSources))

	// Convert to Source interface and filter unhealthy sources
	var healthySources []sources.Source
	for i := range dbSources {
		src := sources.NewDBSource(&dbSources[i])
		if dbSources[i].IsHealthy() {
			healthySources = append(healthySources, src)
		} else {
			log.Printf("[fetcher] Skipping unhealthy source: %s (errors=%d)",
				dbSources[i].Key, dbSources[i].ErrorCount)
		}
	}

	// Create fetch jobs
	jobs := make([]FetchJob, len(healthySources))
	for i, src := range healthySources {
		jobs[i] = FetchJob{
			Source:  src,
			Fetcher: f,
		}
	}

	// Process jobs concurrently
	results := f.workerPool.ProcessJobs(ctx, jobs, f.timeout)

	// Collect articles and errors
	batchProcessor := NewBatchProcessor(100)
	allArticles, errorResults := batchProcessor.CollectArticles(results)

	// Deduplicate articles before insert
	uniqueArticles := f.deduplicateArticles(allArticles)

	// Insert new articles
	inserted, err := f.articleRepo.BulkInsert(ctx, uniqueArticles)
	if err != nil {
		log.Printf("[fetcher] Error inserting articles: %v", err)
	}

	// Update source statistics
	f.updateSourceStats(ctx, results)

	// Build result
	result := &FetchResult{
		TotalSources:    len(dbSources),
		SuccessfulFeeds: len(results) - len(errorResults),
		FailedFeeds:     len(errorResults),
		TotalArticles:   len(allArticles),
		NewArticles:     inserted,
		Duration:        time.Since(start),
		Errors:          make([]FetchError, 0, len(errorResults)),
	}

	// Collect errors
	for _, er := range errorResults {
		result.Errors = append(result.Errors, FetchError{
			SourceID:  er.SourceID,
			SourceKey: er.SourceKey,
			Error:     er.Error,
		})
	}

	// Log results
	log.Printf("[fetcher] Completed in %v: %d sources, %d articles fetched, %d new",
		result.Duration.Round(time.Millisecond),
		result.TotalSources,
		result.TotalArticles,
		result.NewArticles)

	if len(result.Errors) > 0 {
		log.Printf("[fetcher] %d sources failed:", len(result.Errors))
		for _, e := range result.Errors[:min(5, len(result.Errors))] {
			log.Printf("[fetcher]   - %s: %v", e.SourceKey, e.Error)
		}
		if len(result.Errors) > 5 {
			log.Printf("[fetcher]   ... and %d more", len(result.Errors)-5)
		}
	}

	return result, nil
}

// FetchSource fetches articles from a single source
func (f *Fetcher) FetchSource(ctx context.Context, src sources.Source) ([]models.Article, error) {
	// Parse the feed
	feed, err := f.parser.ParseURL(ctx, src.GetURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Convert feed items to articles
	articles := make([]models.Article, 0, len(feed.Items))
	minDate := time.Now().UTC().Add(-f.maxArticleAge)

	// Check if this source needs translation
	// Translation is needed if: target language is set AND source language differs from target
	sourceLang := strings.ToLower(src.GetLanguage())
	needsTranslation := f.targetLanguage != "" && sourceLang != "" && sourceLang != f.targetLanguage

	for _, item := range feed.Items {
		// Skip old articles
		if item.PubDate.Before(minDate) {
			continue
		}

		title := f.cleaner.SanitizeForDB(item.Title, 1000)
		desc := item.GetCleanDescription(f.cleaner, 5000)

		article := models.NewArticle(
			src.GetID(),
			item.GUID,
			title,
			item.Link,
			item.PubDate,
		)

		// Set description
		article.SetDescription(desc)

		// Set categories
		article.SetCategories(item.Categories)

		// Mark for translation if non-English source
		if needsTranslation {
			article.SetForTranslation(sourceLang)
		}

		// Enrich article
		f.enricher.EnrichArticle(article, src.GetCategory())

		articles = append(articles, *article)
	}

	return articles, nil
}

// deduplicateArticles removes duplicate articles based on GUID
func (f *Fetcher) deduplicateArticles(articles []models.Article) []models.Article {
	seen := make(map[string]bool, len(articles))
	unique := make([]models.Article, 0, len(articles))

	for _, a := range articles {
		if !seen[a.GUID] {
			seen[a.GUID] = true
			unique = append(unique, a)
		}
	}

	return unique
}

// updateSourceStats updates the database with fetch results
func (f *Fetcher) updateSourceStats(ctx context.Context, results []FetchJobResult) {
	for _, r := range results {
		if r.Error != nil {
			// Increment error count for failed fetches
			if err := f.sourceRepo.IncrementErrorCount(ctx, r.SourceID); err != nil {
				log.Printf("[fetcher] Failed to increment error count for %s: %v", r.SourceKey, err)
			}
		} else {
			// Reset error count and update last fetch time for successful fetches
			if err := f.sourceRepo.ResetErrorCount(ctx, r.SourceID); err != nil {
				log.Printf("[fetcher] Failed to reset error count for %s: %v", r.SourceKey, err)
			}
			if err := f.sourceRepo.UpdateLastFetch(ctx, r.SourceID, time.Now().UTC()); err != nil {
				log.Printf("[fetcher] Failed to update last fetch for %s: %v", r.SourceKey, err)
			}
		}
	}
}

// CacheSeenGUIDs caches article GUIDs to avoid re-processing
func (f *Fetcher) CacheSeenGUIDs(ctx context.Context, guids []string) error {
	if len(guids) == 0 {
		return nil
	}

	// Use Redis set for seen GUIDs
	key := "fetcher:seen_guids"
	members := make([]interface{}, len(guids))
	for i, g := range guids {
		members[i] = g
	}

	if err := f.cache.SAdd(ctx, key, members...); err != nil {
		return fmt.Errorf("failed to cache GUIDs: %w", err)
	}

	// Set expiration to 24 hours
	return f.cache.Expire(ctx, key, 24*time.Hour)
}

// IsGUIDSeen checks if a GUID has been seen recently
func (f *Fetcher) IsGUIDSeen(ctx context.Context, guid string) (bool, error) {
	return f.cache.SIsMember(ctx, "fetcher:seen_guids", guid)
}

// GetArticleRepo returns the article repository
func (f *Fetcher) GetArticleRepo() *repository.ArticleRepository {
	return f.articleRepo
}

// GetSourceRepo returns the source repository
func (f *Fetcher) GetSourceRepo() *repository.SourceRepository {
	return f.sourceRepo
}
