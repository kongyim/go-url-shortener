package main

import (
	"context"
	"errors"
	"fmt"
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

	// SET with 1 hour TTL
	err := client.Set(
		ctx,
		"url:demo01",
		"https://example.com",
		time.Hour,
	).Err()

	if err != nil {
		log.Fatal("Redis SET failed: ", err)
	}

	// GET cached URL
	value, err := client.Get(
		ctx,
		"url:demo01",
	).Result()

	if errors.Is(err, redis.Nil) {
		log.Println("Cache miss")
		return
	}
	if err != nil {
		log.Fatal("Redis GET failed: ", err)
	}

	fmt.Println("Cached URL:", value)
}
