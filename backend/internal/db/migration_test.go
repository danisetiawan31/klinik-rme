package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/db"
)

func TestRunMigrations_NonExistentPath(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "non_existent_migrations_dir_12345")

	err := db.RunMigrations(nonExistentPath, "postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	require.Error(t, err, "expected error when migrations path does not exist")
	assert.Contains(t, err.Error(), "migrations directory does not exist")
}

func TestRunMigrations_EmptyFolder(t *testing.T) {
	// Folder exists but contains no .sql files
	emptyDir := t.TempDir()

	err := db.RunMigrations(emptyDir, "postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	require.NoError(t, err, "expected no error when migrations folder is empty (should skip safely)")
}

func TestRunMigrations_RealPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Spin up real PostgreSQL container using testcontainers-go
	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err, "failed to start postgres container")
	defer func() {
		err := postgresContainer.Terminate(ctx)
		assert.NoError(t, err, "failed to terminate postgres container")
	}()

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string from postgres container")

	// 1. Test empty root migrations directory (contains no .sql files)
	migrationsPath := "../../migrations"
	err = db.RunMigrations(migrationsPath, connStr)
	require.NoError(t, err, "RunMigrations on empty migrations folder failed")

	// 2. Test directory with valid .sql migration file against real Postgres container
	tempDir := t.TempDir()
	sqlContent := `CREATE TABLE test_schema_table (id SERIAL PRIMARY KEY, name TEXT);`
	sqlFile := filepath.Join(tempDir, "000001_create_test_schema_table.up.sql")
	err = os.WriteFile(sqlFile, []byte(sqlContent), 0644)
	require.NoError(t, err)

	err = db.RunMigrations(tempDir, connStr)
	require.NoError(t, err, "RunMigrations with valid .sql migration file failed against real Postgres container")
}
