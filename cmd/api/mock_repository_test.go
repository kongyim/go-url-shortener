package main

import (
	"context"
	"database/sql"
	"errors"

	"go-url-shortener/internal/database"
	"go-url-shortener/internal/repository"
)

type MockURLRepository struct {
	// Create
	CreateErr           error
	CreateCalls         int
	CreateContext       context.Context
	CreatedURL          *database.URL
	CollisionsRemaining int

	// Read
	GetErr       error
	GetCalls     int
	GetContext   context.Context
	GetShortCode string
	FoundURL     *database.URL
}

var _ URLRepository = (*MockURLRepository)(nil)

func (m *MockURLRepository) GetByShortCode(
	ctx context.Context,
	shortCode string,
) (*database.URL, error) {
	m.GetCalls++
	m.GetContext = ctx
	m.GetShortCode = shortCode

	if m.GetErr != nil {
		return nil, m.GetErr
	}
	if m.FoundURL == nil {
		return nil, sql.ErrNoRows
	}

	return m.FoundURL, nil
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
