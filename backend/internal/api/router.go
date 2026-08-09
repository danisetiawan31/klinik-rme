package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/api/handler"
	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
)

// SetupRouter initializes and configures the Gin engine with middlewares and routes.
func SetupRouter(pool *pgxpool.Pool) *gin.Engine {
	r := gin.New()

	// Global middlewares
	r.Use(middleware.RequestID())
	r.Use(middleware.GlobalRecovery())

	// Public routes
	r.GET("/health", handler.Health(pool))

	return r
}
