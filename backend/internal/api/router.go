package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/api/handler"
	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	"github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/mailer"
)

// SetupRouter initializes and configures the Gin engine with middlewares and routes.
func SetupRouter(pool *pgxpool.Pool, emailSender mailer.EmailSender, frontendBaseURL string) *gin.Engine {
	r := gin.New()

	// Global middlewares
	r.Use(middleware.RequestID())
	r.Use(middleware.GlobalRecovery())

	// Health check endpoints (dual registration for container orchestrators and Nginx /api/* proxy compatibility)
	healthHandler := handler.Health(pool)
	r.GET("/health", healthHandler)
	r.GET("/api/v1/health", healthHandler)

	// Instantiate sqlc queries generator
	var q *dbgen.Queries
	if pool != nil {
		q = dbgen.New(pool)
	}

	apiV1 := r.Group("/api/v1")
	{
		// Public Auth routes
		authPublic := apiV1.Group("/auth")
		{
			authPublic.POST("/login", handler.Login(pool, q))
			authPublic.POST("/forgot-password", handler.ForgotPassword(q, emailSender, frontendBaseURL))
			authPublic.POST("/reset-password", handler.ResetPassword(q))
		}

		// Authenticated Auth routes (requires valid session cookie)
		authProtected := apiV1.Group("/auth")
		if q != nil {
			authProtected.Use(middleware.Authenticate(q))
		}
		{
			authProtected.POST("/logout", handler.Logout(pool, q))
			authProtected.GET("/me", handler.Me(q))
			authProtected.PATCH("/me/password", handler.ChangePassword(q))
		}

		// Admin routes (requires valid session cookie + admin role)
		adminProtected := apiV1.Group("/admin")
		if q != nil {
			adminProtected.Use(middleware.Authenticate(q))
			adminProtected.Use(middleware.RequireRole("admin"))
		}
		{
			adminProtected.POST("/users", handler.CreateUser(pool, q, emailSender, frontendBaseURL))
		}
	}

	return r
}
