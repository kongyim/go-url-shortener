package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type ShortenRequest struct {
	URL string `json:"url" binding:"required"`
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
		c.String(http.StatusBadRequest, "Invalid JSON\n")
		return
	}

	if request.URL == "" {
		c.String(http.StatusBadRequest, "URL is required\n")
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
		c.String(http.StatusMethodNotAllowed, "Method not allowed\n")
	})
	router.GET("/health", healthHandler)
	router.POST("/shorten", shortenHandler)
	return router
}

func main() {
	router := setupRouter()
	log.Println("Starting server on port 8080...")
	log.Fatal(router.Run(":8080"))
}
