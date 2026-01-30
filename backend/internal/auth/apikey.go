package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cryptosignal-news/backend/internal/database"
	"github.com/cryptosignal-news/backend/internal/models"
)

const (
	// APIKeyPrefix is the prefix for all API keys
	APIKeyPrefix = "csn_live_"
	// APIKeyLength is the length of the random part of the API key
	APIKeyLength = 32
)

var (
	// ErrAPIKeyNotFound is returned when an API key is not found
	ErrAPIKeyNotFound = errors.New("api key not found")
	// ErrAPIKeyRevoked is returned when an API key has been revoked
	ErrAPIKeyRevoked = errors.New("api key has been revoked")
	// ErrAPIKeyInvalid is returned when an API key format is invalid
	ErrAPIKeyInvalid = errors.New("invalid api key format")
)

// APIKeyService handles API key operations
type APIKeyService struct {
	db *database.DB
}

// NewAPIKeyService creates a new API key service
func NewAPIKeyService(db *database.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

// GeneratedKey contains both the plain text key (shown once) and the stored key info
type GeneratedKey struct {
	PlainTextKey string         `json:"key"`       // Only shown once at creation
	KeyInfo      *models.APIKey `json:"key_info"` // Stored information
}

// Generate creates a new API key for a user
func (s *APIKeyService) Generate(ctx context.Context, userID string, name string) (*GeneratedKey, error) {
	// Generate a secure random API key
	plainKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate api key: %w", err)
	}

	// Hash the key for storage
	keyHash := hashAPIKey(plainKey)

	// Extract prefix for identification (first 7 chars after csn_live_)
	keyPrefix := plainKey[:len(APIKeyPrefix)+7] // csn_live_ (9) + 7 = 16 chars

	// Create the API key record
	apiKey := &models.APIKey{
		ID:        uuid.New().String(),
		UserID:    userID,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Name:      name,
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	// Store in database
	query := `
		INSERT INTO api_keys (id, user_id, key_hash, key_prefix, name, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = s.db.Exec(ctx, query,
		apiKey.ID, apiKey.UserID, apiKey.KeyHash, apiKey.KeyPrefix, apiKey.Name, apiKey.IsActive, apiKey.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to store api key: %w", err)
	}

	return &GeneratedKey{
		PlainTextKey: plainKey,
		KeyInfo:      apiKey,
	}, nil
}

// Validate validates an API key and returns the associated user
func (s *APIKeyService) Validate(ctx context.Context, key string) (*models.User, error) {
	// Validate key format
	if len(key) < len(APIKeyPrefix) || key[:len(APIKeyPrefix)] != APIKeyPrefix {
		return nil, ErrAPIKeyInvalid
	}

	// Hash the provided key
	keyHash := hashAPIKey(key)

	// Look up the key and associated user
	query := `
		SELECT u.id, u.email, u.password_hash, u.tier, u.api_calls_today, u.api_calls_month, u.created_at, u.updated_at
		FROM api_keys ak
		JOIN users u ON ak.user_id = u.id
		WHERE ak.key_hash = $1
	`
	var user models.User
	err := s.db.QueryRow(ctx, query, keyHash).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Tier,
		&user.APICallsToday, &user.APICallsMonth, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}

	// Check if key is active
	var isActive bool
	checkQuery := `SELECT is_active FROM api_keys WHERE key_hash = $1`
	err = s.db.QueryRow(ctx, checkQuery, keyHash).Scan(&isActive)
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}
	if !isActive {
		return nil, ErrAPIKeyRevoked
	}

	// Update last used timestamp
	updateQuery := `UPDATE api_keys SET last_used_at = $1 WHERE key_hash = $2`
	_, _ = s.db.Exec(ctx, updateQuery, time.Now(), keyHash)

	return &user, nil
}

// Revoke revokes an API key
func (s *APIKeyService) Revoke(ctx context.Context, keyID string, userID string) error {
	query := `UPDATE api_keys SET is_active = false WHERE id = $1 AND user_id = $2`
	rowsAffected, err := s.db.Exec(ctx, query, keyID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// List returns all API keys for a user (without the actual key values)
func (s *APIKeyService) List(ctx context.Context, userID string) ([]models.APIKey, error) {
	query := `
		SELECT id, user_id, key_prefix, name, is_active, last_used_at, created_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var key models.APIKey
		var lastUsed *time.Time
		err := rows.Scan(&key.ID, &key.UserID, &key.KeyPrefix, &key.Name, &key.IsActive, &lastUsed, &key.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}
		if lastUsed != nil {
			key.LastUsed = *lastUsed
		}
		keys = append(keys, key)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating api keys: %w", err)
	}

	return keys, nil
}

// Delete permanently deletes an API key
func (s *APIKeyService) Delete(ctx context.Context, keyID string, userID string) error {
	query := `DELETE FROM api_keys WHERE id = $1 AND user_id = $2`
	rowsAffected, err := s.db.Exec(ctx, query, keyID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// generateAPIKey generates a secure random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, APIKeyLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return APIKeyPrefix + hex.EncodeToString(bytes), nil
}

// hashAPIKey creates a SHA-256 hash of an API key
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
