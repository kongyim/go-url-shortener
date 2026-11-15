package main

import (
	"bytes"
	"encoding/json"
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

			setupRouter().ServeHTTP(recorder, req)

			assert.Equal(t, tt.wantStatus, recorder.Code)
		})
	}
}

func TestShortenHandler(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		method       string
		body         string
		wantStatus   int
		wantResponse *ShortenResponse
		wantAllow    string
		wantError    string
	}{
		{
			name:       "valid URL returns short code",
			url:        "/shorten",
			method:     http.MethodPost,
			body:       `{"url":"https://google.com"}`,
			wantStatus: http.StatusCreated,
			wantResponse: &ShortenResponse{
				OriginalURL: "https://google.com",
				ShortCode:   "abc123",
			},
		},
		{
			name:       "invalid JSON",
			url:        "/shorten",
			method:     http.MethodPost,
			body:       `{`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "missing URL",
			url:        "/shorten",
			method:     http.MethodPost,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "empty URL",
			url:        "/shorten",
			method:     http.MethodPost,
			body:       `{"url":""}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "surrounding whitespace",
			url:        "/shorten",
			method:     http.MethodPost,
			body:       `{"url": " https://google.com"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "invalid URL format",
			url:        "/shorten",
			method:     http.MethodPost,
			body:       `{"url": "https://example.com/%zz"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "unsupported scheme",
			url:        "/shorten",
			method:     http.MethodPost,
			body:       `{"url": "ftp://example.com"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:       "missing hostname",
			url:        "/shorten",
			method:     http.MethodPost,
			body:       `{"url": "https:///path"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "A valid HTTP or HTTPS URL is required\n",
		},
		{
			name:         "URL with credentials",
			url:          "/shorten",
			method:       http.MethodPost,
			body:         `{"url": "https://user:password@example.com"}`,
			wantStatus:   http.StatusCreated,
			wantResponse: &ShortenResponse{OriginalURL: "https://user:password@example.com", ShortCode: "abc123"},
		},
		{
			name:         "URL with fragment",
			url:          "/shorten",
			method:       http.MethodPost,
			body:         `{"url": "https://example.com/#section"}`,
			wantStatus:   http.StatusCreated,
			wantResponse: &ShortenResponse{OriginalURL: "https://example.com/#section", ShortCode: "abc123"},
		},
		{
			name:         "numeric port is not range checked",
			url:          "/shorten",
			method:       http.MethodPost,
			body:         `{"url": "https://example.com:99999"}`,
			wantStatus:   http.StatusCreated,
			wantResponse: &ShortenResponse{OriginalURL: "https://example.com:99999", ShortCode: "abc123"},
		},
		{
			name:       "GET is not allowed",
			url:        "/shorten",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			setupRouter().ServeHTTP(recorder, req)

			require.Equal(t, tt.wantStatus, recorder.Code)
			assert.Equal(t, tt.wantAllow, recorder.Header().Get("Allow"))

			if tt.wantError != "" {
				assert.Equal(t, tt.wantError, recorder.Body.String())
			}

			if tt.wantResponse != nil {
				var response ShortenResponse
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
				assert.Equal(t, *tt.wantResponse, response)
			}
		})
	}
}
