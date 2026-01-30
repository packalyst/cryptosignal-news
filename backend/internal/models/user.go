package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID            string    `json:"id" db:"id"`
	Email         string    `json:"email" db:"email"`
	PasswordHash  string    `json:"-" db:"password_hash"`
	Tier          string    `json:"tier" db:"tier"`
	APICallsToday int       `json:"api_calls_today" db:"api_calls_today"`
	APICallsMonth int       `json:"api_calls_month" db:"api_calls_month"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// APIKey represents an API key for a user
type APIKey struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	KeyHash   string    `json:"-" db:"key_hash"`
	KeyPrefix string    `json:"key_prefix" db:"key_prefix"`
	Name      string    `json:"name" db:"name"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	LastUsed  time.Time `json:"last_used,omitempty" db:"last_used"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserTier constants
const (
	TierFree       = "free"
	TierPro        = "pro"
	TierEnterprise = "enterprise"
	TierAnonymous  = "anonymous"
)

// IsValidTier checks if a tier is valid
func IsValidTier(tier string) bool {
	switch tier {
	case TierFree, TierPro, TierEnterprise:
		return true
	default:
		return false
	}
}

// TierHierarchy returns the hierarchy level of a tier (higher = more privileges)
func TierHierarchy(tier string) int {
	switch tier {
	case TierEnterprise:
		return 3
	case TierPro:
		return 2
	case TierFree:
		return 1
	default:
		return 0
	}
}
