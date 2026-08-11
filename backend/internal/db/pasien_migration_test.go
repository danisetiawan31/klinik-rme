package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func TestPasienDomain_RealPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_pasien_db"),
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

	q := dbgen.New(pool)

	// 1. Verifikasi Constraint DB & Permisif Duplikasi NIK
	t.Run("DB Constraints & Duplicate NIK Verification", func(t *testing.T) {
		// jenis_kelamin CHECK constraint (must be 'L' or 'P') -> 'X' must fail
		_, err := pool.Exec(ctx, `INSERT INTO pasien (nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at) 
			VALUES ('Invalid Gender', '1990-01-01', 'X', 'Alamat', '08123', NOW())`)
		require.Error(t, err, "jenis_kelamin != L/P must violate CHECK constraint")

		// consent_at NOT NULL constraint -> NULL consent_at must fail
		_, err = pool.Exec(ctx, `INSERT INTO pasien (nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at) 
			VALUES ('No Consent', '1990-01-01', 'L', 'Alamat', '08123', NULL)`)
		require.Error(t, err, "NULL consent_at must violate NOT NULL constraint")

		// Default version = 1 when omitted in raw SQL insert
		var ver int32
		err = pool.QueryRow(ctx, `INSERT INTO pasien (nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at) 
			VALUES ('Default Ver', '1990-01-01', 'L', 'Alamat', '08123', NOW()) RETURNING version`).Scan(&ver)
		require.NoError(t, err)
		assert.Equal(t, int32(1), ver, "default version must be 1")

		// CRITICAL REQUIREMENT: Duplicate NIK is ALLOWED (No UNIQUE constraint)
		duplicateNIK := "3171012345670001"

		p1, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: duplicateNIK, Valid: true},
			Nama:         "Pasien NIK 1",
			TanggalLahir: pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L",
			Alamat:       "Alamat 1",
			NoTelp:       "08123456789",
			ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err, "first insert with NIK must succeed")

		p2, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: duplicateNIK, Valid: true},
			Nama:         "Pasien NIK 2 (Duplicate)",
			TanggalLahir: pgtype.Date{Time: time.Date(1992, 2, 2, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "P",
			Alamat:       "Alamat 2",
			NoTelp:       "08987654321",
			ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err, "second insert with DUPLICATE NIK MUST SUCCEED (no UNIQUE constraint)")

		assert.NotEqual(t, p1.ID, p2.ID)
		assert.Equal(t, duplicateNIK, p1.Nik.String)
		assert.Equal(t, duplicateNIK, p2.Nik.String)
	})

	// 2. Verifikasi Query SearchPasien, Pagination & Soft Delete Filter
	t.Run("SearchPasien & Soft Delete Verification", func(t *testing.T) {
		// Clean table for search test
		_, _ = pool.Exec(ctx, "TRUNCATE pasien RESTART IDENTITY CASCADE")

		now := time.Now()
		dob := pgtype.Date{Time: time.Date(1985, 5, 20, 0, 0, 0, 0, time.UTC), Valid: true}

		// Insert test set
		_, _ = q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "111111", Valid: true},
			Nama:         "Budi Setyo",
			TanggalLahir: dob, JenisKelamin: "L", Alamat: "Jakarta", NoTelp: "08111", ConsentAt: pgtype.Timestamptz{Time: now, Valid: true},
		})
		p2, _ := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "222222", Valid: true},
			Nama:         "Budi Santoso",
			TanggalLahir: dob, JenisKelamin: "L", Alamat: "Bandung", NoTelp: "08222", ConsentAt: pgtype.Timestamptz{Time: now, Valid: true},
		})
		_, _ = q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "333333", Valid: true},
			Nama:         "Siti Rahma",
			TanggalLahir: dob, JenisKelamin: "P", Alamat: "Surabaya", NoTelp: "08333", ConsentAt: pgtype.Timestamptz{Time: now, Valid: true},
		})

		// 1. Search by NIK exact match
		resNIK, err := q.SearchPasien(ctx, dbgen.SearchPasienParams{
			Limit:  10,
			Offset: 0,
			Nik:    pgtype.Text{String: "222222", Valid: true},
			Nama:   pgtype.Text{Valid: false},
		})
		require.NoError(t, err)
		require.Len(t, resNIK, 1)
		assert.Equal(t, "Budi Santoso", resNIK[0].Nama)

		// 2. Search by Nama partial match
		resNama, err := q.SearchPasien(ctx, dbgen.SearchPasienParams{
			Limit:  10,
			Offset: 0,
			Nik:    pgtype.Text{Valid: false},
			Nama:   pgtype.Text{String: "Budi", Valid: true},
		})
		require.NoError(t, err)
		require.Len(t, resNama, 2)

		// 3. Combination NIK & Nama (AND logic)
		resCombined, err := q.SearchPasien(ctx, dbgen.SearchPasienParams{
			Limit:  10,
			Offset: 0,
			Nik:    pgtype.Text{String: "222222", Valid: true},
			Nama:   pgtype.Text{String: "Budi", Valid: true},
		})
		require.NoError(t, err)
		require.Len(t, resCombined, 1, "AND combination must narrow down result to exactly 1 match")
		assert.Equal(t, p2.ID, resCombined[0].ID)

		// 4. Pagination (Limit 1, Offset 0 vs Offset 1)
		page1, err := q.SearchPasien(ctx, dbgen.SearchPasienParams{
			Limit:  1,
			Offset: 0,
			Nik:    pgtype.Text{Valid: false},
			Nama:   pgtype.Text{Valid: false},
		})
		require.NoError(t, err)
		require.Len(t, page1, 1)

		page2, err := q.SearchPasien(ctx, dbgen.SearchPasienParams{
			Limit:  1,
			Offset: 1,
			Nik:    pgtype.Text{Valid: false},
			Nama:   pgtype.Text{Valid: false},
		})
		require.NoError(t, err)
		require.Len(t, page2, 1)
		assert.NotEqual(t, page1[0].ID, page2[0].ID, "offset 1 must return second record")

		// 5. Soft Delete Filtering Verification
		// Mark p2 as soft-deleted in DB
		_, err = pool.Exec(ctx, "UPDATE pasien SET deleted_at = NOW() WHERE id = $1", p2.ID)
		require.NoError(t, err)

		// GetPasienByID for p2 must return ErrNoRows (404)
		_, err = q.GetPasienByID(ctx, p2.ID)
		assert.ErrorIs(t, err, pgx.ErrNoRows, "GetPasienByID must fail with ErrNoRows when deleted_at is set")

		// GetPasienByIDIncludingDeleted for p2 must succeed
		p2Deleted, err := q.GetPasienByIDIncludingDeleted(ctx, p2.ID)
		require.NoError(t, err)
		assert.Equal(t, p2.ID, p2Deleted.ID)
		assert.True(t, p2Deleted.DeletedAt.Valid, "deleted_at must be set")

		// SearchPasien must NOT include p2
		resAfterDelete, err := q.SearchPasien(ctx, dbgen.SearchPasienParams{
			Limit:  10,
			Offset: 0,
			Nik:    pgtype.Text{Valid: false},
			Nama:   pgtype.Text{Valid: false},
		})
		require.NoError(t, err)
		assert.Len(t, resAfterDelete, 2, "soft deleted patient must not appear in search results")
	})

	// 3. Verifikasi UpdatePasienOptimistic
	t.Run("UpdatePasienOptimistic Verification", func(t *testing.T) {
		now := time.Now()
		p, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "999999", Valid: true},
			Nama:         "Pasien Optimistic",
			TanggalLahir: pgtype.Date{Time: time.Date(1995, 10, 10, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L",
			Alamat:       "Alamat Lama",
			NoTelp:       "08999",
			ConsentAt:    pgtype.Timestamptz{Time: now, Valid: true},
		})
		require.NoError(t, err)
		assert.Equal(t, int32(1), p.Version)

		// 1. Version matches (version = 1) -> Update succeeds, version increments to 2
		pUpdated, err := q.UpdatePasienOptimistic(ctx, dbgen.UpdatePasienOptimisticParams{
			ID:           p.ID,
			Version:      1,
			Nik:          pgtype.Text{Valid: false}, // unset/unchanged
			Nama:         pgtype.Text{String: "Pasien Optimistic Updated", Valid: true},
			TanggalLahir: pgtype.Date{Valid: false},
			JenisKelamin: pgtype.Text{Valid: false},
			Alamat:       pgtype.Text{String: "Alamat Baru", Valid: true},
			NoTelp:       pgtype.Text{Valid: false},
		})
		require.NoError(t, err, "update with matching version 1 must succeed")
		assert.Equal(t, int32(2), pUpdated.Version, "version must increment to 2")
		assert.Equal(t, "Pasien Optimistic Updated", pUpdated.Nama)
		assert.Equal(t, "Alamat Baru", pUpdated.Alamat)
		assert.Equal(t, "999999", pUpdated.Nik.String, "unset field (Nik) must remain unchanged")

		// 2. Version mismatch (try version = 1 again when DB has version = 2) -> 0 rows / ErrNoRows
		_, err = q.UpdatePasienOptimistic(ctx, dbgen.UpdatePasienOptimisticParams{
			ID:      p.ID,
			Version: 1, // stale version!
			Nama:    pgtype.Text{String: "Stale Edit", Valid: true},
		})
		assert.ErrorIs(t, err, pgx.ErrNoRows, "update with stale version must return ErrNoRows (0 rows returned)")

		// 3. Non-existent ID -> 0 rows / ErrNoRows
		_, err = q.UpdatePasienOptimistic(ctx, dbgen.UpdatePasienOptimisticParams{
			ID:      999999,
			Version: 1,
			Nama:    pgtype.Text{String: "Non-existent", Valid: true},
		})
		assert.ErrorIs(t, err, pgx.ErrNoRows, "update with non-existent ID must return ErrNoRows")
	})
}
