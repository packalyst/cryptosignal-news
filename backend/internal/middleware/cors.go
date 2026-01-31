package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// CORS returns a configured CORS middleware
func CORS() func(next http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allow all origins in development
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "If-None-Match"},
		ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "ETag"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any major browser
	})
}

// CORSWithOrigins returns a CORS middleware with specific allowed origins
func CORSWithOrigins(origins []string) func(next http.Handler) http.Handler {
	// Check if wildcard is used - if so, disable credentials (browser requirement)
	allowCredentials := true
	for _, o := range origins {
		if o == "*" {
			allowCredentials = false
			break
		}
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-API-Key", "If-None-Match"},
		ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining", "ETag"},
		AllowCredentials: allowCredentials,
		MaxAge:           300,
	})
}
