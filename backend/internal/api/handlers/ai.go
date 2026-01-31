package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"cryptosignal-news/backend/internal/ai"
	"cryptosignal-news/backend/internal/api/response"
	"cryptosignal-news/backend/internal/models"
	"cryptosignal-news/backend/internal/service"
)

// AIHandler handles AI-related API endpoints
type AIHandler struct {
	sentimentService *ai.SentimentService
	summaryService   *ai.SummaryService
	signalsService   *ai.SignalsService
	newsService      *service.NewsService
}

// NewAIHandler creates a new AI handler
func NewAIHandler(
	sentimentService *ai.SentimentService,
	summaryService *ai.SummaryService,
	signalsService *ai.SignalsService,
	newsService *service.NewsService,
) *AIHandler {
	return &AIHandler{
		sentimentService: sentimentService,
		summaryService:   summaryService,
		signalsService:   signalsService,
		newsService:      newsService,
	}
}

// convertToAIArticles converts models.ArticleResponse to ai.Article
func convertToAIArticles(articles []models.ArticleResponse) []ai.Article {
	result := make([]ai.Article, len(articles))
	for i, a := range articles {
		// Parse the pub_date string back to time.Time
		pubDate, err := time.Parse(time.RFC3339, a.PubDate)
		if err != nil {
			pubDate = time.Now() // Fallback to now if parsing fails
		}
		result[i] = ai.Article{
			ID:          a.ID,
			Title:       a.Title,
			Description: a.Description,
			Link:        a.Link,
			Source:      a.Source,
			PubDate:     pubDate,
		}
	}
	return result
}

// GetSentiment handles GET /api/v1/ai/sentiment?coin=BTC
// Returns sentiment analysis for a specific coin
func (h *AIHandler) GetSentiment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get coin symbol from query params
	coin := strings.ToUpper(r.URL.Query().Get("coin"))
	if coin == "" {
		response.BadRequest(w, "coin parameter is required")
		return
	}

	// Validate coin symbol length (typical symbols are 2-10 chars)
	if len(coin) > 10 {
		response.BadRequest(w, "invalid coin symbol")
		return
	}

	// Get recent articles mentioning this coin
	articles, err := h.newsService.GetByCoin(ctx, coin, 50)
	if err != nil {
		response.InternalError(w, "failed to fetch articles")
		return
	}

	// Convert to AI articles
	aiArticles := convertToAIArticles(articles)

	// Get coin sentiment
	sentiment, err := h.sentimentService.GetCoinSentiment(ctx, coin, aiArticles)
	if err != nil {
		response.InternalError(w, "failed to analyze sentiment")
		return
	}

	response.Success(w, sentiment)
}

// SummaryResponse wraps market summary with the articles used
type SummaryResponse struct {
	*ai.MarketSummary
	Articles []models.ArticleResponse `json:"articles"`
}

// GetSummary handles GET /api/v1/ai/summary
// Returns daily market summary with the 20 articles used
func (h *AIHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get latest 20 articles for summary
	opts := service.ListOptions{
		Limit:  20,
		Offset: 0,
	}
	result, err := h.newsService.GetLatest(ctx, opts)
	if err != nil {
		response.InternalError(w, "failed to fetch articles")
		return
	}

	// Try to get cached summary first
	summary, err := h.summaryService.GetCachedSummary(ctx)
	if err != nil || summary == nil {
		// Convert to AI articles and generate summary
		aiArticles := convertToAIArticles(result.Articles)
		summary, err = h.summaryService.GenerateDailySummary(ctx, aiArticles)
		if err != nil {
			response.InternalError(w, "failed to generate summary")
			return
		}
	}

	// Return summary with articles
	response.Success(w, SummaryResponse{
		MarketSummary: summary,
		Articles:      result.Articles,
	})
}

