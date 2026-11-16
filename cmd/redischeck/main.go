package main

import (
	"context"
	"errors"
	"fmt"
	"go-url-shortener/internal/cache"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		log.Fatal("REDIS_ADDR is required")
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal("Redis connection failed: ", err)
	}

	urlCache := cache.NewRedisURLCache(client)

	err := urlCache.Set(
		ctx,
		"demo01",
		"https://example.com",
		time.Hour,
	)
	if err != nil {
		log.Fatal("Cache SET failed: ", err)
	}

	originalURL, err := urlCache.Get(ctx, "demo01")
	if err != nil {
		log.Fatal("Cache GET failed: ", err)
	}

	fmt.Println("Cached URL:", originalURL)

	_, err = urlCache.Get(ctx, "missing-code-123")

	if errors.Is(err, cache.ErrCacheMiss) {
		fmt.Println("Cache miss detected correctly")
	} else if err != nil {
		log.Fatal("Unexpected cache error: ", err)
	} else {
		log.Fatal("Expected cache miss, but key exists")
	}

}
