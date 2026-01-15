package database

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisConnection manages Redis client connection
type RedisConnection struct {
	Client *redis.Client
}

// NewRedisConnection creates a new Redis connection
// Supports both simple addr and full URL (for Upstash, etc.)
func NewRedisConnection(addr, url, password string, db int, tlsSkipVerify bool) *RedisConnection {
	var client *redis.Client

	// If URL is provided (Upstash), use ParseURL
	// ParseURL handles TLS automatically when using rediss:// scheme
	if url != "" {
		opts, err := redis.ParseURL(url)
		if err != nil {
			panic(err)
		}

		// If TLS skip verify is enabled (for Upstash in Docker/Windows)
		if tlsSkipVerify && opts.TLSConfig != nil {
			opts.TLSConfig.InsecureSkipVerify = true
		}

		client = redis.NewClient(opts)
	} else {
		// Traditional connection with addr
		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		})
	}

	return &RedisConnection{Client: client}
}

// Ping verifies the connection is alive
func (r *RedisConnection) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisConnection) Close() error {
	return r.Client.Close()
}
