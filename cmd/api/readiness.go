package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type databasePinger interface {
	PingContext(context.Context) error
}

func readinessHandler(db databasePinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 250*time.Millisecond)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
