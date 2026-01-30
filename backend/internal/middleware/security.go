package middleware

import "net/http"

// SecurityHeaders adds security headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking - don't allow embedding in iframes
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// XSS protection for older browsers
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy - don't leak URLs to other sites
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions policy - disable unnecessary browser features
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}
