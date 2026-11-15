package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go-url-shortener/internal/database"
)

type URLRepository struct {
	db *sql.DB
}

func NewURLRepository(db *sql.DB) *URLRepository {
	return &URLRepository{
		db: db,
	}
}

func (r *URLRepository) Create(
	ctx context.Context,
	shortCode string,
	originalURL string,
) (*database.URL, error) {
	query := `
        INSERT INTO urls (short_code, original_url)
        VALUES ($1, $2)
        RETURNING id, short_code, original_url, created_at
    `

	var savedURL database.URL

	err := r.db.QueryRowContext(
		ctx,
		query,
		shortCode,
		originalURL,
	).Scan(
		&savedURL.ID,
		&savedURL.ShortCode,
		&savedURL.OriginalURL,
		&savedURL.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create URL: %w", err)
	}

	return &savedURL, nil
}

func (r *URLRepository) GetByShortCode(
	ctx context.Context,
	shortCode string,
) (*database.URL, error) {
	query := `
        SELECT id, short_code, original_url, created_at
        FROM urls
        WHERE short_code = $1
    `

	var foundURL database.URL

	err := r.db.QueryRowContext(
		ctx,
		query,
		shortCode,
	).Scan(
		&foundURL.ID,
		&foundURL.ShortCode,
		&foundURL.OriginalURL,
		&foundURL.CreatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(
			"get URL by short code %q: %w",
			shortCode,
			sql.ErrNoRows,
		)
	}

	if err != nil {
		return nil, fmt.Errorf(
			"get URL by short code: %w",
			err,
		)
	}

	return &foundURL, nil
}
