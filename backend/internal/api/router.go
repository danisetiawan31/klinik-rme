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
		authProtected.Use(middleware.Authenticate(q))
		{
			authProtected.POST("/logout", handler.Logout(pool, q))
			authProtected.GET("/me", handler.Me(q))
			authProtected.PATCH("/me/password", handler.ChangePassword(q))
		}

		// Admin routes (requires valid session cookie + admin role)
		adminProtected := apiV1.Group("/admin")
		adminProtected.Use(middleware.Authenticate(q))
		adminProtected.Use(middleware.RequireRole("admin"))
		{
			adminProtected.POST("/users", handler.CreateUser(pool, q, emailSender, frontendBaseURL))
		}

		// Pasien routes (requires valid session cookie + appropriate role per endpoint)
		pasienGroup := apiV1.Group("/pasien")
		pasienGroup.Use(middleware.Authenticate(q))
		{
			pasienGroup.POST("", middleware.RequireRole("petugas", "admin"), handler.CreatePasien(pool, q))
			pasienGroup.GET("/search", middleware.RequireRole("petugas", "dokter", "admin"), handler.SearchPasien(q))
			pasienGroup.GET("/:id", middleware.RequireRole("petugas", "dokter", "admin"), handler.GetPasienByID(q))
			pasienGroup.PATCH("/:id", middleware.RequireRole("petugas", "admin"), handler.UpdatePasien(pool, q))
		}

		// Klinik & Antrian routes (requires valid session cookie + appropriate role per endpoint)
		klinikAntrianH := handler.NewKlinikAntrianHandler(pool)

		klinikGroup := apiV1.Group("/klinik")
		klinikGroup.Use(middleware.Authenticate(q))
		{
			klinikGroup.GET("/:id", middleware.RequireRole("petugas", "dokter", "admin"), klinikAntrianH.GetKlinikByID)
			klinikGroup.GET("/:id/antrian", middleware.RequireRole("petugas", "dokter", "admin"), klinikAntrianH.GetAntrianKlinik)
			klinikGroup.POST("/:id/panggil-berikutnya", middleware.RequireRole("dokter"), klinikAntrianH.PanggilBerikutnya)
		}

		kunjunganGroup := apiV1.Group("/kunjungan")
		kunjunganGroup.Use(middleware.Authenticate(q))
		{
			kunjunganGroup.POST("", middleware.RequireRole("petugas", "admin"), klinikAntrianH.CreateKunjungan)
			kunjunganGroup.GET("/:id", middleware.RequireRole("petugas", "dokter", "admin"), klinikAntrianH.GetKunjunganByID)
			kunjunganGroup.POST("/:id/lewati", middleware.RequireRole("dokter"), klinikAntrianH.Lewati)
			kunjunganGroup.POST("/:id/tidak-hadir", middleware.RequireRole("dokter", "admin"), klinikAntrianH.TidakHadir)
		}
	}

	return r
}
