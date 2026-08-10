package db_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/db"
)

func TestAuditTrailMigrations_RealPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_audit_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	defer func() {
		_ = postgresContainer.Terminate(ctx)
	}()

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	err = db.RunMigrations("../../migrations", connStr)
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	// 1. Verifikasi Genesis Seed & Hash Calculation
	t.Run("Genesis Seed Row Verification", func(t *testing.T) {
		var id int
		var lastHash string
		err := pool.QueryRow(ctx, "SELECT id, last_hash FROM audit_log_tail WHERE id = 1").Scan(&id, &lastHash)
		require.NoError(t, err, "genesis seed row must exist in audit_log_tail")

		assert.Equal(t, 1, id)

		// Calculate manual SHA256 of 'klinik-rme-genesis' in Go
		genesisInput := "klinik-rme-genesis"
		expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(genesisInput)))

		assert.Equal(t, expectedHash, lastHash, "genesis seed last_hash in DB must match manual SHA256 calculation of 'klinik-rme-genesis'")
		assert.Equal(t, "f5ebe6fb00b0cf82d9b6c624cd93d9ceb6f6647b48ab7c0bad7915f62caffb8f", lastHash)
	})

	// 2. Verifikasi Constraints (Singleton audit_log_tail.id=1 & audit_log.aksi)
	t.Run("DB Constraints Verification", func(t *testing.T) {
		// audit_log_tail singleton constraint (id = 1) -> inserting id=2 must fail
		_, err := pool.Exec(ctx, "INSERT INTO audit_log_tail (id, last_hash) VALUES (2, 'dummy_hash')")
		require.Error(t, err, "inserting id != 1 in audit_log_tail must violate check constraint")

		// Create a test user for FK requirement
		var actorUserID int
		err = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ('Actor', 'actor@test.com', 'hash') RETURNING id").Scan(&actorUserID)
		require.NoError(t, err)

		// audit_log.aksi CHECK IN ('create', 'update') -> inserting aksi='invalid' must fail
		_, err = pool.Exec(ctx, `INSERT INTO audit_log (tabel_target, record_id, actor_user_id, aksi, after_data, hash_entry, previous_hash, created_at) 
			VALUES ('users', 1, $1, 'invalid_action', '{}', 'h1', 'p1', now())`, actorUserID)
		require.Error(t, err, "inserting invalid aksi in audit_log must violate check constraint")

		// audit_log.actor_user_id FK constraint -> non-existent user must fail
		_, err = pool.Exec(ctx, `INSERT INTO audit_log (tabel_target, record_id, actor_user_id, aksi, after_data, hash_entry, previous_hash, created_at) 
			VALUES ('users', 1, 99999, 'create', '{}', 'h1', 'p1', now())`)
		require.Error(t, err, "inserting non-existent actor_user_id in audit_log must violate FK constraint")
	})

	// 3. Verifikasi Tamper Prevention Triggers (BEFORE UPDATE & BEFORE DELETE)
	t.Run("Tamper Prevention Triggers Verification", func(t *testing.T) {
		var actorUserID int
		err := pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ('Audit User', 'audit.user@test.com', 'hash') RETURNING id").Scan(&actorUserID)
		require.NoError(t, err)

		var auditLogID int
		err = pool.QueryRow(ctx, `INSERT INTO audit_log (tabel_target, record_id, actor_user_id, aksi, after_data, hash_entry, previous_hash, created_at) 
			VALUES ('pasien', 101, $1, 'create', '{"nama":"Budi"}', 'hash123', 'prev123', now()) RETURNING id`, actorUserID).Scan(&auditLogID)
		require.NoError(t, err, "inserting valid row into audit_log must succeed")

		// Attempt UPDATE on audit_log -> MUST fail with trigger exception
		_, err = pool.Exec(ctx, "UPDATE audit_log SET tabel_target = 'modified' WHERE id = $1", auditLogID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audit_log is append-only")

		// Attempt DELETE on audit_log -> MUST fail with trigger exception
		_, err = pool.Exec(ctx, "DELETE FROM audit_log WHERE id = $1", auditLogID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "audit_log is append-only")
	})
}
