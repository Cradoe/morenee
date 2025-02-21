package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	ctx    context.Context
}

func New(redisAddr string, db int) *Cache {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   db,
	})

	return &Cache{
		client: client,
		ctx:    context.Background(),
	}
}

// Set stores a key-value pair with an expiration time
func (c *Cache) Set(key string, value string, expiration time.Duration) error {
	return c.client.Set(c.ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (c *Cache) Get(key string) (string, error) {
	return c.client.Get(c.ctx, key).Result()
}

// Delete removes a key from the cache
func (c *Cache) Delete(key string) error {
	return c.client.Del(c.ctx, key).Err()
}

// Exists checks if a key exists in cache
func (c *Cache) Exists(key string) (bool, error) {
	count, err := c.client.Exists(c.ctx, key).Result()
	return count > 0, err
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	return c.client.Close()
}
