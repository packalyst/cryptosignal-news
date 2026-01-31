// Package config provides application configuration management.
// It loads configuration from environment variables with sensible defaults.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	// Server settings
	Port string
	Env  string

	// Database settings
	DatabaseURL string

	// Redis settings
	RedisURL string

	// Authentication
	JWTSecret string

	// External APIs
	GroqAPIKey string

	// CORS
	CORSOrigins []string

	// Rate limiting (per minute)
	RateLimitEnabled    bool
	RateLimitAnonymous  int
	RateLimitFree       int
	RateLimitPro        int
	RateLimitEnterprise int

	// Proxy settings
	TrustProxy bool // Trust X-Forwarded-For header (only enable behind reverse proxy)

	// Security
	CSPPolicy              string        // Content-Security-Policy header value (empty = disabled)
	JWTRefreshGracePeriod  time.Duration // How long after expiry a token can still be refreshed
	MaxAPIKeysPerUser      int           // Maximum API keys a user can create
	HSTSEnabled            bool          // Enable Strict-Transport-Security header (only for HTTPS)

	// Cache TTL (seconds)
	CacheTTL int

	// Feature flags
	EnableMetrics bool

	// Fetcher settings
	FetcherWorkers  int
	FetcherTimeout  time.Duration
	FetcherInterval time.Duration
	FetcherMaxAge   time.Duration

	// Translation settings
	TranslationEnabled        bool
	TranslationTargetLanguage string // Target language code (e.g., "en", "ro")
	TranslationInterval       time.Duration
	TranslationBatchSize      int

	// AI Model settings
	ModelTranslation string // Model for translation (default: llama-3.1-8b-instant)
	ModelSentiment   string // Model for sentiment analysis (default: llama-3.3-70b-versatile)
	ModelSummary     string // Model for summaries (default: llama-3.3-70b-versatile)
}

// Load returns a new Config struct populated from environment variables
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/cryptonews?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:          getJWTSecret(),
		GroqAPIKey:         getEnv("GROQ_API_KEY", ""),
		CORSOrigins:         getEnvSlice("CORS_ORIGINS", []string{"*"}),
		RateLimitEnabled:    getEnvBool("RATE_LIMIT_ENABLED", true),
		RateLimitAnonymous:  getEnvInt("RATE_LIMIT_ANONYMOUS", 10),
		RateLimitFree:       getEnvInt("RATE_LIMIT_FREE", 60),
		RateLimitPro:        getEnvInt("RATE_LIMIT_PRO", 300),
		RateLimitEnterprise: getEnvInt("RATE_LIMIT_ENTERPRISE", 1000),
		TrustProxy:          getEnvBool("TRUST_PROXY", false),
		CSPPolicy:             getEnv("CSP_POLICY", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'"),
		JWTRefreshGracePeriod: getEnvDuration("JWT_REFRESH_GRACE_PERIOD", 24*time.Hour),
		MaxAPIKeysPerUser:     getEnvInt("MAX_API_KEYS_PER_USER", 10),
		HSTSEnabled:           getEnvBool("HSTS_ENABLED", false),
		CacheTTL:              getEnvInt("CACHE_TTL", 60),
		EnableMetrics:      getEnvBool("ENABLE_METRICS", false),
		FetcherWorkers:     getEnvInt("FETCHER_WORKERS", 50),
		FetcherTimeout:     getEnvDuration("FETCHER_TIMEOUT", 10*time.Second),
		FetcherInterval:    getEnvDuration("FETCH_INTERVAL", 3*time.Minute),
		FetcherMaxAge:      getEnvDuration("FETCHER_MAX_AGE", 7*24*time.Hour),

		TranslationEnabled:        getEnv("GROQ_API_KEY", "") != "",
		TranslationTargetLanguage: getEnv("TRANSLATION_TARGET_LANGUAGE", "en"),
		TranslationInterval:       getEnvDuration("TRANSLATION_INTERVAL", 30*time.Second),
		TranslationBatchSize:      getEnvInt("TRANSLATION_BATCH_SIZE", 5),

		ModelTranslation: getEnv("MODEL_TRANSLATION", "llama-3.1-8b-instant"),
		ModelSentiment:   getEnv("MODEL_SENTIMENT", "llama-3.3-70b-versatile"),
		ModelSummary:     getEnv("MODEL_SUMMARY", "llama-3.3-70b-versatile"),
	}
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

const jwtSecretFileName = ".jwt_secret"

// getJWTSecret retrieves the JWT secret with the following priority:
// 1. JWT_SECRET environment variable
// 2. .jwt_secret file in secrets directory (SECRETS_DIR env or /app/secrets)
// 3. Auto-generate and save to secrets directory
func getJWTSecret() string {
	// Check environment variable first
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}

	// Try to read from file
	secretPath := getSecretFilePath()
	if data, err := os.ReadFile(secretPath); err == nil {
		secret := strings.TrimSpace(string(data))
		if secret != "" {
			fmt.Printf("[config] JWT secret loaded from %s\n", secretPath)
			return secret
		}
	}

	// Generate new secret
	secret, err := generateSecureSecret(32)
	if err != nil {
		log.Fatal("[config] CRITICAL: Failed to generate JWT secret, cannot start securely")
	}

	// Ensure secrets directory exists
	secretsDir := getSecretsDir()
	if err := os.MkdirAll(secretsDir, 0700); err != nil {
		fmt.Printf("[config] WARNING: Failed to create secrets directory: %v\n", err)
	}

	// Save to file
	if err := os.WriteFile(secretPath, []byte(secret), 0600); err != nil {
		fmt.Printf("[config] WARNING: Failed to save JWT secret to file: %v\n", err)
	} else {
		fmt.Printf("[config] Generated new JWT secret and saved to %s\n", secretPath)
	}

	return secret
}

// getSecretsDir returns the directory for storing secrets
func getSecretsDir() string {
	// Check environment variable
	if dir := os.Getenv("SECRETS_DIR"); dir != "" {
		return dir
	}
	// Default: /app/secrets in Docker, or working directory locally
	if _, err := os.Stat("/app"); err == nil {
		return "/app/secrets"
	}
	return "."
}

// getSecretFilePath returns the path to the JWT secret file
func getSecretFilePath() string {
	return filepath.Join(getSecretsDir(), jwtSecretFileName)
}

// generateSecureSecret generates a cryptographically secure random hex string
func generateSecureSecret(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvBool retrieves a boolean environment variable or returns a default value.
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// getEnvDuration retrieves a duration environment variable or returns a default value.
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// getEnvSlice retrieves a comma-separated environment variable as a slice.
func getEnvSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return defaultValue
	}
	return result
}
