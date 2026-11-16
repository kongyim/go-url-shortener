package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-url-shortener/internal/cache"
	"go-url-shortener/internal/database"
	"go-url-shortener/internal/metrics"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockURLCache struct {
	value    string
	getErr   error
	setErr   error
	getCalls int
	setCalls int

	getKey string
	setKey string
	stored string
	ttl    time.Duration
}

func (m *mockURLCache) Get(ctx context.Context, key string) (string, error) {
	m.getCalls++
	m.getKey = key
	return m.value, m.getErr
}

func (m *mockURLCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	m.setCalls++
	m.setKey, m.stored, m.ttl = key, value, ttl
	return m.setErr
}

func TestRedirectHandlerCache(t *testing.T) {
	for _, tt := range []struct {
		name       string
		cached     string
		getErr     error
		setErr     error
		repoErr    error
		missing    bool
		wantStatus int
		wantReads  int
		wantWrites int

		wantHits   uint64
		wantMisses uint64
		wantErrors uint64
	}{
		{name: "hit skips database", cached: "https://cached.example/path", wantStatus: 302, wantHits: 1},
		{name: "empty cached value counts as miss", wantStatus: 302, wantReads: 1, wantWrites: 1, wantMisses: 1},
		{name: "miss populates cache", getErr: cache.ErrCacheMiss, wantStatus: 302, wantReads: 1, wantWrites: 1, wantMisses: 1},
		{name: "read failure falls back", getErr: errors.New("redis unavailable"), wantStatus: 302, wantReads: 1, wantWrites: 1, wantErrors: 1},
		{name: "write failure still redirects", getErr: cache.ErrCacheMiss, setErr: errors.New("redis unavailable"), wantStatus: 302, wantReads: 1, wantWrites: 1, wantMisses: 1},
		{name: "missing URL is not cached", getErr: cache.ErrCacheMiss, missing: true, wantStatus: 404, wantReads: 1, wantMisses: 1},
		{
			name:       "database error after cache miss is not cached",
			getErr:     cache.ErrCacheMiss,
			repoErr:    errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
			wantReads:  1,
			wantWrites: 0,
			wantMisses: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockURLRepository{GetErr: tt.repoErr}
			if !tt.missing && tt.repoErr == nil {
				repo.FoundURL = &database.URL{ShortCode: "abc123", OriginalURL: "https://stored.example/path"}
			}
			urlCache := &mockURLCache{value: tt.cached, getErr: tt.getErr, setErr: tt.setErr}
			recorder := httptest.NewRecorder()
			cacheMetrics := &metrics.CacheMetrics{}
			setupRouter(repo, urlCache, cacheMetrics).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/abc123", nil))

			stats := cacheMetrics.Snapshot()
			assert.Equal(t, tt.wantHits, stats.Hits, "cache hits")
			assert.Equal(t, tt.wantMisses, stats.Misses, "cache misses")
			assert.Equal(t, tt.wantErrors, stats.Errors, "cache lookup errors")
			require.Equal(t, tt.wantStatus, recorder.Code)
			assert.Equal(t, 1, urlCache.getCalls)
			assert.Equal(t, "abc123", urlCache.getKey)
			assert.Equal(t, tt.wantReads, repo.GetCalls)
			assert.Equal(t, tt.wantWrites, urlCache.setCalls)
			assert.Equal(t, uint64(1), stats.RedisGETCount)
			if tt.wantStatus == 302 {
				destination := tt.cached
				if destination == "" {
					destination = repo.FoundURL.OriginalURL
				}
				assert.Equal(t, destination, recorder.Header().Get("Location"))
			} else {
				assert.Empty(t, recorder.Header().Get("Location"))
			}
			if tt.wantWrites > 0 {
				assert.Equal(t, "abc123", urlCache.setKey)
				assert.Equal(t, repo.FoundURL.OriginalURL, urlCache.stored)
				assert.Equal(t, time.Hour, urlCache.ttl)
			}
		})
	}
}

func TestShortenHandlerCache(t *testing.T) {
	for _, tt := range []struct {
		name       string
		body       string
		createErr  error
		cacheErr   error
		wantStatus int
		wantWrites int
	}{
		{name: "created URL is cached", body: `{"url":"https://example.com"}`, wantStatus: 201, wantWrites: 1},
		{name: "cache failure still returns created", body: `{"url":"https://example.com"}`, cacheErr: errors.New("redis unavailable"), wantStatus: 201, wantWrites: 1},
		{name: "failed insert is not cached", body: `{"url":"https://example.com"}`, createErr: errors.New("database unavailable"), wantStatus: 500},
		{name: "invalid input is not cached", body: `{}`, wantStatus: 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockURLRepository{CreateErr: tt.createErr}
			urlCache := &mockURLCache{setErr: tt.cacheErr}
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			setupRouter(repo, urlCache, nil).ServeHTTP(recorder, req)
			require.Equal(t, tt.wantStatus, recorder.Code)
			assert.Zero(t, urlCache.getCalls)
			assert.Equal(t, tt.wantWrites, urlCache.setCalls)
			if tt.wantWrites > 0 {
				require.NotNil(t, repo.CreatedURL)
				assert.Equal(t, repo.CreatedURL.ShortCode, urlCache.setKey)
				assert.Equal(t, repo.CreatedURL.OriginalURL, urlCache.stored)
				assert.Equal(t, time.Hour, urlCache.ttl)
			}
		})
	}
}
