package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"cryptosignal-news/backend/internal/auth"
	"cryptosignal-news/backend/internal/models"
	"cryptosignal-news/backend/internal/repository"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userRepo      *repository.UserRepository
	jwtService    *auth.JWTService
	apiKeyService *auth.APIKeyService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	userRepo *repository.UserRepository,
	jwtService *auth.JWTService,
	apiKeyService *auth.APIKeyService,
) *AuthHandler {
	return &AuthHandler{
		userRepo:      userRepo,
		jwtService:    jwtService,
		apiKeyService: apiKeyService,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Token     string        `json:"token"`
	ExpiresIn int64         `json:"expires_in"`
	User      *UserResponse `json:"user"`
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Tier      string    `json:"tier"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

// APIKeyResponse represents an API key in API responses
type APIKeyResponse struct {
	ID        string     `json:"id"`
	KeyPrefix string     `json:"key_prefix"`
	Name      string     `json:"name"`
	IsActive  bool       `json:"is_active"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// CreateAPIKeyResponse includes the full key (only shown once)
type CreateAPIKeyResponse struct {
	Key     string          `json:"key"` // Full key, only shown once
	KeyInfo *APIKeyResponse `json:"key_info"`
}

// Register handles user registration
// POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Validate email
	if !isValidEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid_email", "Invalid email address")
		return
	}

	// Validate password strength
	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, "weak_password", err.Error())
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to process registration")
		return
	}

	// Create user
	user := &models.User{
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: passwordHash,
		Tier:         models.TierFree,
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		if err == repository.ErrUserExists {
			writeError(w, http.StatusConflict, "user_exists", "An account with this email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to create account")
		return
	}

	// Generate JWT token
	token, err := h.jwtService.Generate(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to generate token")
		return
	}

	writeJSON(w, http.StatusCreated, AuthResponse{
		Token:     token,
		ExpiresIn: int64(h.jwtService.GetExpiration().Seconds()),
		User: &UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Tier:      user.Tier,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Login handles user login
// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Get user by email
	user, err := h.userRepo.GetByEmail(r.Context(), email)
	if err != nil {
		// Don't reveal whether the email exists
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Check password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Generate JWT token
	token, err := h.jwtService.Generate(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		Token:     token,
		ExpiresIn: int64(h.jwtService.GetExpiration().Seconds()),
		User: &UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Tier:      user.Tier,
			CreatedAt: user.CreatedAt,
		},
	})
}

// RefreshToken refreshes a JWT token
// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeError(w, http.StatusUnauthorized, "missing_token", "Authorization header required")
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid authorization header format")
		return
	}

	tokenString := parts[1]

	// Refresh the token
	newToken, err := h.jwtService.Refresh(tokenString)
	if err != nil {
		switch err {
		case auth.ErrExpiredToken:
			writeError(w, http.StatusUnauthorized, "token_expired", "Token has expired and cannot be refreshed")
		case auth.ErrInvalidToken:
			writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid token")
		default:
			writeError(w, http.StatusUnauthorized, "invalid_token", "Failed to refresh token")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":      newToken,
		"expires_in": int64(h.jwtService.GetExpiration().Seconds()),
	})
}

// GetCurrentUser returns the current authenticated user
// GET /api/v1/user/me
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	// Fetch full user data from database
	fullUser, err := h.userRepo.GetByID(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to fetch user data")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": &UserResponse{
			ID:        fullUser.ID,
			Email:     fullUser.Email,
			Tier:      fullUser.Tier,
			CreatedAt: fullUser.CreatedAt,
		},
	})
}

// CreateAPIKey creates a new API key for the user
// POST /api/v1/user/api-keys
func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body, use default name
		req.Name = "API Key"
	}

	if req.Name == "" {
		req.Name = "API Key"
	}

	// Generate the API key
	generated, err := h.apiKeyService.Generate(r.Context(), user.ID, req.Name)
	if err != nil {
		if errors.Is(err, auth.ErrAPIKeyLimitReached) {
			writeError(w, http.StatusBadRequest, "limit_reached", "Maximum API key limit reached")
			return
		}
		log.Printf("[auth] CreateAPIKey error: %v", err)
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to create API key")
		return
	}

	var lastUsed *time.Time
	if !generated.KeyInfo.LastUsed.IsZero() {
		lastUsed = &generated.KeyInfo.LastUsed
	}

	writeJSON(w, http.StatusCreated, CreateAPIKeyResponse{
		Key: generated.PlainTextKey,
		KeyInfo: &APIKeyResponse{
			ID:        generated.KeyInfo.ID,
			KeyPrefix: generated.KeyInfo.KeyPrefix,
			Name:      generated.KeyInfo.Name,
			IsActive:  generated.KeyInfo.IsActive,
			LastUsed:  lastUsed,
			CreatedAt: generated.KeyInfo.CreatedAt,
		},
	})
}

// ListAPIKeys lists all API keys for the user
// GET /api/v1/user/api-keys
func (h *AuthHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	keys, err := h.apiKeyService.List(r.Context(), user.ID)
	if err != nil {
		log.Printf("[auth] ListAPIKeys error: %v", err)
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to list API keys")
		return
	}

	response := make([]APIKeyResponse, len(keys))
	for i, key := range keys {
		var lastUsed *time.Time
		if !key.LastUsed.IsZero() {
			lastUsed = &key.LastUsed
		}
		response[i] = APIKeyResponse{
			ID:        key.ID,
			KeyPrefix: key.KeyPrefix,
			Name:      key.Name,
			IsActive:  key.IsActive,
			LastUsed:  lastUsed,
			CreatedAt: key.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"api_keys": response,
	})
}

// RevokeAPIKey revokes an API key
// DELETE /api/v1/user/api-keys/{keyID}
func (h *AuthHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	keyID := chi.URLParam(r, "keyID")
	if keyID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Key ID is required")
		return
	}

	err := h.apiKeyService.Revoke(r.Context(), keyID, user.ID)
	if err != nil {
		if err == auth.ErrAPIKeyNotFound {
			writeError(w, http.StatusNotFound, "not_found", "API key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", "Failed to revoke API key")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "API key revoked successfully",
	})
}

// isValidEmail validates an email address format
func isValidEmail(email string) bool {
	// Simple email regex - not perfect but good enough for basic validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error":   code,
		"message": message,
	})
}
