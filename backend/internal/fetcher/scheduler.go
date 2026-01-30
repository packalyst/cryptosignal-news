package fetcher

import (
	"context"
	"log"
	"sync"
	"time"
)

// Scheduler manages periodic fetch operations
type Scheduler struct {
	fetcher     *Fetcher
	interval    time.Duration
	stopCh      chan struct{}
	doneCh      chan struct{}
	mu          sync.Mutex
	running     bool
	lastFetch   time.Time
	lastResult  *FetchResult
	fetchCount  int64
	errorCount  int64
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	Interval time.Duration
}

// DefaultSchedulerConfig returns default scheduler configuration
func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		Interval: 3 * time.Minute,
	}
}

// NewScheduler creates a new scheduler
func NewScheduler(fetcher *Fetcher, cfg *SchedulerConfig) *Scheduler {
	if cfg == nil {
		cfg = DefaultSchedulerConfig()
	}

	return &Scheduler{
		fetcher:  fetcher,
		interval: cfg.Interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins the periodic fetch loop
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		log.Println("[scheduler] Already running")
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Printf("[scheduler] Starting with interval: %v", s.interval)

	// Run initial fetch immediately
	s.runFetch(ctx)

	// Start ticker for periodic fetches
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[scheduler] Context cancelled, stopping")
			s.markStopped()
			return

		case <-s.stopCh:
			log.Println("[scheduler] Stop signal received")
			s.markStopped()
			return

		case <-ticker.C:
			s.runFetch(ctx)
		}
	}
}

// Stop signals the scheduler to stop
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	log.Println("[scheduler] Stopping...")
	close(s.stopCh)

	// Wait for the scheduler to finish with timeout
	select {
	case <-s.doneCh:
		log.Println("[scheduler] Stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("[scheduler] Stop timed out")
	}
}

// markStopped marks the scheduler as stopped
func (s *Scheduler) markStopped() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
	close(s.doneCh)
}

// runFetch executes a single fetch operation
func (s *Scheduler) runFetch(ctx context.Context) {
	log.Println("[scheduler] Starting fetch cycle")
	start := time.Now()

	result, err := s.fetcher.FetchAll(ctx)

	s.mu.Lock()
	s.lastFetch = start
	s.fetchCount++
	if err != nil {
		s.errorCount++
		log.Printf("[scheduler] Fetch failed: %v", err)
	} else {
		s.lastResult = result
	}
	s.mu.Unlock()

	if result != nil {
		log.Printf("[scheduler] Fetch completed: %d new articles from %d sources in %v",
			result.NewArticles, result.SuccessfulFeeds, result.Duration.Round(time.Millisecond))
	}
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// GetStats returns scheduler statistics
func (s *Scheduler) GetStats() SchedulerStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats := SchedulerStats{
		Running:    s.running,
		Interval:   s.interval,
		LastFetch:  s.lastFetch,
		FetchCount: s.fetchCount,
		ErrorCount: s.errorCount,
	}

	if s.lastResult != nil {
		stats.LastSuccessfulFeeds = s.lastResult.SuccessfulFeeds
		stats.LastFailedFeeds = s.lastResult.FailedFeeds
		stats.LastNewArticles = s.lastResult.NewArticles
		stats.LastDuration = s.lastResult.Duration
	}

	return stats
}

// SchedulerStats contains scheduler statistics
type SchedulerStats struct {
	Running             bool          `json:"running"`
	Interval            time.Duration `json:"interval"`
	LastFetch           time.Time     `json:"last_fetch"`
	FetchCount          int64         `json:"fetch_count"`
	ErrorCount          int64         `json:"error_count"`
	LastSuccessfulFeeds int           `json:"last_successful_feeds"`
	LastFailedFeeds     int           `json:"last_failed_feeds"`
	LastNewArticles     int           `json:"last_new_articles"`
	LastDuration        time.Duration `json:"last_duration"`
}

// NextFetchIn returns the duration until the next scheduled fetch
func (s *Scheduler) NextFetchIn() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.lastFetch.IsZero() {
		return 0
	}

	nextFetch := s.lastFetch.Add(s.interval)
	until := time.Until(nextFetch)
	if until < 0 {
		return 0
	}
	return until
}

// RunOnce runs a single fetch operation (useful for testing or manual triggers)
func (s *Scheduler) RunOnce(ctx context.Context) (*FetchResult, error) {
	return s.fetcher.FetchAll(ctx)
}

// SetInterval updates the fetch interval
func (s *Scheduler) SetInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = interval
	log.Printf("[scheduler] Interval updated to: %v", interval)
}
