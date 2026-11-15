package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"time"

	"go-url-shortener/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal("Failed to load .env: ", err)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := database.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Successfully connected to PostgreSQL!")

	// Step 3.3.1: Insert a URL into PostgreSQL

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	var randomBytes [6]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		log.Fatal("Failed to generate short code: ", err)
	}
	shortCode := fmt.Sprintf("demo%x", randomBytes)

	var id int64
	var originalURL string
	var createdAt time.Time

	query := `
        INSERT INTO urls (short_code, original_url)
        VALUES ($1, $2)
        RETURNING id, original_url, created_at
    `

	err = db.QueryRowContext(
		ctx,
		query,
		shortCode,
		"https://example.com",
	).Scan(
		&id,
		&originalURL,
		&createdAt,
	)

	if err != nil {
		log.Fatal("Failed to insert URL: ", err)
	}

	fmt.Println("Inserted URL successfully!")
	fmt.Println("ID:", id)
	fmt.Println("Short Code:", shortCode)
	fmt.Println("Original URL:", originalURL)
	fmt.Println("Created At:", createdAt)

}
