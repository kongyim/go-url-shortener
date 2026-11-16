// lookupbench compares warm, sequential repository reads and Redis cache hits.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"time"

	"go-url-shortener/internal/cache"
	"go-url-shortener/internal/database"
	"go-url-shortener/internal/repository"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type summary struct {
	AverageMS float64 `json:"averageMs"`
	P50MS     float64 `json:"p50Ms"`
	P95MS     float64 `json:"p95Ms"`
}

func summarize(samples []float64) summary {
	var total float64
	for _, sample := range samples {
		total += sample
	}
	sort.Float64s(samples)
	return summary{
		AverageMS: total / float64(len(samples)),
		P50MS:     samples[int(math.Ceil(float64(len(samples))*0.50))-1],
		P95MS:     samples[int(math.Ceil(float64(len(samples))*0.95))-1],
	}
}

func run() error {
	samples := flag.Int("samples", 5000, "measured lookups per backend")
	warmup := flag.Int("warmup", 100, "unmeasured lookups per backend")
	flag.Parse()
	if *samples < 1 || *warmup < 0 {
		return fmt.Errorf("samples must be positive and warmup nonnegative")
	}
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return err
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	db, err := database.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	var shortCode, originalURL, postgresVersion string
	var rowCount int
	if err := db.QueryRowContext(ctx, "SELECT short_code, original_url FROM urls ORDER BY id LIMIT 1").Scan(&shortCode, &originalURL); err != nil {
		return fmt.Errorf("select existing benchmark URL: %w", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM urls").Scan(&rowCount); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&postgresVersion); err != nil {
		return err
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 200 * time.Millisecond, ReadTimeout: 200 * time.Millisecond, WriteTimeout: 200 * time.Millisecond, ContextTimeoutEnabled: true, MaxRetries: -1})
	defer client.Close()
	urlCache := cache.NewRedisURLCache(client)
	// A unique key avoids overwriting or deleting any application cache entry.
	benchmarkCode := fmt.Sprintf("benchmark:%d", time.Now().UnixNano())
	if err := urlCache.Set(ctx, benchmarkCode, originalURL, 10*time.Minute); err != nil {
		return err
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Second)
		defer cleanupCancel()
		if err := client.Del(cleanupCtx, "url:"+benchmarkCode).Err(); err != nil {
			fmt.Fprintln(os.Stderr, "cleanup failed; benchmark key expires after 10 minutes:", err)
		}
	}()
	repo := repository.NewURLRepository(db)
	postgresLookup := func() error {
		found, err := repo.GetByShortCode(ctx, shortCode)
		if err != nil {
			return err
		}
		if found.OriginalURL != originalURL {
			return fmt.Errorf("PostgreSQL returned an unexpected URL")
		}
		return nil
	}
	redisLookup := func() error {
		lookupCtx, lookupCancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer lookupCancel()
		found, err := urlCache.Get(lookupCtx, benchmarkCode)
		if err != nil {
			return err
		}
		if found != originalURL {
			return fmt.Errorf("Redis returned an unexpected URL")
		}
		return nil
	}
	for i := 0; i < *warmup; i++ {
		if err := postgresLookup(); err != nil {
			return err
		}
		if err := redisLookup(); err != nil {
			return err
		}
	}
	pgSamples, redisSamples := make([]float64, 0, *samples), make([]float64, 0, *samples)
	measure := func(lookup func() error, values *[]float64) error {
		start := time.Now()
		if err := lookup(); err != nil {
			return err
		}
		*values = append(*values, float64(time.Since(start).Nanoseconds())/1e6)
		return nil
	}
	for i := 0; i < *samples; i++ {
		if i%2 == 0 {
			if err := measure(postgresLookup, &pgSamples); err != nil {
				return err
			}
			if err := measure(redisLookup, &redisSamples); err != nil {
				return err
			}
		} else {
			if err := measure(redisLookup, &redisSamples); err != nil {
				return err
			}
			if err := measure(postgresLookup, &pgSamples); err != nil {
				return err
			}
		}
	}
	pg, r := summarize(pgSamples), summarize(redisSamples)
	output := struct {
		MeasuredAt      string             `json:"measuredAt"`
		Platform        string             `json:"platform"`
		GoVersion       string             `json:"goVersion"`
		PostgresVersion string             `json:"postgresVersion"`
		Rows            int                `json:"postgresRows"`
		Samples         int                `json:"samplesPerBackend"`
		Warmup          int                `json:"warmupPerBackend"`
		Postgres        summary            `json:"postgres"`
		Redis           summary            `json:"redis"`
		Speedup         map[string]float64 `json:"postgresDividedByRedis"`
	}{time.Now().UTC().Format(time.RFC3339), runtime.GOOS + "/" + runtime.GOARCH, runtime.Version(), postgresVersion, rowCount, *samples, *warmup, pg, r, map[string]float64{"average": pg.AverageMS / r.AverageMS, "p50": pg.P50MS / r.P50MS, "p95": pg.P95MS / r.P95MS}}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
