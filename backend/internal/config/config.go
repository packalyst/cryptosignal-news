// Package config provides application configuration management.
// It loads configuration from environment variables with sensible defaults.
package config

import (
	"os"
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

	// Rate limiting
	RateLimitPerMinute int

	// Cache TTL (seconds)
	CacheTTL int

	// Feature flags
	EnableMetrics bool

	// Fetcher settings
	FetcherWorkers  int
	FetcherTimeout  time.Duration
	FetcherInterval time.Duration
	FetcherMaxAge   time.Duration
}

// Load returns a new Config struct populated from environment variables
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("ENV", "development"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/cryptonews?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		GroqAPIKey:         getEnv("GROQ_API_KEY", ""),
		CORSOrigins:        getEnvSlice("CORS_ORIGINS", []string{"*"}),
		RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 100),
		CacheTTL:           getEnvInt("CACHE_TTL", 60),
		EnableMetrics:      getEnvBool("ENABLE_METRICS", false),
		FetcherWorkers:     getEnvInt("FETCHER_WORKERS", 50),
		FetcherTimeout:     getEnvDuration("FETCHER_TIMEOUT", 10*time.Second),
		FetcherInterval:    getEnvDuration("FETCH_INTERVAL", 3*time.Minute),
		FetcherMaxAge:      getEnvDuration("FETCHER_MAX_AGE", 7*24*time.Hour),
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
