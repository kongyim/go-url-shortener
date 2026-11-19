package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakePinger struct{ err error }

func (p fakePinger) PingContext(context.Context) error { return p.err }

func TestReadinessHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "database available", want: http.StatusOK},
		{name: "database unavailable", err: errors.New("connection down"), want: http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := setupRouter(fakePinger{err: tc.err}, &MockURLRepository{}, nil, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d", w.Code, tc.want)
			}
		})
	}
}
