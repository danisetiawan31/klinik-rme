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

func TestRekamMedisDomain_RealPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_rekam_medis_db"),
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

	// Setup prerequisites: User (Dokter), Pasien, Klinik, Kunjungan
	dokter, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		Nama:  "Dr. Rekam Medis",
		Email: "dr.rm@test.com",
	})
	require.NoError(t, err)

	pasien, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nama:         "Pasien RM Test",
		TanggalLahir: pgtype.Date{Time: time.Date(1995, 5, 5, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "L",
		Alamat:       "Jl RM Test",
		NoTelp:       "0812345678",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)

	var klinikID int32
	err = pool.QueryRow(ctx, `
		INSERT INTO klinik (nama, jam_buka, jam_tutup)
		VALUES ('Klinik RM Test', '08:00', '17:00')
		RETURNING id
	`).Scan(&klinikID)
	require.NoError(t, err)

	kunjungan, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
		PasienID:         pasien.ID,
		KlinikID:         klinikID,
		DokterID:         pgtype.Int4{Int32: dokter.ID, Valid: true},
		TanggalKunjungan: pgtype.Date{Time: time.Now(), Valid: true},
		NomorAntrian:     1,
		IsPriority:       false,
		Status:           "dipanggil",
	})
	require.NoError(t, err)

	// 1. Audit Log aksi='addendum' Extended Constraint Verification
	t.Run("Audit_Log_Aksi_Extended_Constraint_Verification", func(t *testing.T) {
		// aksi='addendum' MUST succeed
		_, err := pool.Exec(ctx, `
			INSERT INTO audit_log (tabel_target, record_id, actor_user_id, aksi, after_data, hash_entry, previous_hash, created_at)
			VALUES ('rekam_medis', 1, $1, 'addendum', '{}'::jsonb, 'hash1', 'genesis', NOW())
		`, dokter.ID)
		assert.NoError(t, err, "aksi='addendum' must be accepted by extended check constraint")

		// aksi='invalid' MUST fail
		_, err = pool.Exec(ctx, `
			INSERT INTO audit_log (tabel_target, record_id, actor_user_id, aksi, after_data, hash_entry, previous_hash, created_at)
			VALUES ('rekam_medis', 1, $1, 'invalid_action', '{}'::jsonb, 'hash2', 'hash1', NOW())
		`, dokter.ID)
		assert.Error(t, err, "aksi='invalid_action' must violate check constraint")
	})

	// 2. uq_rekam_medis_root_per_kunjungan Partial Unique Index Verification
	t.Run("uq_rekam_medis_root_per_kunjungan_Verification", func(t *testing.T) {
		// Root record 1 for kunjungan -> SUCCESS
		root1, err := q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjungan.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Pusing dan mual",
			HasilPemeriksaan: "Tensi 120/80",
			IsAddendum:       false,
			AddendumOf:       pgtype.Int4{Valid: false},
			AlasanAddendum:   pgtype.Text{Valid: false},
		})
		require.NoError(t, err)

		// Root record 2 for SAME kunjungan (addendum_of NULL) -> MUST FAIL (23505)
		_, err = q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjungan.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Batuk darah",
			HasilPemeriksaan: "Tensi 140/90",
			IsAddendum:       false,
			AddendumOf:       pgtype.Int4{Valid: false},
			AlasanAddendum:   pgtype.Text{Valid: false},
		})
		assert.Error(t, err, "Second root record for same kunjungan must fail due to uq_rekam_medis_root_per_kunjungan")

		// Addendum record for SAME kunjungan (addendum_of NOT NULL) -> MUST SUCCEED (Root + Addendum no conflict)
		addendum1, err := q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjungan.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Pusing dan mual hebat",
			HasilPemeriksaan: "Tensi 125/85",
			IsAddendum:       true,
			AddendumOf:       pgtype.Int4{Int32: root1.ID, Valid: true},
			AlasanAddendum:   pgtype.Text{String: "Koreksi keluhan", Valid: true},
		})
		require.NoError(t, err)
		assert.Equal(t, root1.ID, addendum1.AddendumOf.Int32)
	})

	// 3. uq_addendum_of_active Partial Unique Index Verification
	t.Run("uq_addendum_of_active_Verification", func(t *testing.T) {
		// Create a separate kunjungan & root record
		kunjunganB, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasien.ID,
			KlinikID:         klinikID,
			DokterID:         pgtype.Int4{Int32: dokter.ID, Valid: true},
			TanggalKunjungan: pgtype.Date{Time: time.Now(), Valid: true},
			NomorAntrian:     2,
			IsPriority:       false,
			Status:           "dipanggil",
		})
		require.NoError(t, err)

		rootB, err := q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjunganB.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Demam tinggi",
			HasilPemeriksaan: "Suhu 38.5C",
			IsAddendum:       false,
			AddendumOf:       pgtype.Int4{Valid: false},
			AlasanAddendum:   pgtype.Text{Valid: false},
		})
		require.NoError(t, err)

		// First addendum to rootB -> SUCCESS
		addB1, err := q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjunganB.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Demam tinggi dan menggigil",
			HasilPemeriksaan: "Suhu 39C",
			IsAddendum:       true,
			AddendumOf:       pgtype.Int4{Int32: rootB.ID, Valid: true},
			AlasanAddendum:   pgtype.Text{String: "Update suhu", Valid: true},
		})
		require.NoError(t, err)

		// Second active addendum to SAME rootB (addB1 active) -> MUST FAIL (23505)
		_, err = q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjunganB.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Konflik addendum",
			HasilPemeriksaan: "Suhu 40C",
			IsAddendum:       true,
			AddendumOf:       pgtype.Int4{Int32: rootB.ID, Valid: true},
			AlasanAddendum:   pgtype.Text{String: "Konflik", Valid: true},
		})
		assert.Error(t, err, "Concurrent/second addendum to same parent must fail due to uq_addendum_of_active")

		// Soft delete addB1
		_, err = pool.Exec(ctx, `UPDATE rekam_medis SET deleted_at = NOW() WHERE id = $1`, addB1.ID)
		require.NoError(t, err)

		// Re-attempt addendum to rootB after soft-delete -> MUST SUCCEED (soft-delete opens slot)
		addB2, err := q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjunganB.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Demam setelah soft delete",
			HasilPemeriksaan: "Suhu 37.5C",
			IsAddendum:       true,
			AddendumOf:       pgtype.Int4{Int32: rootB.ID, Valid: true},
			AlasanAddendum:   pgtype.Text{String: "Koreksi setelah soft delete", Valid: true},
		})
		require.NoError(t, err)
		assert.Equal(t, rootB.ID, addB2.AddendumOf.Int32)
	})

	// 4. GetLeafRekamMedisByKunjunganID Chain 3-Level Traversal
	t.Run("GetLeafRekamMedisByKunjunganID_Chain_Traversal", func(t *testing.T) {
		kunjunganC, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasien.ID,
			KlinikID:         klinikID,
			DokterID:         pgtype.Int4{Int32: dokter.ID, Valid: true},
			TanggalKunjungan: pgtype.Date{Time: time.Now(), Valid: true},
			NomorAntrian:     3,
			IsPriority:       false,
			Status:           "dipanggil",
		})
		require.NoError(t, err)

		// Root (Level 1)
		rootC, err := q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjunganC.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Level 1 Root",
			HasilPemeriksaan: "Obs 1",
			IsAddendum:       false,
			AddendumOf:       pgtype.Int4{Valid: false},
			AlasanAddendum:   pgtype.Text{Valid: false},
		})
		require.NoError(t, err)

		// Addendum 1 (Level 2)
		addC1, err := q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjunganC.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Level 2 Addendum 1",
			HasilPemeriksaan: "Obs 2",
			IsAddendum:       true,
			AddendumOf:       pgtype.Int4{Int32: rootC.ID, Valid: true},
			AlasanAddendum:   pgtype.Text{String: "Addend 1", Valid: true},
		})
		require.NoError(t, err)

		// Addendum 2 (Level 3 - Leaf)
		addC2, err := q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjunganC.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Level 3 Addendum 2 (Leaf)",
			HasilPemeriksaan: "Obs 3 Final",
			IsAddendum:       true,
			AddendumOf:       pgtype.Int4{Int32: addC1.ID, Valid: true},
			AlasanAddendum:   pgtype.Text{String: "Addend 2", Valid: true},
		})
		require.NoError(t, err)

		// Query Leaf for kunjunganC
		leaf, err := q.GetLeafRekamMedisByKunjunganID(ctx, kunjunganC.ID)
		require.NoError(t, err)
		assert.Equal(t, addC2.ID, leaf.ID, "GetLeafRekamMedisByKunjunganID MUST return the latest leaf (addC2)")
		assert.Equal(t, "Level 3 Addendum 2 (Leaf)", leaf.Keluhan)
	})

	// 5. Diagnosis & Tindakan Basic Correctness & Check Constraint
	t.Run("Diagnosis_And_Tindakan_Queries_And_Constraints", func(t *testing.T) {
		kunjunganD, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasien.ID,
			KlinikID:         klinikID,
			DokterID:         pgtype.Int4{Int32: dokter.ID, Valid: true},
			TanggalKunjungan: pgtype.Date{Time: time.Now(), Valid: true},
			NomorAntrian:     4,
			IsPriority:       false,
			Status:           "dipanggil",
		})
		require.NoError(t, err)

		rmD, err := q.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      kunjunganD.ID,
			DokterID:         dokter.ID,
			Keluhan:          "Nyeri Perut",
			HasilPemeriksaan: "Palpasi positif",
			IsAddendum:       false,
			AddendumOf:       pgtype.Int4{Valid: false},
			AlasanAddendum:   pgtype.Text{Valid: false},
		})
		require.NoError(t, err)

		// Insert Diagnosis
		diag, err := q.InsertDiagnosis(ctx, dbgen.InsertDiagnosisParams{
			RekamMedisID: rmD.ID,
			KodeIcd:      pgtype.Text{String: "K29.7", Valid: true},
			Deskripsi:    "Gastritis, unspecified",
		})
		require.NoError(t, err)
		assert.Equal(t, "K29.7", diag.KodeIcd.String)

		diags, err := q.GetDiagnosisByRekamMedisID(ctx, rmD.ID)
		require.NoError(t, err)
		require.Len(t, diags, 1)
		assert.Equal(t, diag.ID, diags[0].ID)

		// Insert Tindakan (valid 'tindakan')
		tin1, err := q.InsertTindakan(ctx, dbgen.InsertTindakanParams{
			RekamMedisID: rmD.ID,
			Jenis:        "tindakan",
			Deskripsi:    "Injeksi Antasida",
		})
		require.NoError(t, err)

		// Insert Tindakan (valid 'resep')
		tin2, err := q.InsertTindakan(ctx, dbgen.InsertTindakanParams{
			RekamMedisID: rmD.ID,
			Jenis:        "resep",
			Deskripsi:    "Omeprazole 20mg 2x1",
		})
		require.NoError(t, err)

		tins, err := q.GetTindakanByRekamMedisID(ctx, rmD.ID)
		require.NoError(t, err)
		require.Len(t, tins, 2)
		assert.Equal(t, tin1.ID, tins[0].ID)
		assert.Equal(t, tin2.ID, tins[1].ID)

		// Insert Tindakan (invalid jenis) -> MUST FAIL (CHECK constraint)
		_, err = q.InsertTindakan(ctx, dbgen.InsertTindakanParams{
			RekamMedisID: rmD.ID,
			Jenis:        "invalid_type",
			Deskripsi:    "Operasi",
		})
		assert.Error(t, err, "Invalid jenis tindakan must fail CHECK constraint")
	})

	// 6. ListLeafRekamMedisWithKunjunganByPasienID Query Verification
	t.Run("ListLeafRekamMedisWithKunjunganByPasienID_Verification", func(t *testing.T) {
		history, err := q.ListLeafRekamMedisWithKunjunganByPasienID(ctx, pasien.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(history), 4, "Should list all leaf rekam medis for the patient")
	})
}
