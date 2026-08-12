package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func TestRealtimeMigrationAndQueries_RealPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_realtime_db"),
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

	pool, err := db.NewPool(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	q := dbgen.New(pool)

	t.Run("1._Migration_000015_Column_Verification", func(t *testing.T) {
		var isNullable string
		var dataType string

		err := pool.QueryRow(ctx, `
			SELECT is_nullable, data_type 
			FROM information_schema.columns 
			WHERE table_name = 'klinik' AND column_name = 'display_token_hash'
		`).Scan(&isNullable, &dataType)

		require.NoError(t, err, "Column display_token_hash harus ada di tabel klinik")
		assert.Equal(t, "YES", isNullable, "Column display_token_hash harus nullable")
		assert.Equal(t, "text", dataType, "Column display_token_hash harus bertipe text")
	})

	t.Run("2._Insert_Klinik_Starts_With_NULL_DisplayTokenHash", func(t *testing.T) {
		var klinikID int32
		err := pool.QueryRow(ctx, `
			INSERT INTO klinik (nama, jam_buka, jam_tutup)
			VALUES ('Klinik Realtime Test', '08:00', '23:59')
			RETURNING id
		`).Scan(&klinikID)
		require.NoError(t, err)

		hashVal, err := q.GetKlinikDisplayTokenHash(ctx, klinikID)
		require.NoError(t, err)
		assert.False(t, hashVal.Valid, "Display token hash harus NULL secara default untuk klinik baru")
	})

	t.Run("3._UpdateKlinikDisplayTokenHash_&_GetKlinikDisplayTokenHash_Correctness", func(t *testing.T) {
		var klinikID int32
		err := pool.QueryRow(ctx, `
			INSERT INTO klinik (nama, jam_buka, jam_tutup)
			VALUES ('Klinik Token Hash Test', '08:00', '23:59')
			RETURNING id
		`).Scan(&klinikID)
		require.NoError(t, err)

		mockHash := "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678" // 64-char SHA256 hex

		// Update display token hash
		updatedKlinik, err := q.UpdateKlinikDisplayTokenHash(ctx, dbgen.UpdateKlinikDisplayTokenHashParams{
			DisplayTokenHash: pgtype.Text{String: mockHash, Valid: true},
			ID:               klinikID,
		})
		require.NoError(t, err)
		assert.Equal(t, klinikID, updatedKlinik.ID)
		assert.True(t, updatedKlinik.DisplayTokenHash.Valid)
		assert.Equal(t, mockHash, updatedKlinik.DisplayTokenHash.String)

		// Get display token hash terpisah
		fetchedHash, err := q.GetKlinikDisplayTokenHash(ctx, klinikID)
		require.NoError(t, err)
		assert.True(t, fetchedHash.Valid)
		assert.Equal(t, mockHash, fetchedHash.String)
	})
}
