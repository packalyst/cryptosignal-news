package repository

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"cryptosignal-news/backend/internal/database"
	"cryptosignal-news/backend/internal/models"
)

// sanitizeUTF8 removes invalid UTF8 sequences from a string
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// Remove invalid bytes
	v := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				continue // skip invalid byte
			}
		}
		v = append(v, r)
	}
	return string(v)
}

// ArticleRepository handles article database operations
type ArticleRepository struct {
	db *database.DB
}

// NewArticleRepository creates a new article repository
func NewArticleRepository(db *database.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

// ListOptions defines options for listing articles
type ListOptions struct {
	Limit              int
	Offset             int
	Source             string
	Categories         []string // Filter by multiple categories (OR logic)
	Language           string
	From               *time.Time
	To                 *time.Time
	ExcludeUntranslated bool // If true, exclude articles with translation_status = 'pending' or 'failed'
}

// ListResult contains articles and total count
type ListResult struct {
	Articles []models.Article
	Total    int
}

// List returns a paginated list of articles
func (r *ArticleRepository) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	conditions := []string{"1=1"}
	args := []interface{}{}
	argNum := 1

	if opts.Source != "" {
		conditions = append(conditions, fmt.Sprintf("s.key = $%d", argNum))
		args = append(args, opts.Source)
		argNum++
	}

	if len(opts.Categories) > 0 {
		// Use array overlap operator to match articles that have ANY of the requested categories
		conditions = append(conditions, fmt.Sprintf("a.categories && $%d::text[]", argNum))
		args = append(args, opts.Categories)
		argNum++
	}

	if opts.Language != "" {
		conditions = append(conditions, fmt.Sprintf("s.language = $%d", argNum))
		args = append(args, opts.Language)
		argNum++
	}

	if opts.From != nil {
		conditions = append(conditions, fmt.Sprintf("a.pub_date >= $%d", argNum))
		args = append(args, *opts.From)
		argNum++
	}

	if opts.To != nil {
		conditions = append(conditions, fmt.Sprintf("a.pub_date <= $%d", argNum))
		args = append(args, *opts.To)
		argNum++
	}

	// Exclude untranslated articles if translation filtering is enabled
	if opts.ExcludeUntranslated {
		conditions = append(conditions, "(a.translation_status IS NULL OR a.translation_status IN ('none', 'completed'))")
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM articles a
		JOIN sources s ON a.source_id = s.id
		WHERE %s`, whereClause)

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count articles: %w", err)
	}

	// Fetch articles
	args = append(args, opts.Limit, opts.Offset)
	query := fmt.Sprintf(`
		SELECT
			a.id, a.source_id, a.guid, a.title, a.link, a.description,
			a.pub_date, a.categories, a.sentiment, a.sentiment_score,
			a.mentioned_coins, a.is_breaking, a.created_at,
			s.name as source_name, s.key as source_key
		FROM articles a
		JOIN sources s ON a.source_id = s.id
		WHERE %s
		ORDER BY a.pub_date DESC
		LIMIT $%d OFFSET $%d`, whereClause, argNum, argNum+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	articles, err := r.scanArticles(rows)
	if err != nil {
		return nil, err
	}

	return &ListResult{
		Articles: articles,
		Total:    total,
	}, nil
}

// Search performs full-text search on articles using PostgreSQL's text search
func (r *ArticleRepository) Search(ctx context.Context, queryStr string, limit int, excludeUntranslated bool) ([]models.Article, error) {
	if limit <= 0 {
		limit = 50
	}

	// Build query with optional translation filter
	translationFilter := ""
	if excludeUntranslated {
		translationFilter = " AND (a.translation_status IS NULL OR a.translation_status IN ('none', 'completed'))"
	}

	// Use PostgreSQL full-text search with the GIN index
	query := fmt.Sprintf(`
		SELECT
			a.id, a.source_id, a.guid, a.title, a.link, a.description,
			a.pub_date, a.categories, a.sentiment, a.sentiment_score,
			a.mentioned_coins, a.is_breaking, a.created_at,
			s.name as source_name, s.key as source_key
		FROM articles a
		JOIN sources s ON a.source_id = s.id
		WHERE to_tsvector('english', COALESCE(a.title, '') || ' ' || COALESCE(a.description, ''))
			@@ plainto_tsquery('english', $1)%s
		ORDER BY ts_rank(
			to_tsvector('english', COALESCE(a.title, '') || ' ' || COALESCE(a.description, '')),
			plainto_tsquery('english', $1)
		) DESC, a.pub_date DESC
		LIMIT $2`, translationFilter)

	rows, err := r.db.Query(ctx, query, queryStr, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search articles: %w", err)
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

// GetByID returns a single article by ID
func (r *ArticleRepository) GetByID(ctx context.Context, id int64) (*models.Article, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			a.id, a.source_id, a.guid, a.title, a.link, a.description,
			a.pub_date, a.categories, a.sentiment, a.sentiment_score,
			a.mentioned_coins, a.is_breaking, a.created_at,
			s.name as source_name, s.key as source_key
		FROM articles a
		JOIN sources s ON a.source_id = s.id
		WHERE a.id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}
	defer rows.Close()

	articles, err := r.scanArticles(rows)
	if err != nil {
		return nil, err
	}

	if len(articles) == 0 {
		return nil, nil
	}
	return &articles[0], nil
}

// BulkInsert inserts multiple articles, ignoring duplicates
// Returns the number of articles actually inserted
func (r *ArticleRepository) BulkInsert(ctx context.Context, articles []models.Article) (int, error) {
	if len(articles) == 0 {
		return 0, nil
	}

	// Use batch for efficiency
	const batchSize = 100
	totalInserted := 0

	for i := 0; i < len(articles); i += batchSize {
		end := i + batchSize
		if end > len(articles) {
			end = len(articles)
		}
		batch := articles[i:end]

		inserted, err := r.insertBatch(ctx, batch)
		if err != nil {
			return totalInserted, fmt.Errorf("failed to insert batch: %w", err)
		}
		totalInserted += inserted
	}

	return totalInserted, nil
}

// insertBatch inserts a batch of articles using a single query
func (r *ArticleRepository) insertBatch(ctx context.Context, articles []models.Article) (int, error) {
	// Build the INSERT query with ON CONFLICT DO NOTHING
	valueStrings := make([]string, 0, len(articles))
	valueArgs := make([]interface{}, 0, len(articles)*13)
	argIdx := 1

	for _, a := range articles {
		valueStrings = append(valueStrings,
			fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6, argIdx+7, argIdx+8, argIdx+9, argIdx+10, argIdx+11, argIdx+12))
		valueArgs = append(valueArgs,
			a.SourceID,
			sanitizeUTF8(a.GUID),
			sanitizeUTF8(a.Title),
			sanitizeUTF8(a.Link),
			sanitizeUTF8(a.Description),
			a.PubDate,
			a.Categories,
			a.MentionedCoins,
			a.IsBreaking,
			sanitizeUTF8(a.OriginalTitle),
			sanitizeUTF8(a.OriginalDescription),
			a.OriginalLanguage,
			a.TranslationStatus,
		)
		argIdx += 13
	}

	query := fmt.Sprintf(`
		INSERT INTO articles (source_id, guid, title, link, description, pub_date, categories, mentioned_coins, is_breaking, original_title, original_description, original_language, translation_status)
		VALUES %s
		ON CONFLICT (source_id, guid) DO NOTHING
	`, strings.Join(valueStrings, ", "))

	result, err := r.db.Exec(ctx, query, valueArgs...)
	if err != nil {
		return 0, err
	}

	return int(result), nil
}

// GetPendingTranslations retrieves articles that need translation (includes failed for retry)
func (r *ArticleRepository) GetPendingTranslations(ctx context.Context, limit int) ([]models.Article, error) {
	if limit <= 0 {
		limit = 10
	}

	// Include 'failed' articles for retry - they might succeed after rate limit resets
	// Prioritize 'pending' first, then 'failed'
	rows, err := r.db.Query(ctx, `
		SELECT
			a.id, a.source_id, a.guid, a.title, a.link, a.description,
			a.pub_date, a.categories, a.sentiment, a.sentiment_score,
			a.mentioned_coins, a.is_breaking, a.created_at,
			a.original_title, a.original_description, a.original_language, a.translation_status,
			s.name as source_name, s.key as source_key
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		WHERE a.translation_status IN ('pending', 'failed')
		ORDER BY
			CASE WHEN a.translation_status = 'pending' THEN 0 ELSE 1 END,
			a.id ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending translations: %w", err)
	}
	defer rows.Close()

	return r.scanArticlesWithTranslation(rows)
}

