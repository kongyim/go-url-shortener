package main

import (
	"context"
	"errors"

	"go-url-shortener/internal/database"
)

type MockURLRepository struct {
	CreateErr     error
	CreateCalls   int
	CreateContext context.Context
	CreatedURL    *database.URL
}

func (m *MockURLRepository) Create(
	ctx context.Context,
	shortCode string,
	originalURL string,
) (*database.URL, error) {
	m.CreateCalls++
	m.CreateContext = ctx

	if m.CreateErr != nil {
		return nil, m.CreateErr
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
