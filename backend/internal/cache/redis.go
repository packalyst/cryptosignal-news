package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis wraps the redis client
type Redis struct {
	client *redis.Client
}

// Config holds Redis configuration
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// NewRedis creates a new Redis client
func NewRedis(cfg Config) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Redis{client: client}, nil
}

// NewRedisFromURL creates a new Redis client from a URL
func NewRedisFromURL(redisURL string) (*Redis, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Redis{client: client}, nil
}

// Client returns the underlying redis client
func (r *Redis) Client() *redis.Client {
	return r.client
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	return r.client.Close()
}

// Health checks the Redis health
func (r *Redis) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Get retrieves a value from Redis
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set stores a value in Redis with optional expiration
func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Delete removes a key from Redis
func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

// Incr increments a key's value
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Expire sets expiration on a key
func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// ZAdd adds members to a sorted set
func (r *Redis) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return r.client.ZAdd(ctx, key, members...).Err()
}

// ZRemRangeByScore removes members from a sorted set by score range
func (r *Redis) ZRemRangeByScore(ctx context.Context, key, min, max string) error {
	return r.client.ZRemRangeByScore(ctx, key, min, max).Err()
}

// ZCard returns the number of members in a sorted set
func (r *Redis) ZCard(ctx context.Context, key string) (int64, error) {
	return r.client.ZCard(ctx, key).Result()
}

// ZCount counts members in a sorted set by score range
func (r *Redis) ZCount(ctx context.Context, key, min, max string) (int64, error) {
	return r.client.ZCount(ctx, key, min, max).Result()
}

// TTL returns the remaining time to live of a key
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// SAdd adds members to a set
func (r *Redis) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SAdd(ctx, key, members...).Err()
}

// SIsMember checks if a value is a member of a set
func (r *Redis) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.client.SIsMember(ctx, key, member).Result()
}

// SMembers returns all members of a set
func (r *Redis) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

// Pipeline creates a new pipeline for batch operations
func (r *Redis) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}