// GetSignals handles GET /api/v1/ai/signals
// Returns trading signals from news (cached 30 min)
func (h *AIHandler) GetSignals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Optional filters
	coin := strings.ToUpper(r.URL.Query().Get("coin"))
	direction := strings.ToLower(r.URL.Query().Get("direction"))
	minStrength := strings.ToLower(r.URL.Query().Get("min_strength"))

	// Validate optional coin parameter
	if coin != "" && len(coin) > 10 {
		response.BadRequest(w, "invalid coin symbol")
		return
	}

	// Try to get cached signals first
	signals, err := h.signalsService.GetCachedSignals(ctx)
	if err != nil || signals == nil {
		// Get recent articles for signal generation
		opts := service.ListOptions{
			Limit:  50,
			Offset: 0,
		}
		result, err := h.newsService.GetLatest(ctx, opts)
		if err != nil {
			response.InternalError(w, "failed to fetch articles")
			return
		}

		// Filter to last 6 hours for more relevant signals
		cutoff := time.Now().Add(-6 * time.Hour)
		var recentArticles []models.ArticleResponse
		for _, article := range result.Articles {
			pubDate, err := time.Parse(time.RFC3339, article.PubDate)
			if err == nil && pubDate.After(cutoff) {
				recentArticles = append(recentArticles, article)
			}
		}

		// Convert to AI articles
		aiArticles := convertToAIArticles(recentArticles)

		// Generate signals
		signals, err = h.signalsService.GenerateSignals(ctx, aiArticles)
		if err != nil {
			response.InternalError(w, "failed to generate signals")
			return
		}
	}

	// Apply filters
	filteredSignals := signals.Signals
	if coin != "" {
		filteredSignals = h.signalsService.FilterByCoin(filteredSignals, coin)
	}
	if direction != "" {
		filteredSignals = h.signalsService.FilterByDirection(filteredSignals, direction)
	}
	if minStrength != "" {
		filteredSignals = h.signalsService.FilterByStrength(filteredSignals, minStrength)
	}

	// Create response with filtered signals
	signalsResponse := &ai.SignalsResult{
		Signals:      filteredSignals,
		MarketMood:   signals.MarketMood,
		GeneratedAt:  signals.GeneratedAt,
		ArticleCount: signals.ArticleCount,
	}

	response.Success(w, signalsResponse)
}

// AnalyzeTextRequest is the request body for custom text analysis
type AnalyzeTextRequest struct {
	Text string `json:"text"`
}

// AnalyzeTextResponse is the response for custom text analysis
type AnalyzeTextResponse struct {
	Sentiment      string   `json:"sentiment"`
	Score          float64  `json:"score"`
	Confidence     float64  `json:"confidence"`
	CoinsMentioned []string `json:"coins_mentioned"`
	Reasoning      string   `json:"reasoning"`
	Actionable     bool     `json:"actionable"`
}

// AnalyzeText handles POST /api/v1/ai/analyze
// Analyze custom text (premium only)
func (h *AIHandler) AnalyzeText(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req AnalyzeTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Text == "" {
		response.BadRequest(w, "text field is required")
		return
	}

	// Limit text length
	if len(req.Text) > 10000 {
		response.BadRequest(w, "text exceeds maximum length of 10000 characters")
		return
	}

	// Create a temporary article for analysis
	article := &ai.Article{
		ID:          0, // Temporary ID, won't be cached
		Title:       "Custom Analysis",
		Description: req.Text,
		PubDate:     time.Now(),
	}

	// Analyze the text
	result, err := h.sentimentService.AnalyzeArticle(ctx, article)
	if err != nil {
		response.InternalError(w, "failed to analyze text")
		return
	}

	// Build response
	analyzeResponse := AnalyzeTextResponse{
		Sentiment:      result.Sentiment,
		Score:          result.Score,
		Confidence:     result.Confidence,
		CoinsMentioned: result.CoinsMentioned,
		Reasoning:      result.Reasoning,
		Actionable:     result.Confidence > 0.7 && result.Sentiment != "neutral",
	}

	response.Success(w, analyzeResponse)
}
