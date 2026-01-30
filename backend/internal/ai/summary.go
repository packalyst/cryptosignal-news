package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// MarketSummary represents a daily market summary
type MarketSummary struct {
	OverallSentiment string   `json:"overall_sentiment"`
	Summary          string   `json:"summary"`
	KeyDevelopments  []string `json:"key_developments"`
	MentionedCoins   []string `json:"mentioned_coins"`
	NotableEvents    []string `json:"notable_events"`
	GeneratedAt      string   `json:"generated_at"`
	ArticleCount     int      `json:"article_count"`
}

// SummaryService handles market summary generation
type SummaryService struct {
	groq  *GroqClient
	cache *AICache
}

// NewSummaryService creates a new summary service
func NewSummaryService(groq *GroqClient, cache *AICache) *SummaryService {
	return &SummaryService{
		groq:  groq,
		cache: cache,
	}
}

// GenerateDailySummary generates a market summary from recent articles
func (s *SummaryService) GenerateDailySummary(ctx context.Context, articles []Article) (*MarketSummary, error) {
	if len(articles) == 0 {
		return &MarketSummary{
			OverallSentiment: "neutral",
			Summary:          "No recent articles available for analysis.",
			KeyDevelopments:  []string{},
			MentionedCoins:   []string{},
			NotableEvents:    []string{},
			GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
			ArticleCount:     0,
		}, nil
	}

	// Convert articles to summary format
	articleSummaries := make([]ArticleSummary, 0, len(articles))
	for _, article := range articles {
		articleSummaries = append(articleSummaries, ArticleSummary{
			Title:   article.Title,
			Source:  article.Source,
			TimeAgo: formatTimeAgo(article.PubDate),
		})
	}

	// Limit to most recent 50 articles for the prompt
	if len(articleSummaries) > 50 {
		articleSummaries = articleSummaries[:50]
	}

	// Render the prompt
	prompt, err := RenderSummaryPrompt(articleSummaries)
	if err != nil {
		return nil, fmt.Errorf("failed to render summary prompt: %w", err)
	}

	// Make API request
	req := &ChatRequest{
		Model:       DefaultGroqModel,
		Temperature: 0.5,
		MaxTokens:   2048,
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "You are a crypto market analyst. Analyze news articles and provide comprehensive market summaries. Respond ONLY with valid JSON. No markdown, no explanations.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	resp, err := s.groq.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Parse the response
	content := resp.GetMessageContent()
	summary, err := parseSummaryResponse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse summary response: %w", err)
	}

	// Set metadata
	summary.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	summary.ArticleCount = len(articles)

	// Cache the result
	if s.cache != nil {
		if cacheErr := s.cache.SetSummary(ctx, summary); cacheErr != nil {
			log.Printf("warning: failed to cache summary: %v", cacheErr)
		}
	}

	return summary, nil
}

// GetCachedSummary retrieves a cached summary if available
func (s *SummaryService) GetCachedSummary(ctx context.Context) (*MarketSummary, error) {
	if s.cache == nil {
		return nil, nil
	}
	return s.cache.GetSummary(ctx)
}

// GetOrGenerateSummary returns cached summary or generates a new one
func (s *SummaryService) GetOrGenerateSummary(ctx context.Context, articles []Article) (*MarketSummary, error) {
	// Try to get cached summary first
	if s.cache != nil {
		cached, err := s.cache.GetSummary(ctx)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Generate new summary
	return s.GenerateDailySummary(ctx, articles)
}

// InvalidateCache invalidates the cached summary
func (s *SummaryService) InvalidateCache(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.InvalidateSummary(ctx)
}

// parseSummaryResponse parses the JSON response from the LLM
func parseSummaryResponse(content string) (*MarketSummary, error) {
	// Clean up the response
	content = cleanJSONResponse(content)

	var result MarketSummary
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// Try to extract JSON from the response
		jsonStr := extractJSON(content)
		if jsonStr != "" {
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response: %w", err)
		}
	}

	// Normalize sentiment
	result.OverallSentiment = normalizeSentiment(result.OverallSentiment)

	// Ensure slices are not nil
	if result.KeyDevelopments == nil {
		result.KeyDevelopments = []string{}
	}
	if result.MentionedCoins == nil {
		result.MentionedCoins = []string{}
	}
	if result.NotableEvents == nil {
		result.NotableEvents = []string{}
	}

	return &result, nil
}

// formatTimeAgo formats a time as a human-readable "time ago" string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
