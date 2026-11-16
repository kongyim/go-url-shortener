package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type URLCache interface {
	Get(ctx context.Context, shortCode string) (string, error)

	Set(
		ctx context.Context,
		shortCode string,
		originalURL string,
		ttl time.Duration,
	) error
}

var ErrCacheMiss = errors.New("cache miss")

type RedisURLCache struct {
	client *redis.Client
}

func NewRedisURLCache(client *redis.Client) *RedisURLCache {
	return &RedisURLCache{
		client: client,
	}
}

func cacheKey(shortCode string) string {
	return "url:" + shortCode
}

func (r *RedisURLCache) Get(
	ctx context.Context,
	shortCode string,
) (string, error) {
	originalURL, err := r.client.Get(
		ctx,
		cacheKey(shortCode),
	).Result()

	if errors.Is(err, redis.Nil) {
		return "", ErrCacheMiss
	}

	if err != nil {
		return "", fmt.Errorf("get cached URL: %w", err)
	}

	return originalURL, nil
}

func (r *RedisURLCache) Set(
	ctx context.Context,
	shortCode string,
	originalURL string,
	ttl time.Duration,
) error {
	err := r.client.Set(
		ctx,
		cacheKey(shortCode),
		originalURL,
		ttl,
	).Err()

	if err != nil {
		return fmt.Errorf("set cached URL: %w", err)
	}

	return nil
}
