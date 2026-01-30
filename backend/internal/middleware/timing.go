package middleware

import (
	"context"
	"net/http"
	"time"
)

// timingKey is the context key for request start time
type timingKey struct{}

// Timing is a middleware that tracks request timing
func Timing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Add start time to context
		ctx := context.WithValue(r.Context(), timingKey{}, start)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestStartTime retrieves the request start time from the context
func GetRequestStartTime(ctx context.Context) time.Time {
	if start, ok := ctx.Value(timingKey{}).(time.Time); ok {
		return start
	}
	return time.Now()
}

// GetResponseTimeMs returns the elapsed time in milliseconds since request start
func GetResponseTimeMs(ctx context.Context) int64 {
	start := GetRequestStartTime(ctx)
	return time.Since(start).Milliseconds()
}
