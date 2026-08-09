package db_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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
	emptyDir := t.TempDir()

	err := db.RunMigrations(emptyDir, "postgres://user:pass@localhost:5432/dbname?sslmode=disable")
	require.NoError(t, err, "expected no error when migrations folder is empty (should skip safely)")
}

func TestRunMigrations_RealPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

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

	// 1. Test running migrations from backend/migrations containing 4 auth tables
	migrationsPath := "../../migrations"
	err = db.RunMigrations(migrationsPath, connStr)
	require.NoError(t, err, "RunMigrations on domain migrations folder failed")

	// 2. Connect to database to verify table structures & constraints
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	// Verify users table and nullable password_hash constraint
	var userID int
	err = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Test User", "test@example.com", nil).Scan(&userID)
	require.NoError(t, err, "failed to insert user with null password_hash")
	assert.Greater(t, userID, 0)

	// Verify users.email UNIQUE constraint (SQLSTATE 23505)
	_, err = pool.Exec(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3)",
		"Duplicate User", "test@example.com", nil)
	require.Error(t, err)
	var pgErr *pgconn.PgError
	require.True(t, errors.As(err, &pgErr))
	assert.Equal(t, "23505", pgErr.Code, "expected unique_violation SQLSTATE 23505 for duplicate email")

	// Verify user_roles composite PK & CHECK constraint (role IN ('petugas', 'dokter', 'admin'))
	_, err = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", userID, "admin")
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", userID, "admin")
	require.Error(t, err)
	require.True(t, errors.As(err, &pgErr))
	assert.Equal(t, "23505", pgErr.Code, "expected unique_violation SQLSTATE 23505 for duplicate user_role PK")

	_, err = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", userID, "invalid_role")
	require.Error(t, err)
	require.True(t, errors.As(err, &pgErr))
	assert.Equal(t, "23514", pgErr.Code, "expected check_violation SQLSTATE 23514 for invalid role")

	// Verify user_roles FK constraint to users (SQLSTATE 23503)
	_, err = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", 99999, "dokter")
	require.Error(t, err)
	require.True(t, errors.As(err, &pgErr))
	assert.Equal(t, "23503", pgErr.Code, "expected foreign_key_violation SQLSTATE 23503 for non-existent user_id")

	// Verify sessions table FK constraint to users
	now := time.Now()
	_, err = pool.Exec(ctx, "INSERT INTO sessions (id_hash, user_id, created_at, expires_at, absolute_expires_at) VALUES ($1, $2, $3, $4, $5)",
		"hash123", userID, now, now.Add(time.Hour), now.Add(24*time.Hour))
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "INSERT INTO sessions (id_hash, user_id, created_at, expires_at, absolute_expires_at) VALUES ($1, $2, $3, $4, $5)",
		"hash456", 99999, now, now.Add(time.Hour), now.Add(24*time.Hour))
	require.Error(t, err)
	require.True(t, errors.As(err, &pgErr))
	assert.Equal(t, "23503", pgErr.Code, "expected foreign_key_violation SQLSTATE 23503 for sessions user_id")

	// Verify password_tokens table FK & CHECK constraint (type IN ('invite', 'reset'))
	_, err = pool.Exec(ctx, "INSERT INTO password_tokens (token_hash, user_id, type, expires_at) VALUES ($1, $2, $3, $4)",
		"tokenhash123", userID, "invite", now.Add(7*24*time.Hour))
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "INSERT INTO password_tokens (token_hash, user_id, type, expires_at) VALUES ($1, $2, $3, $4)",
		"tokenhash456", userID, "invalid_type", now.Add(time.Hour))
	require.Error(t, err)
	require.True(t, errors.As(err, &pgErr))
	assert.Equal(t, "23514", pgErr.Code, "expected check_violation SQLSTATE 23514 for invalid password_token type")
}
