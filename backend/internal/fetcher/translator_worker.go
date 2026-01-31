package fetcher

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"cryptosignal-news/backend/internal/ai"
	"cryptosignal-news/backend/internal/models"
	"cryptosignal-news/backend/internal/repository"
)

// TranslatorWorkerConfig holds configuration for the translation worker
type TranslatorWorkerConfig struct {
	Interval  time.Duration // How often to check for pending translations
	BatchSize int           // How many articles to translate per batch
}

// DefaultTranslatorWorkerConfig returns sensible defaults
func DefaultTranslatorWorkerConfig() *TranslatorWorkerConfig {
	return &TranslatorWorkerConfig{
		Interval:  30 * time.Second, // Check every 30 seconds
		BatchSize: 5,                // Translate 5 articles per batch
	}
}

// TranslatorWorker handles background translation of articles
type TranslatorWorker struct {
	translator     *ai.TranslatorService
	articleRepo    *repository.ArticleRepository
	config         *TranslatorWorkerConfig
	stopCh         chan struct{}
	wg             sync.WaitGroup
	retryAfter     time.Time // When we can retry after rate limit
}

// NewTranslatorWorker creates a new translation worker
func NewTranslatorWorker(
	translator *ai.TranslatorService,
	articleRepo *repository.ArticleRepository,
	config *TranslatorWorkerConfig,
) *TranslatorWorker {
	if config == nil {
		config = DefaultTranslatorWorkerConfig()
	}

	return &TranslatorWorker{
		translator:  translator,
		articleRepo: articleRepo,
		config:      config,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the translation worker
func (w *TranslatorWorker) Start(ctx context.Context) {
	log.Printf("[translator] Starting worker: interval=%v, batch_size=%d",
		w.config.Interval, w.config.BatchSize)

	w.wg.Add(1)
	go w.run(ctx)
}

// Stop gracefully stops the translation worker
func (w *TranslatorWorker) Stop() {
	log.Println("[translator] Stopping worker...")
	close(w.stopCh)
	w.wg.Wait()
	log.Println("[translator] Worker stopped")
}

// run is the main worker loop
func (w *TranslatorWorker) run(ctx context.Context) {
	defer w.wg.Done()

	// Run immediately on start
	w.processBatch(ctx)

	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

// processBatch fetches and translates a batch of pending articles
func (w *TranslatorWorker) processBatch(ctx context.Context) {
	// Check if we're in rate limit backoff
	if !w.retryAfter.IsZero() && time.Now().Before(w.retryAfter) {
		remaining := time.Until(w.retryAfter).Round(time.Second)
		if remaining > 0 && remaining%(30*time.Second) == 0 { // Log every 30s
			log.Printf("[translator] Rate limited, waiting %v before retry", remaining)
		}
		return // Still in backoff, skip this cycle
	}

	// Reset retry timer if we're past it
	if !w.retryAfter.IsZero() {
		log.Printf("[translator] Rate limit backoff ended, resuming translations")
		w.retryAfter = time.Time{}
	}

	// Get pending articles
	articles, err := w.articleRepo.GetPendingTranslations(ctx, w.config.BatchSize)
	if err != nil {
		log.Printf("[translator] Error fetching pending translations: %v", err)
		return
	}

	if len(articles) == 0 {
		return
	}

	log.Printf("[translator] Processing %d articles for translation", len(articles))

	// Translate each article
	translated := 0
	failed := 0

	for _, article := range articles {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
		}

		if err := w.translateArticle(ctx, &article); err != nil {
			log.Printf("[translator] Failed to translate article %d: %v", article.ID, err)
			failed++

			// Check if it's a rate limit error and extract retry time
			if retryAfter := extractRetryAfter(err); retryAfter > 0 {
				w.retryAfter = time.Now().Add(retryAfter)
				log.Printf("[translator] Rate limit hit, waiting %v before retry", retryAfter)
				// Mark as failed (will be retried later)
				w.articleRepo.UpdateTranslation(ctx, article.ID, article.Title, article.Description, models.TranslationFailed)
				break // Stop processing this batch
			}

			// Mark as failed
			w.articleRepo.UpdateTranslation(ctx, article.ID, article.Title, article.Description, models.TranslationFailed)
		} else {
			translated++
		}

		// Small delay between translations to avoid rate limiting
		time.Sleep(500 * time.Millisecond)
	}

	if translated > 0 || failed > 0 {
		log.Printf("[translator] Batch complete: %d translated, %d failed", translated, failed)
	}
}

// extractRetryAfter extracts retry duration from an API error
func extractRetryAfter(err error) time.Duration {
	if err == nil {
		return 0
	}

	// Check if it's an APIError with RetryAfter
	if apiErr, ok := err.(*ai.APIError); ok {
		if apiErr.RetryAfter > 0 {
			return apiErr.RetryAfter
		}
		// If no retry-after header but it's a 429, default to 60 seconds
		if apiErr.StatusCode == 429 {
			return 60 * time.Second
		}
	}

	// Check error message for rate limit indicators
	errStr := err.Error()
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "Rate limit") {
		return 60 * time.Second // Default backoff
	}

	return 0
}

// translateArticle translates a single article
func (w *TranslatorWorker) translateArticle(ctx context.Context, article *models.Article) error {
	result, err := w.translator.TranslateArticle(
		ctx,
		article.OriginalTitle,
		article.OriginalDescription,
		article.OriginalLanguage,
	)
	if err != nil {
		return err
	}

	// Update the article with translation
	return w.articleRepo.UpdateTranslation(
		ctx,
		article.ID,
		result.Title,
		result.Description,
		models.TranslationCompleted,
	)
}
