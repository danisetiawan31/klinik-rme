package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/danisetiawan31/klinik-rme/internal/api"
	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	"github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/mailer"
)

func main() {
	// Setup context that listens for terminate signals (SIGINT / SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	log.Println("Starting Klinik RME Backend Scaffolding...")

	// 0. Best-effort loading of .env file for local development
	if err := godotenv.Load(); err != nil {
		log.Println("Info: .env file not found, relying on system environment variables")
	}

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Fatal: Configuration failed to load: %v", err)
	}

	log.Printf("Configuration loaded successfully (TZ: %s, DB Host: %s:%d, HTTP Port: %s)",
		cfg.TZ, cfg.DBHost, cfg.DBPort, cfg.HTTPPort)

	// 2. Run database migrations
	log.Println("Running database migrations...")
	if err := db.RunMigrations("migrations", cfg.DSN()); err != nil {
		log.Fatalf("Fatal: Database migration failed: %v", err)
	}

	// 3. Initialize DB pool & sqlc queries
	pool, err := db.NewPool(ctx, cfg.DSN())
	if err != nil {
		log.Fatalf("Fatal: Failed to initialize DB pool: %v", err)
	}
	defer func() {
		log.Println("Closing database pool...")
		pool.Close()
	}()

	q := dbgen.New(pool)

	// 4. Run admin bootstrap seed check
	log.Println("Running admin bootstrap check...")
	if err := bootstrap.SeedAdmin(ctx, pool, q, cfg.SeedAdminEmail); err != nil {
		log.Fatalf("Fatal: Admin bootstrap failed: %v", err)
	}

	// 5. Run klinik bootstrap seed check
	log.Println("Running klinik bootstrap check...")
	if err := bootstrap.SeedKlinik(ctx, pool, q, cfg); err != nil {
		log.Fatalf("Fatal: Klinik bootstrap failed: %v", err)
	}

	// 6. Initialize Mailer service
	resendMailer := mailer.NewResendMailer(cfg.ResendAPIKey, cfg.ResendFromEmail)

	// 6. Setup Gin router & HTTP Server
	router := api.SetupRouter(pool, resendMailer, cfg.FrontendBaseURL)
	serverAddr := fmt.Sprintf(":%s", cfg.HTTPPort)
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	// Start HTTP server in a separate goroutine
	go func() {
		log.Printf("HTTP server listening on %s...", serverAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Fatal: HTTP server failed to listen: %v", err)
		}
	}()

	// 7. Wait for termination signal
	<-ctx.Done()
	log.Println("Termination signal received. Shutting down gracefully...")

	// Graceful shutdown HTTP server with a 5-second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during HTTP server shutdown: %v", err)
	} else {
		log.Println("HTTP server stopped cleanly.")
	}
}
