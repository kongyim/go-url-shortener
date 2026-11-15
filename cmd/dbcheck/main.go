package main

import (
	"context"
	"log"
	"os"

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
}
