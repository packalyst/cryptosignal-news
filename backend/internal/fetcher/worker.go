package fetcher

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cryptosignal-news/backend/internal/models"
	"github.com/cryptosignal-news/backend/internal/sources"
)

// WorkerPool manages concurrent feed fetching
type WorkerPool struct {
	maxWorkers int
	semaphore  chan struct{}
}

// NewWorkerPool creates a new worker pool with the specified concurrency limit
func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = 50
	}
	return &WorkerPool{
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
	}
}

// FetchJob represents a single fetch job
type FetchJob struct {
	Source  sources.Source
	Fetcher *Fetcher
}

// FetchJobResult represents the result of a fetch job
type FetchJobResult struct {
	SourceID   int
	SourceKey  string
	Articles   []models.Article
	FetchTime  time.Duration
	Error      error
	RetryCount int
}

// ProcessJobs processes all jobs concurrently with the worker pool
func (wp *WorkerPool) ProcessJobs(ctx context.Context, jobs []FetchJob, timeout time.Duration) []FetchJobResult {
	results := make([]FetchJobResult, len(jobs))
	var wg sync.WaitGroup

	// Track progress
	var completed int64
	total := len(jobs)

	for i, job := range jobs {
		wg.Add(1)

		go func(idx int, j FetchJob) {
			defer wg.Done()

			// Acquire semaphore slot
			select {
			case wp.semaphore <- struct{}{}:
				defer func() { <-wp.semaphore }()
			case <-ctx.Done():
				results[idx] = FetchJobResult{
					SourceID:  j.Source.GetID(),
					SourceKey: j.Source.GetKey(),
					Error:     ctx.Err(),
				}
				return
			}

			// Execute the fetch with timeout
			result := wp.executeJob(ctx, j, timeout)
			results[idx] = result

			// Log progress
			done := atomic.AddInt64(&completed, 1)
			if done%25 == 0 || done == int64(total) {
				log.Printf("[worker] Progress: %d/%d sources fetched", done, total)
			}
		}(i, job)
	}

	wg.Wait()
	return results
}

// executeJob executes a single fetch job with timeout and retry
func (wp *WorkerPool) executeJob(ctx context.Context, job FetchJob, timeout time.Duration) FetchJobResult {
	start := time.Now()
	result := FetchJobResult{
		SourceID:  job.Source.GetID(),
		SourceKey: job.Source.GetKey(),
	}

	// Create timeout context
	fetchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Attempt fetch with retries
	maxRetries := 2
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff between retries
			backoff := time.Duration(attempt*500) * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-fetchCtx.Done():
				result.Error = fetchCtx.Err()
				result.FetchTime = time.Since(start)
				return result
			}
		}

		articles, err := job.Fetcher.FetchSource(fetchCtx, job.Source)
		if err == nil {
			result.Articles = articles
			result.FetchTime = time.Since(start)
			result.RetryCount = attempt
			return result
		}

		lastErr = err
		result.RetryCount = attempt

		// Don't retry on context errors
		if ctx.Err() != nil || fetchCtx.Err() != nil {
			break
		}
	}

	result.Error = lastErr
	result.FetchTime = time.Since(start)
	return result
}

// BatchProcessor handles batch processing of fetch results
type BatchProcessor struct {
	batchSize int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int) *BatchProcessor {
	if batchSize <= 0 {
		batchSize = 100
	}
	return &BatchProcessor{batchSize: batchSize}
}

// CollectArticles collects all articles from results, filtering out errors
func (bp *BatchProcessor) CollectArticles(results []FetchJobResult) ([]models.Article, []FetchJobResult) {
	var allArticles []models.Article
	var errors []FetchJobResult

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, result)
			continue
		}
		allArticles = append(allArticles, result.Articles...)
	}

	return allArticles, errors
}

// ProcessInBatches processes articles in batches, calling the handler for each batch
func (bp *BatchProcessor) ProcessInBatches(articles []models.Article, handler func(batch []models.Article) error) error {
	for i := 0; i < len(articles); i += bp.batchSize {
		end := i + bp.batchSize
		if end > len(articles) {
			end = len(articles)
		}

		if err := handler(articles[i:end]); err != nil {
			return fmt.Errorf("failed to process batch starting at %d: %w", i, err)
		}
	}
	return nil
}

// FetchStats holds statistics about a fetch operation
type FetchStats struct {
	TotalSources     int
	SuccessfulFetches int
	FailedFetches     int
	TotalArticles     int
	NewArticles       int
	TotalFetchTime    time.Duration
	AverageFetchTime  time.Duration
	FastestFetch      time.Duration
	SlowestFetch      time.Duration
}

// CalculateStats calculates statistics from fetch results
func CalculateStats(results []FetchJobResult, newArticles int) FetchStats {
	stats := FetchStats{
		TotalSources: len(results),
		NewArticles:  newArticles,
	}

	if len(results) == 0 {
		return stats
	}

	var totalTime time.Duration
	stats.FastestFetch = time.Hour // Initialize with large value
	stats.SlowestFetch = 0

	for _, r := range results {
		if r.Error != nil {
			stats.FailedFetches++
		} else {
			stats.SuccessfulFetches++
			stats.TotalArticles += len(r.Articles)
		}

		totalTime += r.FetchTime
		if r.FetchTime < stats.FastestFetch {
			stats.FastestFetch = r.FetchTime
		}
		if r.FetchTime > stats.SlowestFetch {
			stats.SlowestFetch = r.FetchTime
		}
	}

	stats.TotalFetchTime = totalTime
	if stats.TotalSources > 0 {
		stats.AverageFetchTime = totalTime / time.Duration(stats.TotalSources)
	}

	return stats
}

// LogStats logs fetch statistics
func (s FetchStats) Log() {
	log.Printf("[stats] Fetch completed:")
	log.Printf("[stats]   Sources: %d total, %d success, %d failed",
		s.TotalSources, s.SuccessfulFetches, s.FailedFetches)
	log.Printf("[stats]   Articles: %d fetched, %d new",
		s.TotalArticles, s.NewArticles)
	log.Printf("[stats]   Timing: avg=%v, min=%v, max=%v",
		s.AverageFetchTime.Round(time.Millisecond),
		s.FastestFetch.Round(time.Millisecond),
		s.SlowestFetch.Round(time.Millisecond))
}
