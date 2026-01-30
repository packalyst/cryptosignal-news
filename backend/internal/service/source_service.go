package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cryptosignal-news/backend/internal/cache"
	"github.com/cryptosignal-news/backend/internal/models"
	"github.com/cryptosignal-news/backend/internal/repository"
)

// SourceService handles business logic for source operations
type SourceService struct {
	repo  *repository.SourceRepository
	cache *cache.Redis
}

// NewSourceService creates a new source service
func NewSourceService(repo *repository.SourceRepository, cache *cache.Redis) *SourceService {
	return &SourceService{
		repo:  repo,
		cache: cache,
	}
}

// SourceWithCount represents a source with its article count for API response
type SourceWithCount struct {
	ID               int        `json:"id"`
	Key              string     `json:"key"`
	Name             string     `json:"name"`
	WebsiteURL       string     `json:"website_url,omitempty"`
	Category         string     `json:"category,omitempty"`
	Language         string     `json:"language"`
	IsEnabled        bool       `json:"is_enabled"`
	ReliabilityScore float64    `json:"reliability_score"`
	LastFetchAt      *time.Time `json:"last_fetch_at,omitempty"`
	ArticleCount     int        `json:"article_count"`
}

// ListSources returns all sources with article counts
func (s *SourceService) ListSources(ctx context.Context) ([]SourceWithCount, error) {
	// Generate cache key
	cacheKey := "sources:list"

	// Try to get from cache
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var result []SourceWithCount
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result, nil
		}
	}

	// Query from database
	sources, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	result := make([]SourceWithCount, len(sources))
	for i, src := range sources {
		result[i] = SourceWithCount{
			ID:               src.ID,
			Key:              src.Key,
			Name:             src.Name,
			WebsiteURL:       src.WebsiteURL,
			Category:         src.Category,
			Language:         src.Language,
			IsEnabled:        src.IsEnabled,
			ReliabilityScore: src.ReliabilityScore,
			LastFetchAt:      src.LastFetchAt,
			ArticleCount:     src.ArticleCount,
		}
	}

	// Cache the result
	if data, err := json.Marshal(result); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return result, nil
}

// GetCategories returns all categories with article counts
func (s *SourceService) GetCategories(ctx context.Context) ([]models.Category, error) {
	// Generate cache key
	cacheKey := "categories:list"

	// Try to get from cache
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != "" {
		var result []models.Category
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return result, nil
		}
	}

	// Query from database
	categories, err := s.repo.GetCategories(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(categories); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return categories, nil
}
