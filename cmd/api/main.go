package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"go-url-shortener/internal/database"
	"go-url-shortener/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Status: "ok"})
}

const maxCreateAttempts = 3

func shortenHandler(c *gin.Context, repo URLRepository) {
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

func redirectHandler(c *gin.Context, repo URLReader) {
	shortCode := c.Param("shortCode")

	foundURL, err := repo.GetByShortCode(c.Request.Context(), shortCode)
	if errors.Is(err, sql.ErrNoRows) {
		c.String(http.StatusNotFound, "URL not found\n")
		return
	}
	if err != nil {
		log.Printf("Get URL by short code failed: %v", err)
		c.String(http.StatusInternalServerError, "Failed to retrieve URL\n")
		return
	}

	c.Redirect(
		http.StatusFound,
		foundURL.OriginalURL,
	)
}

func setupRouter(repo URLRepository) *gin.Engine {
	router := gin.Default()
	router.HandleMethodNotAllowed = true
	router.NoMethod(func(c *gin.Context) {
		switch c.Request.URL.Path {
		case "/health":
			c.Header("Allow", http.MethodGet)
		case "/shorten":
			c.Header("Allow", http.MethodPost)
		}

		c.String(http.StatusMethodNotAllowed, "Method not allowed\n")
	})
	router.GET("/health", healthHandler)
	router.POST("/shorten", func(c *gin.Context) {
		shortenHandler(c, repo)
	})
	router.GET("/shorten", func(c *gin.Context) {
		c.Header("Allow", http.MethodPost)
		c.String(http.StatusMethodNotAllowed, "Method not allowed\n")
	})
	router.GET("/:shortCode", func(c *gin.Context) {
		redirectHandler(c, repo)
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

	router := setupRouter(repo)
	log.Println("Starting server on port 8080...")
	log.Fatal(router.Run(":8080"))
}
