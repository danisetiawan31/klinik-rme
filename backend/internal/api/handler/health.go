package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
)

type HealthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

// Health returns a Gin handler for GET /health endpoint.
func Health(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if pool == nil {
			middleware.RespondError(
				c,
				http.StatusServiceUnavailable,
				"DB_UNAVAILABLE",
				"Layanan database tidak tersedia",
				nil,
			)
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			middleware.RespondError(
				c,
				http.StatusServiceUnavailable,
				"DB_UNAVAILABLE",
				"Layanan database tidak dapat dijangkau",
				err,
			)
			return
		}

		c.JSON(http.StatusOK, HealthResponse{
			Status: "ok",
			DB:     "ok",
		})
	}
}
