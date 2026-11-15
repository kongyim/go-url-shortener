package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

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

func shortenHandler(c *gin.Context) {
	var request ShortenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, "A valid HTTP or HTTPS URL is required\n")
		return
	}

	// Temporary hardcoded short code.
	c.JSON(http.StatusCreated, ShortenResponse{
		ShortCode:   "abc123",
		OriginalURL: request.URL,
	})
}

func setupRouter() *gin.Engine {
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
	router.POST("/shorten", shortenHandler)
	return router
}

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal("Failed to load .env: ", err)
	}

	router := setupRouter()
	log.Println("Starting server on port 8080...")
	log.Fatal(router.Run(":8080"))
}
