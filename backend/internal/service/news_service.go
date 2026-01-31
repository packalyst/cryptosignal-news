package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"cryptosignal-news/backend/internal/cache"
	"cryptosignal-news/backend/internal/models"
	"cryptosignal-news/backend/internal/repository"
)

// NewsService handles business logic for news operations
type NewsService struct {
	repo                 *repository.ArticleRepository
	cache                *cache.Redis
	excludeUntranslated  bool
}

// NewNewsService creates a new news service
func NewNewsService(repo *repository.ArticleRepository, cache *cache.Redis, excludeUntranslated bool) *NewsService {
	return &NewsService{
		repo:                repo,
		cache:               cache,
		excludeUntranslated: excludeUntranslated,
	}
}

// ListOptions defines options for listing articles
type ListOptions struct {
	Limit      int
	Offset     int
	Source     string
	Categories []string // Filter by multiple categories (comma-separated in API)
	Language   string
	From       *time.Time
	To         *time.Time
}

// NewsResult contains the result of a news list operation
type NewsResult struct {
	Articles []models.ArticleResponse `json:"articles"`
	Total    int                      `json:"total"`
}

// GetLatest returns the latest news articles
func (s *NewsService) GetLatest(ctx context.Context, opts ListOptions) (*NewsResult, error) {
	// Generate cache key (include categories as joined string for cache key)
	categoriesKey := strings.Join(opts.Categories, ",")
	cacheKey := cache.GenerateCacheKey("news:latest", opts.Limit, opts.Offset, opts.Source, categoriesKey, opts.Language)

	// Try to get from cache
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var result NewsResult
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return &result, nil
		}
	}

	// Query from database
	repoOpts := repository.ListOptions{
		Limit:               opts.Limit,
		Offset:              opts.Offset,
		Source:              opts.Source,
		Categories:          opts.Categories,
		Language:            opts.Language,
		From:                opts.From,
		To:                  opts.To,
		ExcludeUntranslated: s.excludeUntranslated,
	}

	listResult, err := s.repo.List(ctx, repoOpts)
	if err != nil {
		return nil, err
	}

	// Convert to response format (pass filter categories to show only matched ones)
	articles := make([]models.ArticleResponse, len(listResult.Articles))
	for i, a := range listResult.Articles {
		articles[i] = a.ToResponseWithFilter(opts.Categories)
	}

	result := &NewsResult{
		Articles: articles,
		Total:    listResult.Total,
	}

	// Cache the result
	if data, err := json.Marshal(result); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 60*time.Second)
	}

	return result, nil
}

// GetBreaking returns breaking news from the last 2 hours
func (s *NewsService) GetBreaking(ctx context.Context, limit int) ([]models.ArticleResponse, error) {
	// Generate cache key
	cacheKey := cache.GenerateCacheKey("news:breaking", limit)

	// Try to get from cache
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var result []models.ArticleResponse
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result, nil
		}
	}

	// Query from database
	articles, err := s.repo.GetBreaking(ctx, limit, s.excludeUntranslated)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	result := make([]models.ArticleResponse, len(articles))
	for i, a := range articles {
		result[i] = a.ToResponse()
	}

	// Cache the result (shorter TTL for breaking news)
	if data, err := json.Marshal(result); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 30*time.Second)
	}

	return result, nil
}

// Search performs full-text search on articles
func (s *NewsService) Search(ctx context.Context, query string, limit int) ([]models.ArticleResponse, error) {
	// Generate cache key
	cacheKey := cache.GenerateCacheKey("news:search", query, limit)

	// Try to get from cache
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var result []models.ArticleResponse
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result, nil
		}
	}

	// Query from database
	articles, err := s.repo.Search(ctx, query, limit, s.excludeUntranslated)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	result := make([]models.ArticleResponse, len(articles))
	for i, a := range articles {
		result[i] = a.ToResponse()
	}

	// Cache the result
	if data, err := json.Marshal(result); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 60*time.Second)
	}

	return result, nil
}

// GetByID returns a single article by ID
func (s *NewsService) GetByID(ctx context.Context, id int64) (*models.ArticleResponse, error) {
	// Generate cache key
	cacheKey := cache.GenerateCacheKey("news:article", id)

	// Try to get from cache
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var result models.ArticleResponse
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return &result, nil
		}
	}

	// Query from database
	article, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, nil
	}

	// Convert to response format
	result := article.ToResponse()

	// Cache the result (longer TTL for individual articles)
	if data, err := json.Marshal(result); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return &result, nil
}

// GetByCoin returns articles mentioning a specific coin
func (s *NewsService) GetByCoin(ctx context.Context, symbol string, limit int) ([]models.ArticleResponse, error) {
	// Generate cache key
	cacheKey := cache.GenerateCacheKey("news:coin", symbol, limit)

	// Try to get from cache
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var result []models.ArticleResponse
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result, nil
		}
	}

	// Query from database
	articles, err := s.repo.GetByCoin(ctx, symbol, limit, s.excludeUntranslated)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	result := make([]models.ArticleResponse, len(articles))
	for i, a := range articles {
		result[i] = a.ToResponse()
	}

	// Cache the result
	if data, err := json.Marshal(result); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 60*time.Second)
	}

	return result, nil
}
