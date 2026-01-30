package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Logger logs HTTP requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Log request
		duration := time.Since(start)
		requestID := GetRequestID(r.Context())

		log.Printf("[%s] %s %s %d %d %s",
			requestID,
			r.Method,
			r.URL.Path,
			wrapped.status,
			wrapped.size,
			duration,
		)
	})
}
