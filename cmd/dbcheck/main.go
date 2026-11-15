package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"go-url-shortener/internal/database"
	"go-url-shortener/internal/repository"

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
	repo := repository.NewURLRepository(db)

	// Step 3.3.1: Insert a URL into PostgreSQL

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	shortCode := strconv.FormatInt(time.Now().UnixNano(), 36)

	savedURL, err := repo.Create(
		ctx,
		shortCode,
		"https://example.com",
	)

	if err != nil {
		log.Fatal("Failed to create URL: ", err)
	}

	fmt.Printf("Created URL: %+v\n", savedURL)

	// Step 3.3.2: Find the URL by short code

	// Find the URL we just created
	foundURL, err := repo.GetByShortCode(
		ctx,
		savedURL.ShortCode,
	)

	if err != nil {
		log.Fatal("Failed to find URL: ", err)
	}

	fmt.Printf("Found URL: %+v\n", foundURL)

	_, err = repo.GetByShortCode(
		ctx,
		"this-code-does-not-exist",
	)

	if errors.Is(err, sql.ErrNoRows) {
		fmt.Println("Correctly detected: URL not found")
	} else if err != nil {
		log.Fatal("Unexpected database error: ", err)
	} else {
		log.Fatal("Expected URL not found, but got no error")
	}
}
