package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/api"
	"github.com/danisetiawan31/klinik-rme/internal/api/handler"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func TestCreateRekamMedisAwal_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_rm_handler_db"),
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

	err = db.RunMigrations("../../../migrations", connStr)
	require.NoError(t, err)

	pool, err := db.NewPool(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	q := dbgen.New(pool)

	// Setup users: Dokter, Petugas, Admin
	dokterCookie, dokterUser := createKlinikAntrianTestUser(t, ctx, pool, q, "dokter.rm@test.com", []string{"dokter"})
	petugasCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "petugas.rm@test.com", []string{"petugas"})
	adminCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "admin.rm@test.com", []string{"admin"})

	// Setup klinik & pasien
	var klinikID int32
	err = pool.QueryRow(ctx, `
		INSERT INTO klinik (nama, jam_buka, jam_tutup)
		VALUES ('Klinik RM Handler', '08:00', '23:59')
		RETURNING id
	`).Scan(&klinikID)
	require.NoError(t, err)

	pasien, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nama:         "Pasien RM Handler",
		TanggalLahir: pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "L",
		Alamat:       "Jl RM Handler",
		NoTelp:       "089999999",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)

	todayDate := pgtype.Date{Time: time.Now(), Valid: true}

	router := api.SetupRouter(pool, nil, "http://localhost:3000")

	// Helper to create a test kunjungan
	createTestKunjungan := func(noAntrian int32) dbgen.Kunjungan {
		k, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasien.ID,
			KlinikID:         klinikID,
			DokterID:         pgtype.Int4{Int32: dokterUser.ID, Valid: true},
			TanggalKunjungan: todayDate,
			NomorAntrian:     noAntrian,
			IsPriority:       false,
			Status:           "dipanggil",
		})
		require.NoError(t, err)
		return k
	}

	// 1. Sukses: 201 Created & DB State Verification
	t.Run("Sukses_201_Created_And_Full_DB_Verification", func(t *testing.T) {
		kunjungan := createTestKunjungan(1)

		icd := "K29.7"
		tindakanArr := []handler.CreateTindakanItemRequest{
			{Jenis: "tindakan", Deskripsi: "Injeksi Antasida"},
			{Jenis: "resep", Deskripsi: "Omeprazole 20mg 2x1"},
		}
		body := handler.CreateRekamMedisRequest{
			Keluhan:          "Pusing dan mual",
			HasilPemeriksaan: "Tensi 120/80 mmHg",
			Diagnosis: []handler.CreateDiagnosisItemRequest{
				{KodeIcd: &icd, Deskripsi: "Gastritis, unspecified"},
			},
			Tindakan: &tindakanArr,
		}

		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokterCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var resp handler.RekamMedisResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Greater(t, resp.ID, int32(0))
		assert.Equal(t, "Pusing dan mual", resp.Keluhan)
		assert.Equal(t, "Tensi 120/80 mmHg", resp.HasilPemeriksaan)
		require.Len(t, resp.Diagnosis, 1)
		assert.Equal(t, "Gastritis, unspecified", resp.Diagnosis[0].Deskripsi)
		require.Len(t, resp.Tindakan, 2)
		assert.Equal(t, "tindakan", resp.Tindakan[0].Jenis)

		// Direct DB Verification:
		// a. kunjungan.status MUST be 'selesai'
		updatedKunjungan, err := q.GetKunjunganByID(ctx, kunjungan.ID)
		require.NoError(t, err)
		assert.Equal(t, "selesai", updatedKunjungan.Status)

		// b. audit_log entry verified
		var auditActorID int
		var auditAksi string
		var beforeDataJSON []byte
		var afterDataJSON []byte
		err = pool.QueryRow(ctx, `
			SELECT actor_user_id, aksi, before_data, after_data
			FROM audit_log
			WHERE tabel_target = 'rekam_medis' AND record_id = $1
		`, resp.ID).Scan(&auditActorID, &auditAksi, &beforeDataJSON, &afterDataJSON)
		require.NoError(t, err)

		assert.Equal(t, int(dokterUser.ID), auditActorID, "Audit actor must be doctor from session")
		assert.Equal(t, "create", auditAksi)
		assert.Nil(t, beforeDataJSON)

		var afterSnap map[string]interface{}
		err = json.Unmarshal(afterDataJSON, &afterSnap)
		require.NoError(t, err)
		assert.Equal(t, "Pusing dan mual", afterSnap["keluhan"])
		assert.NotNil(t, afterSnap["diagnosis"])
		assert.NotNil(t, afterSnap["tindakan"])
	})

	// 2. Diagnosis Kosong / Tidak Dikirim -> 400 Bad Request & Zero Side Effect
	t.Run("Diagnosis_Kosong_Returns_400_And_Zero_Side_Effect", func(t *testing.T) {
		kunjungan := createTestKunjungan(2)

		tindakanArr := []handler.CreateTindakanItemRequest{
			{Jenis: "tindakan", Deskripsi: "Injeksi"},
		}
		body := handler.CreateRekamMedisRequest{
			Keluhan:          "Batuk",
			HasilPemeriksaan: "Ronkhi positif",
			Diagnosis:        []handler.CreateDiagnosisItemRequest{}, // Kosong
			Tindakan:         &tindakanArr,
		}

		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokterCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)

		// Assert 0 rows inserted in rekam_medis & kunjungan.status remains 'dipanggil'
		var rmCount int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM rekam_medis WHERE kunjungan_id = $1`, kunjungan.ID).Scan(&rmCount)
		require.NoError(t, err)
		assert.Equal(t, 0, rmCount, "No rekam_medis row should be created on validation failure")

		updatedKunjungan, err := q.GetKunjunganByID(ctx, kunjungan.ID)
		require.NoError(t, err)
		assert.Equal(t, "dipanggil", updatedKunjungan.Status, "Kunjungan status must remain 'dipanggil'")
	})

	// 3. Tindakan Array Kosong [] (Key Ada) -> 201 Created
	t.Run("Tindakan_Empty_Array_Returns_201_Created", func(t *testing.T) {
		kunjungan := createTestKunjungan(3)

		icd := "J00"
		tindakanArr := []handler.CreateTindakanItemRequest{} // Array kosong
		body := handler.CreateRekamMedisRequest{
			Keluhan:          "Flu ringan",
			HasilPemeriksaan: "Suhu 37C",
			Diagnosis: []handler.CreateDiagnosisItemRequest{
				{KodeIcd: &icd, Deskripsi: "Common cold"},
			},
			Tindakan: &tindakanArr,
		}

		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokterCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var resp handler.RekamMedisResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 0, len(resp.Tindakan))

		var tindakanCount int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM tindakan WHERE rekam_medis_id = $1`, resp.ID).Scan(&tindakanCount)
		require.NoError(t, err)
		assert.Equal(t, 0, tindakanCount)
	})

	// 4. Kunjungan Sudah Punya Root Record -> 409 Conflict
	t.Run("Duplicate_Root_Record_Returns_409_Conflict", func(t *testing.T) {
		kunjungan := createTestKunjungan(4)

		icd := "K29.7"
		tindakanArr := []handler.CreateTindakanItemRequest{}
		body := handler.CreateRekamMedisRequest{
			Keluhan:          "Sakit perut",
			HasilPemeriksaan: "Normal",
			Diagnosis: []handler.CreateDiagnosisItemRequest{
				{KodeIcd: &icd, Deskripsi: "Gastritis"},
			},
			Tindakan: &tindakanArr,
		}

		jsonBody, _ := json.Marshal(body)

		// First Call -> 201
		req1, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), bytes.NewBuffer(jsonBody))
		req1.Header.Set("Content-Type", "application/json")
		req1.AddCookie(dokterCookie)
		rec1 := httptest.NewRecorder()
		router.ServeHTTP(rec1, req1)
		require.Equal(t, http.StatusCreated, rec1.Code)

		// Second Call -> 409 Conflict
		req2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), bytes.NewBuffer(jsonBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.AddCookie(dokterCookie)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)
		assert.Equal(t, http.StatusConflict, rec2.Code)

		// Assert only 1 rekam_medis row exists for this kunjungan
		var rmCount int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM rekam_medis WHERE kunjungan_id = $1`, kunjungan.ID).Scan(&rmCount)
		require.NoError(t, err)
		assert.Equal(t, 1, rmCount)
	})

	// 5. Kunjungan Tidak Ada -> 404 Not Found
	t.Run("Kunjungan_Tidak_Ada_Returns_404", func(t *testing.T) {
		icd := "K29.7"
		tindakanArr := []handler.CreateTindakanItemRequest{}
		body := handler.CreateRekamMedisRequest{
			Keluhan:          "Tidak Ada",
			HasilPemeriksaan: "Tidak Ada",
			Diagnosis: []handler.CreateDiagnosisItemRequest{
				{KodeIcd: &icd, Deskripsi: "Gastritis"},
			},
			Tindakan: &tindakanArr,
		}

		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan/999999/rekam-medis", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokterCookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	// 6. Role Petugas / Admin -> 403 Forbidden
	t.Run("Role_Petugas_Or_Admin_Returns_403_Forbidden", func(t *testing.T) {
		kunjungan := createTestKunjungan(5)

		icd := "K29.7"
		tindakanArr := []handler.CreateTindakanItemRequest{}
		body := handler.CreateRekamMedisRequest{
			Keluhan:          "Role check",
			HasilPemeriksaan: "Role check",
			Diagnosis: []handler.CreateDiagnosisItemRequest{
				{KodeIcd: &icd, Deskripsi: "Gastritis"},
			},
			Tindakan: &tindakanArr,
		}

		jsonBody, _ := json.Marshal(body)

		// Petugas -> 403
		reqPetugas, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), bytes.NewBuffer(jsonBody))
		reqPetugas.Header.Set("Content-Type", "application/json")
		reqPetugas.AddCookie(petugasCookie)
		recPetugas := httptest.NewRecorder()
		router.ServeHTTP(recPetugas, reqPetugas)
		assert.Equal(t, http.StatusForbidden, recPetugas.Code)

		// Admin -> 403
		reqAdmin, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), bytes.NewBuffer(jsonBody))
		reqAdmin.Header.Set("Content-Type", "application/json")
		reqAdmin.AddCookie(adminCookie)
		recAdmin := httptest.NewRecorder()
		router.ServeHTTP(recAdmin, reqAdmin)
		assert.Equal(t, http.StatusForbidden, recAdmin.Code)
	})

	// 7. Tanpa Auth -> 401 Unauthorized
	t.Run("Tanpa_Auth_Returns_401_Unauthorized", func(t *testing.T) {
		kunjungan := createTestKunjungan(6)

		icd := "K29.7"
		tindakanArr := []handler.CreateTindakanItemRequest{}
		body := handler.CreateRekamMedisRequest{
			Keluhan:          "No auth",
			HasilPemeriksaan: "No auth",
			Diagnosis: []handler.CreateDiagnosisItemRequest{
				{KodeIcd: &icd, Deskripsi: "Gastritis"},
			},
			Tindakan: &tindakanArr,
		}

		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
