package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-url-shortener/internal/cache"
	"go-url-shortener/internal/database"
	"go-url-shortener/internal/metrics"
	"go-url-shortener/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type URLRepository interface {
	URLCreator
	URLReader
}

type URLCreator interface {
	Create(
		ctx context.Context,
		shortCode string,
		originalURL string,
	) (*database.URL, error)
}

type URLReader interface {
	GetByShortCode(
		ctx context.Context,
		shortCode string,
	) (*database.URL, error)
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ShortenRequest struct {
	URL string `json:"url" binding:"required,http_url"`
}

type ShortenResponse struct {
	ShortCode   string `json:"shortCode"`
	OriginalURL string `json:"originalUrl"`
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

const maxCreateAttempts = 3
const urlCacheTTL = time.Hour
const cacheTimeout = 200 * time.Millisecond

func cacheURL(ctx context.Context, urlCache cache.URLCache, shortCode, originalURL string) {
	if urlCache == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, cacheTimeout)
	defer cancel()
	if err := urlCache.Set(ctx, shortCode, originalURL, urlCacheTTL); err != nil {
		log.Printf("Cache URL failed: %v", err)
	}
}

func shortenHandler(c *gin.Context, repo URLRepository, urlCache cache.URLCache) {
	var request ShortenRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(
			http.StatusBadRequest,
			"A valid HTTP or HTTPS URL is required\n",
		)
		return
	}

	for attempt := 0; attempt < maxCreateAttempts; attempt++ {
		shortCode, err := generateShortCode()
		if err != nil {
			log.Printf("Generate short code failed: %v", err)
			c.String(
				http.StatusInternalServerError,
				"Failed to generate short code\n",
			)
			return
		}

		savedURL, err := repo.Create(
			c.Request.Context(),
			shortCode,
			request.URL,
		)

		if errors.Is(err, repository.ErrShortCodeConflict) {
			continue
		}

		if err != nil {
			log.Printf("Create URL failed: %v", err)
			c.String(
				http.StatusInternalServerError,
				"Failed to create URL\n",
			)
			return
		}

		cacheURL(c.Request.Context(), urlCache, savedURL.ShortCode, savedURL.OriginalURL)

		c.JSON(http.StatusCreated, ShortenResponse{
			ShortCode:   savedURL.ShortCode,
			OriginalURL: savedURL.OriginalURL,
		})
		return
	}

	err := fmt.Errorf(
		"failed after %d short code attempts",
		maxCreateAttempts,
	)
	log.Printf("%v", err)

	c.String(
		http.StatusInternalServerError,
		"Failed to create URL\n",
	)
}

func redirectHandler(c *gin.Context, repo URLReader, urlCache cache.URLCache, cacheMetrics *metrics.CacheMetrics) {
	shortCode := c.Param("shortCode")
	if urlCache != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), cacheTimeout)
		start := time.Now()
		originalURL, err := urlCache.Get(ctx, shortCode)
		duration := time.Since(start)
		cancel()

		if cacheMetrics != nil {
			cacheMetrics.RecordRedisGETDuration(duration)
		}
		switch {
		case err == nil && originalURL != "":
			if cacheMetrics != nil {
				cacheMetrics.RecordHit()
			}

			c.Redirect(http.StatusFound, originalURL)
			return

		case errors.Is(err, cache.ErrCacheMiss):
			if cacheMetrics != nil {
				cacheMetrics.RecordMiss()
			}

		case err != nil:
			if cacheMetrics != nil {
				cacheMetrics.RecordError()
			}

			log.Printf("Get cached URL failed: %v", err)

		default:
			if cacheMetrics != nil {
				cacheMetrics.RecordMiss()
			}
		}
	}

	start := time.Now()
	foundURL, err := repo.GetByShortCode(c.Request.Context(), shortCode)
	if cacheMetrics != nil {
		cacheMetrics.RecordPostgresLookupDuration(time.Since(start))
	}
	if errors.Is(err, sql.ErrNoRows) {
		c.String(http.StatusNotFound, "URL not found\n")
		return
	}
	if err != nil {
		log.Printf("Get URL by short code failed: %v", err)
		c.String(http.StatusInternalServerError, "Failed to retrieve URL\n")
		return
	}

	cacheURL(c.Request.Context(), urlCache, shortCode, foundURL.OriginalURL)

	c.Redirect(
		http.StatusFound,
		foundURL.OriginalURL,
	)
}

