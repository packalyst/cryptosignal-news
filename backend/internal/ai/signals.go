package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// TradingSignal represents a trading signal derived from news
type TradingSignal struct {
	Coin        string `json:"coin"`
	Direction   string `json:"direction"`
	Strength    string `json:"strength"`
	Catalyst    string `json:"catalyst"`
	SourceTitle string `json:"source_title"`
}

// SignalsResult represents the result of signal generation
type SignalsResult struct {
	Signals      []TradingSignal `json:"signals"`
	MarketMood   string          `json:"market_mood"`
	GeneratedAt  string          `json:"generated_at"`
	ArticleCount int             `json:"article_count"`
}

// SignalsService handles trading signal generation from news
type SignalsService struct {
	groq  *GroqClient
	cache *AICache
	model string
}

// NewSignalsService creates a new signals service
func NewSignalsService(groq *GroqClient, cache *AICache, model string) *SignalsService {
	if model == "" {
		model = DefaultGroqModel
	}
	return &SignalsService{
		groq:  groq,
		cache: cache,
		model: model,
	}
}

// GenerateSignals generates trading signals from recent articles
func (s *SignalsService) GenerateSignals(ctx context.Context, articles []Article) (*SignalsResult, error) {
	if len(articles) == 0 {
		return &SignalsResult{
			Signals:      []TradingSignal{},
			MarketMood:   "neutral",
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
			ArticleCount: 0,
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

	// Limit to most recent 30 articles for signal generation
	if len(articleSummaries) > 30 {
		articleSummaries = articleSummaries[:30]
	}

	// Render the prompt
	prompt, err := RenderSignalsPrompt(articleSummaries)
	if err != nil {
		return nil, fmt.Errorf("failed to render signals prompt: %w", err)
	}

	// Make API request
	req := &ChatRequest{
		Model:       s.model,
		Temperature: 0.4,
		MaxTokens:   1024,
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "You are a crypto trading signal analyst. Identify potential trading opportunities from news. Be conservative - only flag strong signals. Respond ONLY with valid JSON. No markdown, no explanations.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	resp, err := s.groq.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signals: %w", err)
	}

	// Parse the response
	content := resp.GetMessageContent()
	result, err := parseSignalsResponse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signals response: %w", err)
	}

	// Set metadata
	result.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	result.ArticleCount = len(articles)

	// Cache the result
	if s.cache != nil {
		if cacheErr := s.cache.SetSignals(ctx, result); cacheErr != nil {
			log.Printf("warning: failed to cache signals: %v", cacheErr)
		}
	}

	return result, nil
}

// GetCachedSignals retrieves cached signals if available
func (s *SignalsService) GetCachedSignals(ctx context.Context) (*SignalsResult, error) {
	if s.cache == nil {
		return nil, nil
	}
	return s.cache.GetSignals(ctx)
}

// GetOrGenerateSignals returns cached signals or generates new ones
func (s *SignalsService) GetOrGenerateSignals(ctx context.Context, articles []Article) (*SignalsResult, error) {
	// Try to get cached signals first
	if s.cache != nil {
		cached, err := s.cache.GetSignals(ctx)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Generate new signals
	return s.GenerateSignals(ctx, articles)
}

// InvalidateCache invalidates the cached signals
func (s *SignalsService) InvalidateCache(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.InvalidateSignals(ctx)
}

// FilterByStrength filters signals by minimum strength
func (s *SignalsService) FilterByStrength(signals []TradingSignal, minStrength string) []TradingSignal {
	strengthOrder := map[string]int{
		"weak":     1,
		"moderate": 2,
		"strong":   3,
	}

	minLevel, ok := strengthOrder[strings.ToLower(minStrength)]
	if !ok {
		return signals
	}

	filtered := make([]TradingSignal, 0)
	for _, signal := range signals {
		if level, ok := strengthOrder[strings.ToLower(signal.Strength)]; ok && level >= minLevel {
			filtered = append(filtered, signal)
		}
	}

	return filtered
}

// FilterByCoin filters signals for a specific coin
func (s *SignalsService) FilterByCoin(signals []TradingSignal, coin string) []TradingSignal {
	coin = strings.ToUpper(coin)
	filtered := make([]TradingSignal, 0)
	for _, signal := range signals {
		if strings.ToUpper(signal.Coin) == coin {
			filtered = append(filtered, signal)
		}
	}
	return filtered
}

// FilterByDirection filters signals by direction (bullish/bearish)
func (s *SignalsService) FilterByDirection(signals []TradingSignal, direction string) []TradingSignal {
	direction = strings.ToLower(direction)
	filtered := make([]TradingSignal, 0)
	for _, signal := range signals {
		if strings.ToLower(signal.Direction) == direction {
			filtered = append(filtered, signal)
		}
	}
	return filtered
}

// parseSignalsResponse parses the JSON response from the LLM
func parseSignalsResponse(content string) (*SignalsResult, error) {
	// Clean up the response
	content = cleanJSONResponse(content)

	var result SignalsResult
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

	// Normalize and validate signals
	for i := range result.Signals {
		result.Signals[i].Direction = normalizeDirection(result.Signals[i].Direction)
		result.Signals[i].Strength = normalizeStrength(result.Signals[i].Strength)
		result.Signals[i].Coin = strings.ToUpper(result.Signals[i].Coin)
	}

	// Normalize market mood
	result.MarketMood = normalizeMarketMood(result.MarketMood)

	// Ensure signals is not nil
	if result.Signals == nil {
		result.Signals = []TradingSignal{}
	}

	return &result, nil
}

// normalizeDirection normalizes direction values
func normalizeDirection(direction string) string {
	direction = strings.ToLower(strings.TrimSpace(direction))
	switch direction {
	case "bullish", "long", "buy", "positive":
		return "bullish"
	case "bearish", "short", "sell", "negative":
		return "bearish"
	default:
		return direction
	}
}

// normalizeStrength normalizes strength values
func normalizeStrength(strength string) string {
	strength = strings.ToLower(strings.TrimSpace(strength))
	switch strength {
	case "strong", "high":
		return "strong"
	case "moderate", "medium", "mid":
		return "moderate"
	case "weak", "low":
		return "weak"
	default:
		return "moderate"
	}
}

// normalizeMarketMood normalizes market mood values
func normalizeMarketMood(mood string) string {
	mood = strings.ToLower(strings.TrimSpace(mood))
	switch mood {
	case "risk_on", "riskon", "bullish", "positive":
		return "risk_on"
	case "risk_off", "riskoff", "bearish", "negative":
		return "risk_off"
	default:
		return "neutral"
	}
}