// UpdateTranslation updates an article with its translation
func (r *ArticleRepository) UpdateTranslation(ctx context.Context, id int64, title, description, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE articles
		SET title = $2, description = $3, translation_status = $4
		WHERE id = $1
	`, id, sanitizeUTF8(title), sanitizeUTF8(description), status)
	if err != nil {
		return fmt.Errorf("failed to update translation: %w", err)
	}
	return nil
}

// CountPendingTranslations returns the number of articles pending translation
func (r *ArticleRepository) CountPendingTranslations(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM articles WHERE translation_status = 'pending'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending translations: %w", err)
	}
	return count, nil
}

// TranslationStats holds translation statistics
type TranslationStats struct {
	TotalArticles int            `json:"total_articles"`
	ByStatus      map[string]int `json:"by_status"`
	ByLanguage    map[string]int `json:"by_language"`
}

// GetTranslationStats returns detailed translation statistics
func (r *ArticleRepository) GetTranslationStats(ctx context.Context) (*TranslationStats, error) {
	stats := &TranslationStats{
		ByStatus:   make(map[string]int),
		ByLanguage: make(map[string]int),
	}

	// Get total count
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM articles`).Scan(&stats.TotalArticles)
	if err != nil {
		return nil, fmt.Errorf("failed to count articles: %w", err)
	}

	// Get counts by translation status
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(translation_status, 'none'), COUNT(*)
		FROM articles
		GROUP BY translation_status
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get translation status counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.ByStatus[status] = count
	}

	// Get counts by original language (for non-English articles)
	rows2, err := r.db.Query(ctx, `
		SELECT COALESCE(original_language, 'en'), COUNT(*)
		FROM articles
		WHERE original_language IS NOT NULL AND original_language != ''
		GROUP BY original_language
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get language counts: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var lang string
		var count int
		if err := rows2.Scan(&lang, &count); err != nil {
			return nil, err
		}
		stats.ByLanguage[lang] = count
	}

	return stats, nil
}

// scanArticlesWithTranslation scans rows including translation fields
func (r *ArticleRepository) scanArticlesWithTranslation(rows pgx.Rows) ([]models.Article, error) {
	articles := []models.Article{}

	for rows.Next() {
		var a models.Article
		var sentiment, sourceName, sourceKey *string
		var sentimentScore *float64
		var origTitle, origDesc, origLang, transStatus *string

		err := rows.Scan(
			&a.ID,
			&a.SourceID,
			&a.GUID,
			&a.Title,
			&a.Link,
			&a.Description,
			&a.PubDate,
			&a.Categories,
			&sentiment,
			&sentimentScore,
			&a.MentionedCoins,
			&a.IsBreaking,
			&a.CreatedAt,
			&origTitle,
			&origDesc,
			&origLang,
			&transStatus,
			&sourceName,
			&sourceKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		if sentiment != nil {
			a.Sentiment = *sentiment
		}
		if sentimentScore != nil {
			a.SentimentScore = *sentimentScore
		}
		if sourceName != nil {
			a.SourceName = *sourceName
		}
		if sourceKey != nil {
			a.SourceKey = *sourceKey
		}
		if origTitle != nil {
			a.OriginalTitle = *origTitle
		}
		if origDesc != nil {
			a.OriginalDescription = *origDesc
		}
		if origLang != nil {
			a.OriginalLanguage = *origLang
		}
		if transStatus != nil {
			a.TranslationStatus = *transStatus
		}

		articles = append(articles, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating articles: %w", err)
	}

	return articles, nil
}

// Exists checks if an article with the given GUID exists
func (r *ArticleRepository) Exists(ctx context.Context, guid string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM articles WHERE guid = $1)",
		guid,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check article existence: %w", err)
	}
	return exists, nil
}

// ExistsBatch checks existence for multiple GUIDs at once
// Returns a map of GUID -> exists
func (r *ArticleRepository) ExistsBatch(ctx context.Context, guids []string) (map[string]bool, error) {
	result := make(map[string]bool, len(guids))
	if len(guids) == 0 {
		return result, nil
	}

	rows, err := r.db.Query(ctx,
		"SELECT guid FROM articles WHERE guid = ANY($1)",
		guids,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check article existence batch: %w", err)
	}
	defer rows.Close()

	existingGuids := make(map[string]bool)
	for rows.Next() {
		var guid string
		if err := rows.Scan(&guid); err != nil {
			return nil, err
		}
		existingGuids[guid] = true
	}

	for _, guid := range guids {
		result[guid] = existingGuids[guid]
	}

	return result, nil
}

// GetLatest retrieves the most recent articles
func (r *ArticleRepository) GetLatest(ctx context.Context, limit int, excludeUntranslated bool) ([]models.Article, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	query := `
		SELECT
			a.id, a.source_id, a.guid, a.title, a.link, a.description,
			a.pub_date, a.categories, a.sentiment, a.sentiment_score,
			a.mentioned_coins, a.is_breaking, a.created_at,
			s.name as source_name, s.key as source_key
		FROM articles a
		JOIN sources s ON s.id = a.source_id`

	if excludeUntranslated {
		query += ` WHERE (a.translation_status IS NULL OR a.translation_status IN ('none', 'completed'))`
	}

	query += ` ORDER BY a.pub_date DESC LIMIT $1`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest articles: %w", err)
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

// GetBySource retrieves articles from a specific source
func (r *ArticleRepository) GetBySource(ctx context.Context, sourceID int, limit int) ([]models.Article, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			a.id, a.source_id, a.guid, a.title, a.link, a.description,
			a.pub_date, a.categories, a.sentiment, a.sentiment_score,
			a.mentioned_coins, a.is_breaking, a.created_at,
			s.name as source_name, s.key as source_key
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		WHERE a.source_id = $1
		ORDER BY a.pub_date DESC
		LIMIT $2
	`, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles by source: %w", err)
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

// GetBreaking retrieves breaking news articles (less than 2 hours old)
func (r *ArticleRepository) GetBreaking(ctx context.Context, limit int, excludeUntranslated bool) ([]models.Article, error) {
	if limit <= 0 {
		limit = 20
	}

	twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour)

	query := `
		SELECT
			a.id, a.source_id, a.guid, a.title, a.link, a.description,
			a.pub_date, a.categories, a.sentiment, a.sentiment_score,
			a.mentioned_coins, a.is_breaking, a.created_at,
			s.name as source_name, s.key as source_key
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		WHERE (a.pub_date >= $1 OR a.is_breaking = true)`

	if excludeUntranslated {
		query += ` AND (a.translation_status IS NULL OR a.translation_status IN ('none', 'completed'))`
	}

	query += ` ORDER BY a.pub_date DESC LIMIT $2`

	rows, err := r.db.Query(ctx, query, twoHoursAgo, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get breaking articles: %w", err)
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

// GetByCoin retrieves articles mentioning a specific cryptocurrency
func (r *ArticleRepository) GetByCoin(ctx context.Context, coin string, limit int, excludeUntranslated bool) ([]models.Article, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT
			a.id, a.source_id, a.guid, a.title, a.link, a.description,
			a.pub_date, a.categories, a.sentiment, a.sentiment_score,
			a.mentioned_coins, a.is_breaking, a.created_at,
			s.name as source_name, s.key as source_key
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		WHERE $1 = ANY(a.mentioned_coins)`

	if excludeUntranslated {
		query += ` AND (a.translation_status IS NULL OR a.translation_status IN ('none', 'completed'))`
	}

	query += ` ORDER BY a.pub_date DESC LIMIT $2`

	rows, err := r.db.Query(ctx, query, strings.ToUpper(coin), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles by coin: %w", err)
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

// CountBySource returns article counts grouped by source
func (r *ArticleRepository) CountBySource(ctx context.Context, since time.Time) (map[int]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT source_id, COUNT(*) as count
		FROM articles
		WHERE created_at >= $1
		GROUP BY source_id
	`, since)
	if err != nil {
		return nil, fmt.Errorf("failed to count articles by source: %w", err)
	}
	defer rows.Close()

	result := make(map[int]int)
	for rows.Next() {
		var sourceID, count int
		if err := rows.Scan(&sourceID, &count); err != nil {
			return nil, err
		}
		result[sourceID] = count
	}

	return result, nil
}

// scanArticles scans rows into article structs
func (r *ArticleRepository) scanArticles(rows pgx.Rows) ([]models.Article, error) {
	articles := []models.Article{}

	for rows.Next() {
		var a models.Article
		var sentiment, sourceName, sourceKey *string
		var sentimentScore *float64

		err := rows.Scan(
			&a.ID,
			&a.SourceID,
			&a.GUID,
			&a.Title,
			&a.Link,
			&a.Description,
			&a.PubDate,
			&a.Categories,
			&sentiment,
			&sentimentScore,
			&a.MentionedCoins,
			&a.IsBreaking,
			&a.CreatedAt,
			&sourceName,
			&sourceKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		if sentiment != nil {
			a.Sentiment = *sentiment
		}
		if sentimentScore != nil {
			a.SentimentScore = *sentimentScore
		}
		if sourceName != nil {
			a.SourceName = *sourceName
		}
		if sourceKey != nil {
			a.SourceKey = *sourceKey
		}

		articles = append(articles, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating articles: %w", err)
	}

	return articles, nil
}
