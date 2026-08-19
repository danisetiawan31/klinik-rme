package db_test

import (
	"context"
	"errors"
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

	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func TestKlinikAntrianDomain_RealPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_klinik_antrian_db"),
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

	// 1. Verifikasi Migration & DB Constraints
	t.Run("Migration & DB Constraints Verification", func(t *testing.T) {
		// Table klinik exists and has display_token_hash column (added in migration 000015)
		var columnExists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'klinik' AND column_name = 'display_token_hash'
			)`).Scan(&columnExists)
		require.NoError(t, err)
		assert.True(t, columnExists, "klinik table must have display_token_hash column")

		// CHECK status kunjungan (must be waiting/dipanggil/selesai/tidak_hadir)
		// Inserting with invalid status must fail
		_, err = pool.Exec(ctx, `
			INSERT INTO kunjungan (pasien_id, klinik_id, tanggal_kunjungan, nomor_antrian, status)
			VALUES (1, 1, CURRENT_DATE, 1, 'invalid_status')`)
		require.Error(t, err, "inserting invalid kunjungan status must fail DB constraint")
	})

	// 2. Verifikasi Idempotensi Seed Klinik
	t.Run("Seed Klinik Idempotency Verification", func(t *testing.T) {
		cfg := &config.Config{
			KlinikNama:     "Klinik Sehat Jaya",
			KlinikJamBuka:  "08:00",
			KlinikJamTutup: "17:00",
		}

		// Initial empty table -> 1 row inserted
		err := bootstrap.SeedKlinik(ctx, pool, q, cfg)
		require.NoError(t, err)

		count, err := q.CountKlinik(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		klinik, err := q.GetSingleKlinik(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Klinik Sehat Jaya", klinik.Nama)

		// Modify klinik name manually via SQL
		_, err = pool.Exec(ctx, `UPDATE klinik SET nama = 'Klinik Modified Manual' WHERE id = $1`, klinik.ID)
		require.NoError(t, err)

		// Re-run seed function -> must skip total, name remains modified
		err = bootstrap.SeedKlinik(ctx, pool, q, cfg)
		require.NoError(t, err)

		klinikAfter, err := q.GetSingleKlinik(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Klinik Modified Manual", klinikAfter.Nama, "idempotent seed must NOT overwrite existing klinik data")
	})

	// 3. Verifikasi UpsertQueueCounter
	t.Run("UpsertQueueCounter Verification", func(t *testing.T) {
		klinik, err := q.GetSingleKlinik(ctx)
		require.NoError(t, err)

		today := pgtype.Date{Time: time.Now(), Valid: true}
		tomorrow := pgtype.Date{Time: time.Now().Add(24 * time.Hour), Valid: true}

		// First call today -> 1
		num1, err := q.UpsertQueueCounter(ctx, dbgen.UpsertQueueCounterParams{
			KlinikID: klinik.ID,
			Tanggal:  today,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(1), num1)

		// Second call today -> 2
		num2, err := q.UpsertQueueCounter(ctx, dbgen.UpsertQueueCounterParams{
			KlinikID: klinik.ID,
			Tanggal:  today,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(2), num2)

		// Call tomorrow -> 1 (independent per date)
		numTomorrow, err := q.UpsertQueueCounter(ctx, dbgen.UpsertQueueCounterParams{
			KlinikID: klinik.ID,
			Tanggal:  tomorrow,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(1), numTomorrow)
	})

	// 4. Verifikasi ClaimNextKunjungan (Order: Priority DESC, SkipCount ASC, NomorAntrian ASC)
	t.Run("ClaimNextKunjungan Order Verification", func(t *testing.T) {
		klinik, err := q.GetSingleKlinik(ctx)
		require.NoError(t, err)

		// Setup 1 test doctor user
		dokter, err := q.CreateUser(ctx, dbgen.CreateUserParams{
			Nama:  "Dokter Test",
			Email: "dokter.test@klinik.local",
		})
		require.NoError(t, err)

		// Setup 4 test pasien
		p1, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nama:         "Pasien A",
			TanggalLahir: pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L",
			Alamat:       "Jl A",
			NoTelp:       "0812",
			ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)

		p2, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nama:         "Pasien B (Priority)",
			TanggalLahir: pgtype.Date{Time: time.Date(1991, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "P",
			Alamat:       "Jl B",
			NoTelp:       "0813",
			ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)

		p3, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nama:         "Pasien C",
			TanggalLahir: pgtype.Date{Time: time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L",
			Alamat:       "Jl C",
			NoTelp:       "0814",
			ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)

		p4, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nama:         "Pasien D (Skipped)",
			TanggalLahir: pgtype.Date{Time: time.Date(1993, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "P",
			Alamat:       "Jl D",
			NoTelp:       "0815",
			ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)

		testDate := pgtype.Date{Time: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC), Valid: true}

		// Insert Kunjungan rows for testDate:
		// Row A: nomor=1, priority=false, skip=0
		_, err = q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         p1.ID,
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
			NomorAntrian:     1,
			IsPriority:       false,
			SkipCount:        0,
			Status:           "menunggu",
		})
		require.NoError(t, err)

		// Row B: nomor=2, priority=true, skip=0
		_, err = q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         p2.ID,
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
			NomorAntrian:     2,
			IsPriority:       true,
			PriorityReason:   pgtype.Text{String: "Lansia Gawat", Valid: true},
			SkipCount:        0,
			Status:           "menunggu",
		})
		require.NoError(t, err)

		// Row C: nomor=3, priority=false, skip=0
		_, err = q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         p3.ID,
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
			NomorAntrian:     3,
			IsPriority:       false,
			SkipCount:        0,
			Status:           "menunggu",
		})
		require.NoError(t, err)

		// Row D: nomor=4, priority=false, skip=1
		_, err = q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         p4.ID,
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
			NomorAntrian:     4,
			IsPriority:       false,
			SkipCount:        1,
			Status:           "menunggu",
		})
		require.NoError(t, err)

		// Claim 1: Should get Row B (Priority=true)
		claimed1, err := q.ClaimNextKunjungan(ctx, dbgen.ClaimNextKunjunganParams{
			DokterID:         pgtype.Int4{Int32: dokter.ID, Valid: true},
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
		})
		require.NoError(t, err)
		assert.Equal(t, p2.ID, claimed1.PasienID)
		assert.Equal(t, int32(2), claimed1.NomorAntrian)
		assert.Equal(t, "dipanggil", claimed1.Status)
		assert.Equal(t, dokter.ID, claimed1.DokterID.Int32)

		// Claim 2: Should get Row A (Nomor 1, skip=0)
		claimed2, err := q.ClaimNextKunjungan(ctx, dbgen.ClaimNextKunjunganParams{
			DokterID:         pgtype.Int4{Int32: dokter.ID, Valid: true},
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
		})
		require.NoError(t, err)
		assert.Equal(t, p1.ID, claimed2.PasienID)
		assert.Equal(t, int32(1), claimed2.NomorAntrian)

		// Claim 3: Should get Row C (Nomor 3, skip=0) before Row D (skip=1)
		claimed3, err := q.ClaimNextKunjungan(ctx, dbgen.ClaimNextKunjunganParams{
			DokterID:         pgtype.Int4{Int32: dokter.ID, Valid: true},
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
		})
		require.NoError(t, err)
		assert.Equal(t, p3.ID, claimed3.PasienID)
		assert.Equal(t, int32(3), claimed3.NomorAntrian)

		// Claim 4: Should get Row D (skip=1)
		claimed4, err := q.ClaimNextKunjungan(ctx, dbgen.ClaimNextKunjunganParams{
			DokterID:         pgtype.Int4{Int32: dokter.ID, Valid: true},
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
		})
		require.NoError(t, err)
		assert.Equal(t, p4.ID, claimed4.PasienID)
		assert.Equal(t, int32(4), claimed4.NomorAntrian)

		// Claim 5: Queue empty -> returns ErrNoRows
		_, err = q.ClaimNextKunjungan(ctx, dbgen.ClaimNextKunjunganParams{
			DokterID:         pgtype.Int4{Int32: dokter.ID, Valid: true},
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, pgx.ErrNoRows), "claiming empty queue must return pgx.ErrNoRows")
	})

	// 5. Verifikasi UpdateKunjunganSkip
	t.Run("UpdateKunjunganSkip Verification", func(t *testing.T) {
		klinik, err := q.GetSingleKlinik(ctx)
		require.NoError(t, err)

		pasien, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nama:         "Pasien Skip Test",
			TanggalLahir: pgtype.Date{Time: time.Date(1995, 5, 5, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L",
			Alamat:       "Jl Skip",
			NoTelp:       "0816",
			ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)

		testDate := pgtype.Date{Time: time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC), Valid: true}

		kunjungan, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasien.ID,
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
			NomorAntrian:     10,
			Status:           "dipanggil",
			SkipCount:        0,
		})
		require.NoError(t, err)

		// Skip from 'dipanggil' -> success (status becomes 'menunggu', skip_count becomes 1)
		skipped, err := q.UpdateKunjunganSkip(ctx, kunjungan.ID)
		require.NoError(t, err)
		assert.Equal(t, "menunggu", skipped.Status)
		assert.Equal(t, int32(1), skipped.SkipCount)

		// Skip from 'menunggu' (not 'dipanggil') -> 0 rows affected (returns pgx.ErrNoRows)
		_, err = q.UpdateKunjunganSkip(ctx, kunjungan.ID)
		require.Error(t, err)
		assert.True(t, errors.Is(err, pgx.ErrNoRows), "skipping non-dipanggil kunjungan must return pgx.ErrNoRows")
	})

	// 6. Verifikasi UpdateKunjunganTidakHadir
	t.Run("UpdateKunjunganTidakHadir Verification", func(t *testing.T) {
		klinik, err := q.GetSingleKlinik(ctx)
		require.NoError(t, err)

		pasien, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nama:         "Pasien Absence Test",
			TanggalLahir: pgtype.Date{Time: time.Date(1996, 6, 6, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "P",
			Alamat:       "Jl Absence",
			NoTelp:       "0817",
			ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)

		testDate := pgtype.Date{Time: time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), Valid: true}

		// 1. From status 'menunggu' -> success
		k1, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasien.ID,
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
			NomorAntrian:     20,
			Status:           "menunggu",
		})
		require.NoError(t, err)

		res1, err := q.UpdateKunjunganTidakHadir(ctx, k1.ID)
		require.NoError(t, err)
		assert.Equal(t, "tidak_hadir", res1.Status)

		// 2. From status 'dipanggil' -> success
		k2, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasien.ID,
			KlinikID:         klinik.ID,
			TanggalKunjungan: testDate,
			NomorAntrian:     21,
			Status:           "dipanggil",
		})
		require.NoError(t, err)

		res2, err := q.UpdateKunjunganTidakHadir(ctx, k2.ID)
		require.NoError(t, err)
		assert.Equal(t, "tidak_hadir", res2.Status)

		// 3. From status 'tidak_hadir' (already final) -> 0 rows affected (returns pgx.ErrNoRows)
		_, err = q.UpdateKunjunganTidakHadir(ctx, res2.ID)
		require.Error(t, err)
		assert.True(t, errors.Is(err, pgx.ErrNoRows), "marking already final kunjungan as tidak_hadir must return pgx.ErrNoRows")
	})
}
