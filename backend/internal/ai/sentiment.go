package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SentimentResult represents the result of sentiment analysis
type SentimentResult struct {
	Sentiment      string   `json:"sentiment"`
	Score          float64  `json:"score"`
	Confidence     float64  `json:"confidence"`
	Reasoning      string   `json:"reasoning"`
	CoinsMentioned []string `json:"coins_mentioned"`
}

// CoinSentiment represents aggregated sentiment for a specific coin
type CoinSentiment struct {
	Symbol       string  `json:"symbol"`
	Sentiment    string  `json:"sentiment"`
	Score        float64 `json:"score"`
	ArticleCount int     `json:"article_count"`
	BullishCount int     `json:"bullish_count"`
	BearishCount int     `json:"bearish_count"`
	NeutralCount int     `json:"neutral_count"`
	UpdatedAt    string  `json:"updated_at"`
}

// Article represents a news article for sentiment analysis
type Article struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	Source      string    `json:"source"`
	PubDate     time.Time `json:"pub_date"`
	Sentiment   string    `json:"sentiment,omitempty"`
	Score       float64   `json:"score,omitempty"`
}

// SentimentService handles sentiment analysis operations
type SentimentService struct {
	groq  *GroqClient
	cache *AICache
}

// NewSentimentService creates a new sentiment service
func NewSentimentService(groq *GroqClient, cache *AICache) *SentimentService {
	return &SentimentService{
		groq:  groq,
		cache: cache,
	}
}

// AnalyzeArticle analyzes the sentiment of a single article
func (s *SentimentService) AnalyzeArticle(ctx context.Context, article *Article) (*SentimentResult, error) {
	// Check cache first (only for articles with valid IDs)
	if s.cache != nil && article.ID > 0 {
		cached, err := s.cache.GetSentiment(ctx, article.ID)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Render the prompt
	prompt, err := RenderSentimentPrompt(article.Title, article.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to render sentiment prompt: %w", err)
	}

	// Make API request
	req := &ChatRequest{
		Model:       DefaultGroqModel,
		Temperature: 0.3, // Lower temperature for more consistent results
		MaxTokens:   512,
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "You are a financial sentiment analyzer. Analyze crypto news and respond ONLY with valid JSON. No markdown, no explanations.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	resp, err := s.groq.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze sentiment: %w", err)
	}

	// Parse the response
	content := resp.GetMessageContent()
	result, err := parseSentimentResponse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sentiment response: %w", err)
	}

	// Cache the result (only for articles with valid IDs)
	if s.cache != nil && article.ID > 0 {
		if cacheErr := s.cache.SetSentiment(ctx, article.ID, result); cacheErr != nil {
			log.Printf("warning: failed to cache sentiment: %v", cacheErr)
		}
	}

	return result, nil
}

// AnalyzeBatch analyzes sentiment for multiple articles concurrently
func (s *SentimentService) AnalyzeBatch(ctx context.Context, articles []Article) ([]SentimentResult, error) {
	results := make([]SentimentResult, len(articles))
	var wg sync.WaitGroup
	errChan := make(chan error, len(articles))

	// Limit concurrent requests
	semaphore := make(chan struct{}, 5)

	for i := range articles {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := s.AnalyzeArticle(ctx, &articles[idx])
			if err != nil {
				errChan <- fmt.Errorf("article %d: %w", articles[idx].ID, err)
				// Set default result on error
				results[idx] = SentimentResult{
					Sentiment:  "neutral",
					Score:      0,
					Confidence: 0,
					Reasoning:  "Analysis failed",
				}
				return
			}
			results[idx] = *result
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Collect errors (non-fatal)
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		log.Printf("warning: %d articles failed sentiment analysis", len(errs))
	}

	return results, nil
}

