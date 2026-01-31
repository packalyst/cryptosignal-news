package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"cryptosignal-news/backend/internal/database"
	"cryptosignal-news/backend/internal/models"
)

// SourceRepository handles source database operations
type SourceRepository struct {
	db *database.DB
}

// NewSourceRepository creates a new source repository
func NewSourceRepository(db *database.DB) *SourceRepository {
	return &SourceRepository{db: db}
}

// SourceWithCount represents a source with its article count
type SourceWithCount struct {
	models.Source
	ArticleCount int `json:"article_count" db:"article_count"`
}

// List returns all sources with article counts
func (r *SourceRepository) List(ctx context.Context) ([]SourceWithCount, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			s.id, s.key, s.name, s.rss_url, s.website_url, s.category,
			s.language, s.is_enabled, s.reliability_score, s.last_fetch_at,
			s.error_count, s.created_at,
			COUNT(a.id) as article_count
		FROM sources s
		LEFT JOIN articles a ON s.id = a.source_id
		GROUP BY s.id
		ORDER BY s.name`)
	if err != nil {
		return nil, fmt.Errorf("failed to query sources: %w", err)
	}
	defer rows.Close()

	sources := []SourceWithCount{}
	for rows.Next() {
		var s SourceWithCount
		var websiteURL, category *string
		err := rows.Scan(
			&s.ID, &s.Key, &s.Name, &s.RSSURL, &websiteURL, &category,
			&s.Language, &s.IsEnabled, &s.ReliabilityScore, &s.LastFetchAt,
			&s.ErrorCount, &s.CreatedAt, &s.ArticleCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		if websiteURL != nil {
			s.WebsiteURL = *websiteURL
		}
		if category != nil {
			s.Category = *category
		}
		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return sources, nil
}

// GetCategories returns all categories with article counts
func (r *SourceRepository) GetCategories(ctx context.Context) ([]models.Category, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			unnest(categories) as name,
			COUNT(*) as count
		FROM articles
		WHERE categories IS NOT NULL AND array_length(categories, 1) > 0
		GROUP BY unnest(categories)
		ORDER BY count DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	categories := []models.Category{}
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.Name, &c.Count); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return categories, nil
}

// GetAll retrieves all sources
func (r *SourceRepository) GetAll(ctx context.Context) ([]models.Source, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, key, name, rss_url, website_url, category, language,
		       is_enabled, reliability_score, last_fetch_at, error_count, created_at
		FROM sources
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all sources: %w", err)
	}
	defer rows.Close()

	return r.scanSources(rows)
}

// GetEnabled retrieves all enabled sources
func (r *SourceRepository) GetEnabled(ctx context.Context) ([]models.Source, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, key, name, rss_url, website_url, category, language,
		       is_enabled, reliability_score, last_fetch_at, error_count, created_at
		FROM sources
		WHERE is_enabled = true
		ORDER BY reliability_score DESC, name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled sources: %w", err)
	}
	defer rows.Close()

	return r.scanSources(rows)
}

// GetByID retrieves a source by ID
func (r *SourceRepository) GetByID(ctx context.Context, id int) (*models.Source, error) {
	var s models.Source
	var websiteURL, category *string

	err := r.db.QueryRow(ctx, `
		SELECT id, key, name, rss_url, website_url, category, language,
		       is_enabled, reliability_score, last_fetch_at, error_count, created_at
		FROM sources
		WHERE id = $1
	`, id).Scan(
		&s.ID, &s.Key, &s.Name, &s.RSSURL, &websiteURL, &category,
		&s.Language, &s.IsEnabled, &s.ReliabilityScore, &s.LastFetchAt,
		&s.ErrorCount, &s.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source by ID: %w", err)
	}

	if websiteURL != nil {
		s.WebsiteURL = *websiteURL
	}
	if category != nil {
		s.Category = *category
	}

	return &s, nil
}

// GetByKey retrieves a source by key
func (r *SourceRepository) GetByKey(ctx context.Context, key string) (*models.Source, error) {
	var s models.Source
	var websiteURL, category *string

	err := r.db.QueryRow(ctx, `
		SELECT id, key, name, rss_url, website_url, category, language,
		       is_enabled, reliability_score, last_fetch_at, error_count, created_at
		FROM sources
		WHERE key = $1
	`, key).Scan(
		&s.ID, &s.Key, &s.Name, &s.RSSURL, &websiteURL, &category,
		&s.Language, &s.IsEnabled, &s.ReliabilityScore, &s.LastFetchAt,
		&s.ErrorCount, &s.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get source by key: %w", err)
	}

	if websiteURL != nil {
		s.WebsiteURL = *websiteURL
	}
	if category != nil {
		s.Category = *category
	}

	return &s, nil
}

// UpdateLastFetch updates the last fetch timestamp for a source
func (r *SourceRepository) UpdateLastFetch(ctx context.Context, sourceID int, fetchedAt time.Time) error {
	_, err := r.db.Exec(ctx,
		"UPDATE sources SET last_fetch_at = $1 WHERE id = $2",
		fetchedAt, sourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to update last fetch time: %w", err)
	}
	return nil
}

// IncrementErrorCount increments the error count for a source
func (r *SourceRepository) IncrementErrorCount(ctx context.Context, sourceID int) error {
	_, err := r.db.Exec(ctx,
		"UPDATE sources SET error_count = error_count + 1 WHERE id = $1",
		sourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to increment error count: %w", err)
	}
	return nil
}

// ResetErrorCount resets the error count for a source
func (r *SourceRepository) ResetErrorCount(ctx context.Context, sourceID int) error {
	_, err := r.db.Exec(ctx,
		"UPDATE sources SET error_count = 0 WHERE id = $1",
		sourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to reset error count: %w", err)
	}
	return nil
}

// UpdateReliabilityScore updates the reliability score for a source
func (r *SourceRepository) UpdateReliabilityScore(ctx context.Context, sourceID int, score float64) error {
	_, err := r.db.Exec(ctx,
		"UPDATE sources SET reliability_score = $1 WHERE id = $2",
		score, sourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to update reliability score: %w", err)
	}
	return nil
}

// DisableSource disables a source
func (r *SourceRepository) DisableSource(ctx context.Context, sourceID int) error {
	_, err := r.db.Exec(ctx,
		"UPDATE sources SET is_enabled = false WHERE id = $1",
		sourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to disable source: %w", err)
	}
	return nil
}

// EnableSource enables a source
func (r *SourceRepository) EnableSource(ctx context.Context, sourceID int) error {
	_, err := r.db.Exec(ctx,
		"UPDATE sources SET is_enabled = true, error_count = 0 WHERE id = $1",
		sourceID,
	)
	if err != nil {
		return fmt.Errorf("failed to enable source: %w", err)
	}
	return nil
}

// GetSourceStats returns statistics about sources
func (r *SourceRepository) GetSourceStats(ctx context.Context) ([]models.SourceStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			s.id as source_id,
			s.key as source_key,
			COUNT(a.id) as articles_fetched,
			s.error_count,
			COALESCE(s.last_fetch_at, s.created_at) as last_fetch_at
		FROM sources s
		LEFT JOIN articles a ON a.source_id = s.id AND a.created_at >= NOW() - INTERVAL '24 hours'
		WHERE s.is_enabled = true
		GROUP BY s.id, s.key, s.error_count, s.last_fetch_at, s.created_at
		ORDER BY articles_fetched DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get source stats: %w", err)
	}
	defer rows.Close()

	stats := []models.SourceStats{}
	for rows.Next() {
		var s models.SourceStats
		if err := rows.Scan(&s.SourceID, &s.SourceKey, &s.ArticlesFetched, &s.ErrorCount, &s.LastFetchAt); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}

// GetUnhealthySources returns sources with high error counts
func (r *SourceRepository) GetUnhealthySources(ctx context.Context, errorThreshold int) ([]models.Source, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, key, name, rss_url, website_url, category, language,
		       is_enabled, reliability_score, last_fetch_at, error_count, created_at
		FROM sources
		WHERE error_count >= $1
		ORDER BY error_count DESC
	`, errorThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get unhealthy sources: %w", err)
	}
	defer rows.Close()

	return r.scanSources(rows)
}

// scanSources scans rows into source structs
func (r *SourceRepository) scanSources(rows pgx.Rows) ([]models.Source, error) {
	sources := []models.Source{}

	for rows.Next() {
		var s models.Source
		var websiteURL, category *string

		err := rows.Scan(
			&s.ID, &s.Key, &s.Name, &s.RSSURL, &websiteURL, &category,
			&s.Language, &s.IsEnabled, &s.ReliabilityScore, &s.LastFetchAt,
			&s.ErrorCount, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}

		if websiteURL != nil {
			s.WebsiteURL = *websiteURL
		}
		if category != nil {
			s.Category = *category
		}

		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sources: %w", err)
	}

	return sources, nil
}
