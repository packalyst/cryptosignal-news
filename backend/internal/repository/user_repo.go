package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"cryptosignal-news/backend/internal/database"
	"cryptosignal-news/backend/internal/models"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUserExists is returned when trying to create a user that already exists
	ErrUserExists = errors.New("user already exists")
)

// UserRepository handles user database operations
type UserRepository struct {
	db *database.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if user.Tier == "" {
		user.Tier = models.TierFree
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
		INSERT INTO users (id, email, password_hash, tier, api_calls_today, api_calls_month, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Tier,
		user.APICallsToday, user.APICallsMonth, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		// Check for unique constraint violation
		if isUniqueViolation(err) {
			return ErrUserExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, tier, api_calls_today, api_calls_month, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var user models.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Tier,
		&user.APICallsToday, &user.APICallsMonth, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, tier, api_calls_today, api_calls_month, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var user models.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Tier,
		&user.APICallsToday, &user.APICallsMonth, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetByAPIKey retrieves a user by API key (the key should be hashed before calling this)
func (r *UserRepository) GetByAPIKey(ctx context.Context, keyHash string) (*models.User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.tier, u.api_calls_today, u.api_calls_month, u.created_at, u.updated_at
		FROM users u
		JOIN api_keys ak ON u.id = ak.user_id
		WHERE ak.key_hash = $1 AND ak.is_active = true
	`
	var user models.User
	err := r.db.QueryRow(ctx, query, keyHash).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Tier,
		&user.APICallsToday, &user.APICallsMonth, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by api key: %w", err)
	}

	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users
		SET email = $2, password_hash = $3, tier = $4, api_calls_today = $5, api_calls_month = $6, updated_at = $7
		WHERE id = $1
	`
	rowsAffected, err := r.db.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Tier,
		user.APICallsToday, user.APICallsMonth, user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// IncrementAPIUsage increments the API usage counters for a user
func (r *UserRepository) IncrementAPIUsage(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET api_calls_today = api_calls_today + 1,
		    api_calls_month = api_calls_month + 1,
		    updated_at = $2
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to increment api usage: %w", err)
	}

	return nil
}

// ResetDailyUsage resets the daily API usage counter for all users
// This should be called by a cron job at midnight UTC
func (r *UserRepository) ResetDailyUsage(ctx context.Context) error {
	query := `
		UPDATE users
		SET api_calls_today = 0,
		    updated_at = $1
	`
	_, err := r.db.Exec(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to reset daily usage: %w", err)
	}

	return nil
}

// ResetMonthlyUsage resets the monthly API usage counter for all users
// This should be called by a cron job at the start of each month
func (r *UserRepository) ResetMonthlyUsage(ctx context.Context) error {
	query := `
		UPDATE users
		SET api_calls_month = 0,
		    updated_at = $1
	`
	_, err := r.db.Exec(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to reset monthly usage: %w", err)
	}

	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	// First delete all API keys for this user
	_, err := r.db.Exec(ctx, "DELETE FROM api_keys WHERE user_id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete user api keys: %w", err)
	}

	// Then delete the user
	rowsAffected, err := r.db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateTier updates a user's subscription tier
func (r *UserRepository) UpdateTier(ctx context.Context, userID string, tier string) error {
	if !models.IsValidTier(tier) {
		return fmt.Errorf("invalid tier: %s", tier)
	}

	query := `UPDATE users SET tier = $2, updated_at = $3 WHERE id = $1`
	rowsAffected, err := r.db.Exec(ctx, query, userID, tier, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update tier: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// isUniqueViolation checks if an error is a unique constraint violation
func isUniqueViolation(err error) bool {
	// PostgreSQL unique violation error code is 23505
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "23505")
}
