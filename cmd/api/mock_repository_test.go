package main

import (
	"context"
	"errors"

	"go-url-shortener/internal/database"
	"go-url-shortener/internal/repository"
)

type MockURLRepository struct {
	CreateErr           error
	CreateCalls         int
	CreateContext       context.Context
	CreatedURL          *database.URL
	CollisionsRemaining int
}

func (m *MockURLRepository) Create(
	ctx context.Context,
	shortCode string,
	originalURL string,
) (*database.URL, error) {
	m.CreateCalls++
	m.CreateContext = ctx

	// Simulate database error
	if m.CreateErr != nil {
		return nil, m.CreateErr
	}

	// Simulate short code collision
	if m.CollisionsRemaining > 0 {
		m.CollisionsRemaining--

		return nil, repository.ErrShortCodeConflict
	}

	if shortCode == "" || originalURL == "" {
		return nil, errors.New("missing URL data")
	}

	m.CreatedURL = &database.URL{
		ID:          1,
		ShortCode:   shortCode,
		OriginalURL: originalURL,
	}

	return m.CreatedURL, nil
}
