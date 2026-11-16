package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		wantStatus int
	}{
		{
			name:       "GET returns OK",
			url:        "/health",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST is not allowed",
			url:        "/health",
			method:     http.MethodPost,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			recorder := httptest.NewRecorder()

			setupRouter(&MockURLRepository{}).ServeHTTP(recorder, req)

			assert.Equal(t, tt.wantStatus, recorder.Code)
		})
	}
}

func TestShortenHandler(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		body         string
		wantStatus   int
		wantResponse *ShortenResponse
		wantAllow    string
		wantError    string
		createErr    error
	}{
		{
			name: "valid URL returns short code",

			method:     http.MethodPost,
			body:       `{"url":"https://google.com"}`,
			wantStatus: http.StatusCreated,
			wantResponse: &ShortenResponse{
				OriginalURL: "https://google.com",
			},
		},
		{
			name: "invalid JSON",

			method:     http.MethodPost,
			body:       `{`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name: "missing URL",

			method:     http.MethodPost,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name: "empty URL",

			method:     http.MethodPost,
			body:       `{"url":""}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name: "surrounding whitespace",

			method:     http.MethodPost,
			body:       `{"url": " https://google.com"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name: "invalid URL format",

			method:     http.MethodPost,
			body:       `{"url": "https://example.com/%zz"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name: "unsupported scheme",

			method:     http.MethodPost,
			body:       `{"url": "ftp://example.com"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name: "missing hostname",

			method:     http.MethodPost,
			body:       `{"url": "https:///path"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "missing scheme",
			method:     http.MethodPost,
			body:       `{"url": "example.com/path"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "relative path",
			method:     http.MethodPost,
			body:       `{"url": "/some/path"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "protocol-relative URL",
			method:     http.MethodPost,
			body:       `{"url": "//example.com/path"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "scheme without host",
			method:     http.MethodPost,
			body:       `{"url": "https://"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "space in hostname",
			method:     http.MethodPost,
			body:       `{"url": "https://exa mple.com"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "unclosed IPv6 address",
			method:     http.MethodPost,
			body:       `{"url": "https://[::1"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "non-numeric port",
			method:     http.MethodPost,
			body:       `{"url": "https://example.com:abc"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "negative port",
			method:     http.MethodPost,
			body:       `{"url": "https://example.com:-1"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "newline in URL",
			method:     http.MethodPost,
			body:       `{"url": "https://example.com/\npath"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "JavaScript scheme",
			method:     http.MethodPost,
			body:       `{"url": "javascript:alert(1)"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "null URL",
			method:     http.MethodPost,
			body:       `{"url": null}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "numeric URL",
			method:     http.MethodPost,
			body:       `{"url": 123}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:         "URL with credentials",
			method:       http.MethodPost,
			body:         `{"url": "https://user:password@example.com"}`,
			wantStatus:   http.StatusCreated,
			wantResponse: &ShortenResponse{OriginalURL: "https://user:password@example.com"},
		},
		{
			name:         "URL with fragment",
			method:       http.MethodPost,
			body:         `{"url": "https://example.com/#section"}`,
			wantStatus:   http.StatusCreated,
			wantResponse: &ShortenResponse{OriginalURL: "https://example.com/#section"},
		},
		{
			name:         "numeric port is not range checked",
			method:       http.MethodPost,
			body:         `{"url": "https://example.com:99999"}`,
			wantStatus:   http.StatusCreated,
			wantResponse: &ShortenResponse{OriginalURL: "https://example.com:99999"},
		},
		{
			name:       "repository error",
			method:     http.MethodPost,
			body:       `{"url":"https://google.com"}`,
			createErr:  errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to create URL\n",
		},
		{
			name: "GET is not allowed",

			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/shorten", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			repo := &MockURLRepository{CreateErr: tt.createErr}
			setupRouter(repo).ServeHTTP(recorder, req)

			require.Equal(t, tt.wantStatus, recorder.Code)
			assert.Equal(t, tt.wantAllow, recorder.Header().Get("Allow"))

			if tt.wantError != "" {
				assert.Equal(t, tt.wantError, recorder.Body.String())
			}

			if tt.wantResponse != nil {
				var response ShortenResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				assert.Equal(t, 1, repo.CreateCalls)
				assert.Equal(t, req.Context(), repo.CreateContext)
				require.NotNil(t, repo.CreatedURL)
				assert.Equal(t, tt.wantResponse.OriginalURL, repo.CreatedURL.OriginalURL)
				assert.Regexp(t, `^[0-9a-f]{8}$`, repo.CreatedURL.ShortCode)
				assert.Equal(t, repo.CreatedURL.OriginalURL, response.OriginalURL)
				assert.Equal(t, repo.CreatedURL.ShortCode, response.ShortCode)
			} else if tt.createErr != nil {
				assert.Equal(t, 1, repo.CreateCalls)
				assert.Nil(t, repo.CreatedURL)
			} else {
				assert.Zero(t, repo.CreateCalls, "invalid requests must not save a URL")
			}
		})
	}
}

func TestShortenHandlerRetriesCollision(t *testing.T) {
	tests := []struct {
		name       string
		collisions int
		wantCalls  int
		wantStatus int
	}{
		{
			name:       "succeeds after one collision",
			collisions: 1,
			wantCalls:  2,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "succeeds on the final attempt",
			collisions: maxCreateAttempts - 1,
			wantCalls:  maxCreateAttempts,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "stops after exhausting attempts",
			collisions: maxCreateAttempts,
			wantCalls:  maxCreateAttempts,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "succeeds without collision",
			collisions: 0,
			wantCalls:  1,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockURLRepository{CollisionsRemaining: tt.collisions}
			req := httptest.NewRequest(
				http.MethodPost,
				"/shorten",
				bytes.NewBufferString(`{"url":"https://example.com"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			setupRouter(repo).ServeHTTP(recorder, req)

			require.Equal(t, tt.wantStatus, recorder.Code)
			assert.Equal(t, tt.wantCalls, repo.CreateCalls)
			assert.Zero(t, repo.CollisionsRemaining)

			if tt.wantStatus == http.StatusInternalServerError {
				assert.Nil(t, repo.CreatedURL)
				assert.Equal(t, "Failed to create URL\n", recorder.Body.String())
				return
			}

			require.NotNil(t, repo.CreatedURL)
			assert.Equal(t, "https://example.com", repo.CreatedURL.OriginalURL)
			assert.Regexp(t, `^[0-9a-f]{8}$`, repo.CreatedURL.ShortCode)

			var response ShortenResponse
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			assert.Equal(t, "https://example.com", response.OriginalURL)
			assert.Equal(t, repo.CreatedURL.ShortCode, response.ShortCode)
		})
	}
}
