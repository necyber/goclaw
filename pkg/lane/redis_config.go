package lane

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds configuration for a Redis-backed Lane.
type RedisConfig struct {
	// Name is the lane name.
	Name string

	// Capacity is the maximum queue size.
	Capacity int

	// MaxConcurrency is the maximum number of concurrent workers.
	MaxConcurrency int

	// Backpressure is the overload strategy.
	Backpressure BackpressureStrategy

	// RedirectLane is the target lane for Redirect strategy.
	RedirectLane string

	// EnablePriority enables priority queue (Sorted Set) instead of FIFO (List).
	EnablePriority bool

	// EnableDedup enables task deduplication.
	EnableDedup bool

	// DedupTTL is the TTL for dedup keys.
	DedupTTL time.Duration

	// KeyPrefix is the Redis key prefix.
	KeyPrefix string

	// BlockTimeout is the BRPOP timeout for consuming tasks.
	BlockTimeout time.Duration
}

// DefaultRedisConfig returns a RedisConfig with sensible defaults.
func DefaultRedisConfig(name string) *RedisConfig {
	return &RedisConfig{
		Name:           name,
		Capacity:       10000,
		MaxConcurrency: 8,
		Backpressure:   Block,
		EnablePriority: false,
		EnableDedup:    false,
		DedupTTL:       1 * time.Hour,
		KeyPrefix:      "goclaw:lane:",
		BlockTimeout:   2 * time.Second,
	}
}

// Validate validates the Redis lane configuration.
func (c *RedisConfig) Validate() error {
	cfg := &Config{
		Name:           c.Name,
		Capacity:       c.Capacity,
		MaxConcurrency: c.MaxConcurrency,
		Backpressure:   c.Backpressure,
		RedirectLane:   c.RedirectLane,
	}
	return cfg.Validate()
}

// NewRedisClient creates a Redis client from the given options.
func NewRedisClient(opts *redis.Options) *redis.Client {
	return redis.NewClient(opts)
}

// NewRedisSentinelClient creates a Redis Sentinel failover client.
func NewRedisSentinelClient(opts *redis.FailoverOptions) *redis.Client {
	return redis.NewFailoverClient(opts)
}

// PingRedis checks if the Redis connection is healthy.
func PingRedis(ctx context.Context, client redis.Cmdable) error {
	return client.Ping(ctx).Err()
}
