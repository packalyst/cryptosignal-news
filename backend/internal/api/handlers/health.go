package handlers

import (
	"context"
	"net/http"
	"time"

	"cryptosignal-news/backend/internal/api/response"
	"cryptosignal-news/backend/internal/cache"
	"cryptosignal-news/backend/internal/database"
)

// HealthChecker provides health check functionality
type HealthChecker struct {
	db    *database.DB
	cache *cache.Redis
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *database.DB, cache *cache.Redis) *HealthChecker {
	return &HealthChecker{
		db:    db,
		cache: cache,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// Health handles GET /health
func (h *HealthChecker) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	services := make(map[string]string)
	overallStatus := "healthy"

	// Check database
	if err := h.db.Ping(ctx); err != nil {
		services["database"] = "unhealthy"
		overallStatus = "degraded"
	} else {
		services["database"] = "healthy"
	}

	// Check Redis
	if err := h.cache.Health(ctx); err != nil {
		services["redis"] = "unhealthy"
		overallStatus = "degraded"
	} else {
		services["redis"] = "healthy"
	}

	resp := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  services,
	}

	statusCode := http.StatusOK
	if overallStatus != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	response.JSON(w, statusCode, resp)
}

// LivenessProbe handles GET /health/live - simple liveness check
func LivenessProbe(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"status": "alive",
	})
}

// ReadinessProbe handles GET /health/ready - readiness check
func (h *HealthChecker) ReadinessProbe(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check database
	if err := h.db.Ping(ctx); err != nil {
		response.Error(w, http.StatusServiceUnavailable, "Database not ready")
		return
	}

	// Check Redis
	if err := h.cache.Health(ctx); err != nil {
		response.Error(w, http.StatusServiceUnavailable, "Redis not ready")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}
