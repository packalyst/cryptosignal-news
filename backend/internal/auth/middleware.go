package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"cryptosignal-news/backend/internal/models"
)

// Context keys for authentication
type contextKey string

const (
	// UserContextKey is the context key for the authenticated user
	UserContextKey contextKey = "user"
	// ClaimsContextKey is the context key for JWT claims
	ClaimsContextKey contextKey = "claims"
)

// AuthMiddleware holds dependencies for authentication middleware
type AuthMiddleware struct {
	jwtService    *JWTService
	apiKeyService *APIKeyService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtService *JWTService, apiKeyService *APIKeyService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService:    jwtService,
		apiKeyService: apiKeyService,
	}
}

// Authenticate middleware authenticates requests via JWT token or API key
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, claims, err := m.authenticate(r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		// Add user and claims to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		if claims != nil {
			ctx = context.WithValue(ctx, ClaimsContextKey, claims)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth middleware sets user if authenticated but continues if not
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, claims, err := m.authenticate(r)
		if err == nil && user != nil {
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			if claims != nil {
				ctx = context.WithValue(ctx, ClaimsContextKey, claims)
			}
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// RequireTier returns middleware that requires a minimum tier level
func (m *AuthMiddleware) RequireTier(requiredTier string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				writeAuthError(w, ErrInvalidToken)
				return
			}

			// Check tier hierarchy
			requiredLevel := models.TierHierarchy(requiredTier)
			userLevel := models.TierHierarchy(user.Tier)

			if userLevel < requiredLevel {
				writeJSON(w, http.StatusForbidden, map[string]interface{}{
					"error":         "insufficient_tier",
					"message":       "Your subscription tier does not allow access to this resource",
					"required_tier": requiredTier,
					"current_tier":  user.Tier,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// authenticate attempts to authenticate a request
func (m *AuthMiddleware) authenticate(r *http.Request) (*models.User, *Claims, error) {
	// Try API key first (X-API-Key header)
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != "" {
		user, err := m.apiKeyService.Validate(r.Context(), apiKey)
		if err != nil {
			return nil, nil, err
		}
		return user, nil, nil
	}

	// Try JWT token (Authorization header)
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, nil, ErrInvalidToken
	}

	// Extract bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, nil, ErrInvalidToken
	}

	tokenString := parts[1]
	claims, err := m.jwtService.Validate(tokenString)
	if err != nil {
		return nil, nil, err
	}

	// Create user from claims
	user := &models.User{
		ID:    claims.UserID,
		Email: claims.Email,
		Tier:  claims.Tier,
	}

	return user, claims, nil
}

// GetUser returns the authenticated user from context
func GetUser(ctx context.Context) *models.User {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// GetUserID returns the authenticated user ID from context
func GetUserID(ctx context.Context) string {
	user := GetUser(ctx)
	if user == nil {
		return ""
	}
	return user.ID
}

// GetClaims returns the JWT claims from context
func GetClaims(ctx context.Context) *Claims {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

// writeAuthError writes an authentication error response
func writeAuthError(w http.ResponseWriter, err error) {
	status := http.StatusUnauthorized
	message := "Authentication required"

	switch err {
	case ErrExpiredToken:
		message = "Token has expired"
	case ErrInvalidToken:
		message = "Invalid authentication token"
	case ErrTokenNotYetValid:
		message = "Token is not yet valid"
	case ErrAPIKeyNotFound:
		message = "Invalid API key"
	case ErrAPIKeyRevoked:
		message = "API key has been revoked"
	case ErrAPIKeyInvalid:
		message = "Invalid API key format"
	}

	writeJSON(w, status, map[string]interface{}{
		"error":   "unauthorized",
		"message": message,
	})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
