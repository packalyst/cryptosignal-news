package handlers

import (
	"net/http"

	"cryptosignal-news/backend/internal/api/response"
	"cryptosignal-news/backend/internal/cache"
	"cryptosignal-news/backend/internal/middleware"
	"cryptosignal-news/backend/internal/service"
)

// SourceHandler handles source-related HTTP requests
type SourceHandler struct {
	sourceService *service.SourceService
}

// NewSourceHandler creates a new source handler
func NewSourceHandler(sourceService *service.SourceService) *SourceHandler {
	return &SourceHandler{
		sourceService: sourceService,
	}
}

// ListSources handles GET /api/v1/sources
// List all sources with status
func (h *SourceHandler) ListSources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sources, err := h.sourceService.ListSources(ctx)
	if err != nil {
		response.InternalError(w, "Failed to fetch sources")
		return
	}

	// Generate ETag
	etag := cache.GetETag(sources)
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
		Data: sources,
		Meta: meta,
	})
}

// ListCategories handles GET /api/v1/categories
// List categories with article counts
func (h *SourceHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	categories, err := h.sourceService.GetCategories(ctx)
	if err != nil {
		response.InternalError(w, "Failed to fetch categories")
		return
	}

	// Generate ETag
	etag := cache.GetETag(categories)
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
		Data: categories,
		Meta: meta,
	})
}
