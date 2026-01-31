package middleware

import (
	"net/http"

	"cryptosignal-news/backend/internal/config"
)

// SecurityHeaders adds security headers to all responses (without CSP)
func SecurityHeaders(next http.Handler) http.Handler {
	return SecurityHeadersWithConfig(nil)(next)
}

// SecurityHeadersWithConfig adds security headers with configurable CSP
func SecurityHeadersWithConfig(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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

			// Content Security Policy (if configured)
			if cfg != nil && cfg.CSPPolicy != "" {
				w.Header().Set("Content-Security-Policy", cfg.CSPPolicy)
			}

			// HSTS - only enable in production with HTTPS
			if cfg != nil && cfg.HSTSEnabled {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}
