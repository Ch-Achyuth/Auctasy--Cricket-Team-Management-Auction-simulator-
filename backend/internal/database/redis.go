package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisDB wraps the Redis client.
type RedisDB struct {
	Client *redis.Client
}

// ConnectRedis initializes the Redis client using the provided connection string.
// A typical connection string looks like: redis://user:password@localhost:6379/0
func ConnectRedis(redisURL string) (*RedisDB, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opts)

	// Ping the database to verify the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("unable to connect to Redis: %v", err)
	}

	return &RedisDB{Client: client}, nil
}

// Close gracefully shuts down the Redis connection.
func (r *RedisDB) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}