// GetCoinSentiment calculates aggregated sentiment for a specific coin
// Uses a single API call with aggregated headlines instead of per-article analysis
func (s *SentimentService) GetCoinSentiment(ctx context.Context, symbol string, articles []Article) (*CoinSentiment, error) {
	symbol = strings.ToUpper(symbol)

	// Check cache first
	if s.cache != nil {
		cached, err := s.cache.GetCoinSentiment(ctx, symbol)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Filter articles mentioning this coin
	var relevantArticles []Article
	for _, article := range articles {
		if containsCoin(article.Title+" "+article.Description, symbol) {
			relevantArticles = append(relevantArticles, article)
		}
	}

	if len(relevantArticles) == 0 {
		return &CoinSentiment{
			Symbol:       symbol,
			Sentiment:    "neutral",
			Score:        0,
			ArticleCount: 0,
			UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
		}, nil
	}

	// Build aggregated prompt with all headlines (limit to 30)
	maxArticles := 30
	if len(relevantArticles) > maxArticles {
		relevantArticles = relevantArticles[:maxArticles]
	}

	var headlines strings.Builder
	for i, article := range relevantArticles {
		headlines.WriteString(fmt.Sprintf("%d. %s\n", i+1, article.Title))
	}

	// Single API call for aggregated sentiment
	prompt := fmt.Sprintf(`Analyze the overall sentiment for %s based on these recent news headlines:

%s

Respond with JSON only:
{"sentiment": "bullish|bearish|neutral", "score": 0.0 to 1.0, "reasoning": "brief explanation"}`, symbol, headlines.String())

	chatReq := &ChatRequest{
		Model: "llama-3.3-70b-versatile",
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   200,
	}

	resp, err := s.groq.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze sentiment: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	// Parse response
	content := cleanJSONResponse(resp.Choices[0].Message.Content)
	var result struct {
		Sentiment string  `json:"sentiment"`
		Score     float64 `json:"score"`
		Reasoning string  `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		jsonStr := extractJSON(content)
		if jsonStr != "" {
			json.Unmarshal([]byte(jsonStr), &result)
		}
	}

	coinSentiment := &CoinSentiment{
		Symbol:       symbol,
		Sentiment:    result.Sentiment,
		Score:        result.Score,
		ArticleCount: len(relevantArticles),
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	// Cache the result (15 min TTL)
	if s.cache != nil {
		if cacheErr := s.cache.SetCoinSentiment(ctx, symbol, coinSentiment); cacheErr != nil {
			log.Printf("warning: failed to cache coin sentiment: %v", cacheErr)
		}
	}

	return coinSentiment, nil
}

// parseSentimentResponse parses the JSON response from the LLM
func parseSentimentResponse(content string) (*SentimentResult, error) {
	// Clean up the response - remove markdown code blocks if present
	content = cleanJSONResponse(content)

	var result SentimentResult
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

	// Validate and normalize the result
	result.Sentiment = normalizeSentiment(result.Sentiment)
	result.Score = clampFloat(result.Score, -1.0, 1.0)
	result.Confidence = clampFloat(result.Confidence, 0.0, 1.0)

	// Ensure coins mentioned is not nil
	if result.CoinsMentioned == nil {
		result.CoinsMentioned = []string{}
	}

	return &result, nil
}

// cleanJSONResponse removes markdown code blocks and whitespace
func cleanJSONResponse(content string) string {
	// Remove markdown code blocks
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	return content
}

// extractJSON attempts to extract JSON from a string
func extractJSON(s string) string {
	// Find the first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || start >= end {
		return ""
	}
	return s[start : end+1]
}

// normalizeSentiment normalizes sentiment values
func normalizeSentiment(sentiment string) string {
	sentiment = strings.ToLower(strings.TrimSpace(sentiment))
	switch sentiment {
	case "bullish", "positive":
		return "bullish"
	case "bearish", "negative":
		return "bearish"
	default:
		return "neutral"
	}
}

// clampFloat clamps a float value between min and max
func clampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// containsCoin checks if text mentions a specific coin
func containsCoin(text, symbol string) bool {
	text = strings.ToUpper(text)
	symbol = strings.ToUpper(symbol)

	// Check for exact symbol match
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(symbol) + `\b`)
	if pattern.MatchString(text) {
		return true
	}

	// Check for common coin names
	coinNames := map[string][]string{
		"BTC":   {"BITCOIN"},
		"ETH":   {"ETHEREUM", "ETHER"},
		"XRP":   {"RIPPLE"},
		"SOL":   {"SOLANA"},
		"ADA":   {"CARDANO"},
		"DOGE":  {"DOGECOIN"},
		"DOT":   {"POLKADOT"},
		"LINK":  {"CHAINLINK"},
		"AVAX":  {"AVALANCHE"},
		"MATIC": {"POLYGON"},
	}

	if names, ok := coinNames[symbol]; ok {
		for _, name := range names {
			if strings.Contains(text, name) {
				return true
			}
		}
	}

	return false
}

// aggregateSentiments calculates aggregated sentiment from multiple results
func aggregateSentiments(symbol string, results []SentimentResult) *CoinSentiment {
	var totalScore float64
	var bullish, bearish, neutral int

	for _, result := range results {
		totalScore += result.Score
		switch result.Sentiment {
		case "bullish":
			bullish++
		case "bearish":
			bearish++
		default:
			neutral++
		}
	}

	count := len(results)
	avgScore := 0.0
	if count > 0 {
		avgScore = totalScore / float64(count)
	}

	// Determine overall sentiment
	var overallSentiment string
	if bullish > bearish && bullish > neutral {
		overallSentiment = "bullish"
	} else if bearish > bullish && bearish > neutral {
		overallSentiment = "bearish"
	} else {
		overallSentiment = "neutral"
	}

	return &CoinSentiment{
		Symbol:       symbol,
		Sentiment:    overallSentiment,
		Score:        avgScore,
		ArticleCount: count,
		BullishCount: bullish,
		BearishCount: bearish,
		NeutralCount: neutral,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
}
