package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations executes database migrations from the specified migrations directory
// against the target PostgreSQL database DSN using the golang-migrate library.
func RunMigrations(migrationsPath, dbDSN string) error {
	// 1. Resolve absolute path consistently
	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for migrations (%s): %w", migrationsPath, err)
	}

	// 2. Explicit directory check using absPath
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("migrations directory does not exist: %s", absPath)
		}
		return fmt.Errorf("failed to access migrations directory (%s): %w", absPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("migrations path is not a directory: %s", absPath)
	}

	// 3. Scan for *.sql migration files inside absPath
	sqlPattern := filepath.Join(absPath, "*.sql")
	matches, err := filepath.Glob(sqlPattern)
	if err != nil {
		return fmt.Errorf("failed to scan for migration files in %s: %w", absPath, err)
	}

	if len(matches) == 0 {
		log.Printf("Database migration: no .sql migration files found in directory (%s). Skipping migration.", absPath)
		return nil
	}

	// 4. Construct cross-platform file:// URI for golang-migrate
	// golang-migrate concatenates u.Host + u.Path ("file://D:/..." => "D:" + "/..." = "D:/...")
	slashPath := filepath.ToSlash(absPath)
	sourceURL := fmt.Sprintf("file://%s", slashPath)

	// 5. Initialize and run golang-migrate
	m, err := migrate.New(sourceURL, dbDSN)
	if err != nil {
		return fmt.Errorf("failed to initialize migration runner: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("Warning: closing migration source error: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("Warning: closing migration db error: %v", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Database migration: no changes to apply.")
			return nil
		}
		return fmt.Errorf("failed to execute database migrations: %w", err)
	}

	log.Println("Database migrations applied successfully.")
	return nil
}