func metricsHandler(
	c *gin.Context,
	cacheMetrics *metrics.CacheMetrics,
) {

	if cacheMetrics == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "metrics unavailable",
		})
		return
	}

	stats := cacheMetrics.Snapshot()

	c.JSON(http.StatusOK, gin.H{
		"hits":                           stats.Hits,
		"misses":                         stats.Misses,
		"errors":                         stats.Errors,
		"hitRatePercent":                 stats.HitRate(),
		"redisGetCount":                  stats.RedisGETCount,
		"averageRedisGetLatencyMs":       stats.AverageRedisGETLatencyMS(),
		"postgresLookupCount":            stats.PostgresLookupCount,
		"averagePostgresLookupLatencyMs": stats.AveragePostgresLookupLatencyMS(),
	})
}

func setupRouter(db databasePinger, repo URLRepository, urlCache cache.URLCache, cacheMetrics *metrics.CacheMetrics) *gin.Engine {
	router := gin.Default()
	router.HandleMethodNotAllowed = true
	router.NoMethod(func(c *gin.Context) {
		switch c.Request.URL.Path {
		case "/metrics":
			c.Header("Allow", http.MethodGet)
		case "/health", "/readyz":
			c.Header("Allow", http.MethodGet)
		case "/shorten":
			c.Header("Allow", http.MethodPost)
		}

		c.String(http.StatusMethodNotAllowed, "Method not allowed\n")
	})
	router.GET("/metrics", func(c *gin.Context) {
		metricsHandler(c, cacheMetrics)
	})
	router.GET("/health", healthHandler)
	router.GET("/readyz", readinessHandler(db))
	router.POST("/shorten", func(c *gin.Context) {
		shortenHandler(c, repo, urlCache)
	})
	router.GET("/shorten", func(c *gin.Context) {
		c.Header("Allow", http.MethodPost)
		c.String(http.StatusMethodNotAllowed, "Method not allowed\n")
	})
	router.GET("/:shortCode", func(c *gin.Context) {
		redirectHandler(c, repo, urlCache, cacheMetrics)
	})
	return router
}

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal("Failed to load .env: ", err)
	}

	// Get database connection string
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	// Connect to PostgreSQL
	db, err := database.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	defer db.Close()

	// Create repository
	repo := repository.NewURLRepository(db)
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr:                  addr,
		DialTimeout:           cacheTimeout,
		ReadTimeout:           cacheTimeout,
		WriteTimeout:          cacheTimeout,
		ContextTimeoutEnabled: true,
		MaxRetries:            -1,
	})
	defer client.Close()
	urlCache := cache.NewRedisURLCache(client)

	cacheMetrics := &metrics.CacheMetrics{}

	router := setupRouter(db, repo, urlCache, cacheMetrics)
	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Starting server on port http://localhost%s...\n", server.Addr)

	shutdownCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.ListenAndServe()
	}()

	log.Println("Starting server on port 8080...")

	select {
	case <-shutdownCtx.Done():
		log.Println("Shutdown signal received")

	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server failed: %v", err)
		}
		return
	}

	// Stop handling the signal so a second Ctrl+C can terminate
	// the process using its default behavior.
	stop()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	log.Println("Shutting down HTTP server...")

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown failed: %v", err)
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("HTTP server close failed: %v", closeErr)
		}
		return
	}

	log.Println("HTTP server stopped gracefully")

}
