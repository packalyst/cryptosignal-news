package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/cryptosignal-news/backend/internal/api/response"
)

// Recoverer is a middleware that recovers from panics
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := GetRequestID(r.Context())

				// Log the panic with stack trace
				log.Printf("[%s] PANIC: %v\n%s", requestID, rec, debug.Stack())

				// Return 500 error
				response.InternalError(w, "An unexpected error occurred")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
