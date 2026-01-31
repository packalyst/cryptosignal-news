package handlers

import (
	"net/http"
	"strings"

	"cryptosignal-news/backend/internal/api/request"
	"cryptosignal-news/backend/internal/api/response"
	"cryptosignal-news/backend/internal/cache"
	"cryptosignal-news/backend/internal/middleware"
	"cryptosignal-news/backend/internal/service"
)

// NewsHandler handles news-related HTTP requests
type NewsHandler struct {
	newsService *service.NewsService
}

// NewNewsHandler creates a new news handler
func NewNewsHandler(newsService *service.NewsService) *NewsHandler {
	return &NewsHandler{
		newsService: newsService,
	}
}

// ListNews handles GET /api/v1/news
// Query params: limit (1-100, default 20), offset, source, category (comma-separated), language, from, to
func (h *NewsHandler) ListNews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limit := request.GetQueryIntWithRange(r, "limit", 20, 1, 100)
	offset := request.GetQueryInt(r, "offset", 0)
	source := request.GetQueryString(r, "source", "")
	categoryParam := request.GetQueryString(r, "category", "")
	language := request.GetQueryString(r, "language", "")
	from := request.GetQueryTime(r, "from")
	to := request.GetQueryTime(r, "to")

	// Parse comma-separated categories
	var categories []string
	if categoryParam != "" {
		for _, cat := range strings.Split(categoryParam, ",") {
			if trimmed := strings.TrimSpace(cat); trimmed != "" {
				categories = append(categories, trimmed)
			}
		}
	}

	opts := service.ListOptions{
		Limit:      limit,
		Offset:     offset,
		Source:     source,
		Categories: categories,
		Language:   language,
		From:       from,
		To:         to,
	}

	result, err := h.newsService.GetLatest(ctx, opts)
	if err != nil {
		response.InternalError(w, "Failed to fetch news")
		return
	}

	// Generate ETag for caching
	etag := cache.GetETag(result)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=60")

	// Check If-None-Match header for 304 response
	if match := r.Header.Get("If-None-Match"); match == etag {
		response.NotModified(w)
		return
	}

	pagination := response.NewPagination(result.Total, limit, offset)
	meta := response.NewMeta(
		middleware.GetRequestID(ctx),
		middleware.GetResponseTimeMs(ctx),
	)

	response.SuccessWithPagination(w, result.Articles, pagination, meta)
}

// BreakingNews handles GET /api/v1/news/breaking
// Returns articles from the last 2 hours
func (h *NewsHandler) BreakingNews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := request.GetQueryIntWithRange(r, "limit", 20, 1, 50)

	articles, err := h.newsService.GetBreaking(ctx, limit)
	if err != nil {
		response.InternalError(w, "Failed to fetch breaking news")
		return
	}

	// Generate ETag
	etag := cache.GetETag(articles)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=30")

	// Check If-None-Match
	if match := r.Header.Get("If-None-Match"); match == etag {
		response.NotModified(w)
		return
	}

	meta := response.NewMeta(
		middleware.GetRequestID(ctx),
		middleware.GetResponseTimeMs(ctx),
	)

	response.JSON(w, http.StatusOK, response.APIResponse{
		Data: articles,
		Meta: meta,
	})
}

// SearchNews handles GET /api/v1/news/search?q=keyword
// Full-text search with ranking
func (h *NewsHandler) SearchNews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := request.GetQueryString(r, "q", "")
	if strings.TrimSpace(query) == "" {
		response.BadRequest(w, "Search query is required")
		return
	}

	// Limit query length to prevent abuse
	const maxQueryLen = 200
	if len(query) > maxQueryLen {
		response.BadRequest(w, "Search query too long (max 200 characters)")
		return
	}

	limit := request.GetQueryIntWithRange(r, "limit", 20, 1, 100)

	articles, err := h.newsService.Search(ctx, query, limit)
	if err != nil {
		response.InternalError(w, "Failed to search news")
		return
	}

	// Generate ETag
	etag := cache.GetETag(articles)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=60")

	// Check If-None-Match
	if match := r.Header.Get("If-None-Match"); match == etag {
		response.NotModified(w)
		return
	}

	pagination := response.NewPagination(len(articles), limit, 0)
	meta := response.NewMeta(
		middleware.GetRequestID(ctx),
		middleware.GetResponseTimeMs(ctx),
	)

	response.SuccessWithQuery(w, articles, query, pagination, meta)
}

// GetArticle handles GET /api/v1/news/{id}
// Single article with full details
func (h *NewsHandler) GetArticle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := request.GetURLParamInt(r, "id")
	if err != nil {
		response.BadRequest(w, "Invalid article ID")
		return
	}

	article, err := h.newsService.GetByID(ctx, id)
	if err != nil {
		response.InternalError(w, "Failed to fetch article")
		return
	}

	if article == nil {
		response.NotFound(w, "Article not found")
		return
	}

	// Generate ETag
	etag := cache.GetETag(article)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=300")

	// Check If-None-Match
	if match := r.Header.Get("If-None-Match"); match == etag {
		response.NotModified(w)
		return
	}

	meta := response.NewMeta(
		middleware.GetRequestID(ctx),
		middleware.GetResponseTimeMs(ctx),
	)

	response.JSON(w, http.StatusOK, response.APIResponse{
		Data: article,
		Meta: meta,
	})
}

// NewsByCoin handles GET /api/v1/news/coin/{symbol}
// News mentioning specific coin (BTC, ETH, etc.)
func (h *NewsHandler) NewsByCoin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	symbol := request.GetURLParam(r, "symbol")
	if symbol == "" {
		response.BadRequest(w, "Coin symbol is required")
		return
	}

	// Validate symbol length (typical symbols are 2-10 chars)
	if len(symbol) > 10 {
		response.BadRequest(w, "Invalid coin symbol")
		return
	}

	// Normalize symbol to uppercase
	symbol = strings.ToUpper(symbol)

	limit := request.GetQueryIntWithRange(r, "limit", 20, 1, 100)

	articles, err := h.newsService.GetByCoin(ctx, symbol, limit)
	if err != nil {
		response.InternalError(w, "Failed to fetch news for coin")
		return
	}

	// Generate ETag
	etag := cache.GetETag(articles)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=60")

	// Check If-None-Match
	if match := r.Header.Get("If-None-Match"); match == etag {
		response.NotModified(w)
		return
	}

	pagination := response.NewPagination(len(articles), limit, 0)
	meta := response.NewMeta(
		middleware.GetRequestID(ctx),
		middleware.GetResponseTimeMs(ctx),
	)

	response.SuccessWithPagination(w, articles, pagination, meta)
}
