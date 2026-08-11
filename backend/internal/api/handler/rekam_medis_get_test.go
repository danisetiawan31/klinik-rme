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

func TestGetRekamMedis_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_rm_get_db"),
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
	dokterCookie, dokterUser := createKlinikAntrianTestUser(t, ctx, pool, q, "dr.get@test.com", []string{"dokter"})
	petugasCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "petugas.get@test.com", []string{"petugas"})
	adminCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "admin.get@test.com", []string{"admin"})

	// Setup klinik & pasien
	var klinikID int32
	err = pool.QueryRow(ctx, `
		INSERT INTO klinik (nama, jam_buka, jam_tutup)
		VALUES ('Klinik GET Handler', '08:00', '23:59')
		RETURNING id
	`).Scan(&klinikID)
	require.NoError(t, err)

	pasienWithRM, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nama:         "Pasien Has History",
		TanggalLahir: pgtype.Date{Time: time.Date(1985, 5, 20, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "L",
		Alamat:       "Jl GET Handler 1",
		NoTelp:       "0811111111",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)

	pasienNoRM, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nama:         "Pasien No History",
		TanggalLahir: pgtype.Date{Time: time.Date(1995, 10, 10, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "P",
		Alamat:       "Jl GET Handler 2",
		NoTelp:       "0822222222",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)

	todayDate := pgtype.Date{Time: time.Now(), Valid: true}
	router := api.SetupRouter(pool, nil, "http://localhost:3000")

	// 1. GET /kunjungan/:id/rekam-medis [dokter]
	t.Run("GET_Kunjungan_RekamMedis_Returns_Latest_Leaf_After_Multiple_Addendums", func(t *testing.T) {
		// Create kunjungan
		kunjungan, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasienWithRM.ID,
			KlinikID:         klinikID,
			DokterID:         pgtype.Int4{Int32: dokterUser.ID, Valid: true},
			TanggalKunjungan: todayDate,
			NomorAntrian:     1,
			IsPriority:       false,
			Status:           "dipanggil",
		})
		require.NoError(t, err)

		// Step 1: Create initial RM (Level 0)
		icd0 := "K29.7"
		body0 := handler.CreateRekamMedisRequest{
			Keluhan:          "Keluhan Level 0 (Awal)",
			HasilPemeriksaan: "Hasil Level 0",
			Diagnosis:        []handler.CreateDiagnosisItemRequest{{KodeIcd: &icd0, Deskripsi: "Gastritis Level 0"}},
			Tindakan:         &[]handler.CreateTindakanItemRequest{{Jenis: "tindakan", Deskripsi: "Antasida 0"}},
		}
		json0, _ := json.Marshal(body0)
		req0, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), bytes.NewBuffer(json0))
		req0.Header.Set("Content-Type", "application/json")
		req0.AddCookie(dokterCookie)
		rec0 := httptest.NewRecorder()
		router.ServeHTTP(rec0, req0)
		require.Equal(t, http.StatusCreated, rec0.Code)

		var resp0 handler.RekamMedisResponse
		_ = json.Unmarshal(rec0.Body.Bytes(), &resp0)

		// Step 2: Addendum 1 (Level 1)
		k1 := "Keluhan Level 1 (Addendum 1)"
		addBody1 := handler.CreateAddendumRequest{
			AlasanAddendum: "Addendum 1",
			Keluhan:        &k1,
		}
		json1, _ := json.Marshal(addBody1)
		req1, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", resp0.ID), bytes.NewBuffer(json1))
		req1.Header.Set("Content-Type", "application/json")
		req1.AddCookie(dokterCookie)
		rec1 := httptest.NewRecorder()
		router.ServeHTTP(rec1, req1)
		require.Equal(t, http.StatusCreated, rec1.Code)

		var resp1 handler.AddendumResponse
		_ = json.Unmarshal(rec1.Body.Bytes(), &resp1)

		// Step 3: Addendum 2 (Level 2 - Final Leaf)
		k2 := "Keluhan Level 2 (Addendum 2 - Final)"
		addBody2 := handler.CreateAddendumRequest{
			AlasanAddendum: "Addendum 2",
			Keluhan:        &k2,
		}
		json2, _ := json.Marshal(addBody2)
		req2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", resp1.ID), bytes.NewBuffer(json2))
		req2.Header.Set("Content-Type", "application/json")
		req2.AddCookie(dokterCookie)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)
		require.Equal(t, http.StatusCreated, rec2.Code)

		var resp2 handler.AddendumResponse
		_ = json.Unmarshal(rec2.Body.Bytes(), &resp2)

		// NOW TEST GET /api/v1/kunjungan/:id/rekam-medis
		reqGet, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), nil)
		reqGet.AddCookie(dokterCookie)
		recGet := httptest.NewRecorder()
		router.ServeHTTP(recGet, reqGet)

		require.Equal(t, http.StatusOK, recGet.Code)

		var getResp handler.GetRekamMedisResponse
		err = json.Unmarshal(recGet.Body.Bytes(), &getResp)
		require.NoError(t, err)

		// ASSERT returned RM is the leaf (Level 2)
		assert.Equal(t, resp2.ID, getResp.ID, "GET kunjungan/id/rekam-medis MUST return the final leaf ID")
		assert.Equal(t, "Keluhan Level 2 (Addendum 2 - Final)", getResp.Keluhan)
		assert.True(t, getResp.IsAddendum, "isAddendum MUST be true for addendum leaf")
	})

	t.Run("GET_Kunjungan_Without_RekamMedis_Returns_404", func(t *testing.T) {
		kunjunganEmpty, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasienWithRM.ID,
			KlinikID:         klinikID,
			DokterID:         pgtype.Int4{Int32: dokterUser.ID, Valid: true},
			TanggalKunjungan: todayDate,
			NomorAntrian:     2,
			IsPriority:       false,
			Status:           "menunggu",
		})
		require.NoError(t, err)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjunganEmpty.ID), nil)
		req.AddCookie(dokterCookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("GET_Kunjungan_RekamMedis_Role_Petugas_Or_Admin_Returns_403", func(t *testing.T) {
		kunjungan, _ := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasienWithRM.ID,
			KlinikID:         klinikID,
			DokterID:         pgtype.Int4{Int32: dokterUser.ID, Valid: true},
			TanggalKunjungan: todayDate,
			NomorAntrian:     3,
			Status:           "menunggu",
		})

		reqP, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), nil)
		reqP.AddCookie(petugasCookie)
		recP := httptest.NewRecorder()
		router.ServeHTTP(recP, reqP)
		assert.Equal(t, http.StatusForbidden, recP.Code)

		reqA, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", kunjungan.ID), nil)
		reqA.AddCookie(adminCookie)
		recA := httptest.NewRecorder()
		router.ServeHTTP(recA, reqA)
		assert.Equal(t, http.StatusForbidden, recA.Code)
	})

	// 2. GET /pasien/:id/riwayat [dokter]
	t.Run("GET_Pasien_Riwayat_Filters_Only_Visits_With_RM_And_Returns_Leafs", func(t *testing.T) {
		// Pasien P_Multi
		pasienMulti, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nama:         "Pasien Multi Visits",
			TanggalLahir: pgtype.Date{Time: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L",
			Alamat:       "Jl Multi",
			NoTelp:       "0833333333",
			ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)

		// Kunjungan 1 (Yesterday) -> Has RM + Addendum
		t1 := pgtype.Date{Time: time.Now().AddDate(0, 0, -1), Valid: true}
		k1, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasienMulti.ID,
			KlinikID:         klinikID,
			DokterID:         pgtype.Int4{Int32: dokterUser.ID, Valid: true},
			TanggalKunjungan: t1,
			NomorAntrian:     1,
			Status:           "selesai",
		})
		require.NoError(t, err)

		icd1 := "J00"
		body1 := handler.CreateRekamMedisRequest{
			Keluhan:          "Keluhan K1 Awal",
			HasilPemeriksaan: "Hasil K1",
			Diagnosis:        []handler.CreateDiagnosisItemRequest{{KodeIcd: &icd1, Deskripsi: "Flu K1"}},
			Tindakan:         &[]handler.CreateTindakanItemRequest{},
		}
		j1, _ := json.Marshal(body1)
		r1, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", k1.ID), bytes.NewBuffer(j1))
		r1.Header.Set("Content-Type", "application/json")
		r1.AddCookie(dokterCookie)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, r1)
		require.Equal(t, http.StatusCreated, w1.Code)
		var rmResp1 handler.RekamMedisResponse
		_ = json.Unmarshal(w1.Body.Bytes(), &rmResp1)

		// Addendum to K1
		k1Addend := "Keluhan K1 Addendum Leaf"
		addBodyK1 := handler.CreateAddendumRequest{
			AlasanAddendum: "Revisi K1",
			Keluhan:        &k1Addend,
		}
		j1Add, _ := json.Marshal(addBodyK1)
		r1Add, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", rmResp1.ID), bytes.NewBuffer(j1Add))
		r1Add.Header.Set("Content-Type", "application/json")
		r1Add.AddCookie(dokterCookie)
		w1Add := httptest.NewRecorder()
		router.ServeHTTP(w1Add, r1Add)
		require.Equal(t, http.StatusCreated, w1Add.Code)

		// Kunjungan 2 (Today) -> NO RM
		t2 := todayDate
		_, err = q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasienMulti.ID,
			KlinikID:         klinikID,
			DokterID:         pgtype.Int4{Int32: dokterUser.ID, Valid: true},
			TanggalKunjungan: t2,
			NomorAntrian:     2,
			Status:           "menunggu",
		})
		require.NoError(t, err)

		// NOW TEST GET /api/v1/pasien/:id/riwayat
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/pasien/%d/riwayat", pasienMulti.ID), nil)
		req.AddCookie(dokterCookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var riwayat []handler.RiwayatKunjunganItem
		err = json.Unmarshal(rec.Body.Bytes(), &riwayat)
		require.NoError(t, err)

		// Assert length is EXACTLY 1 (Kunjungan 2 without RM is filtered out)
		require.Len(t, riwayat, 1, "Only visits with rekam medis should be included in history")
		assert.Equal(t, k1.ID, riwayat[0].KunjunganID)
		assert.Equal(t, "Keluhan K1 Addendum Leaf", riwayat[0].RekamMedis.Keluhan, "Rekam medis returned for K1 must be the leaf addendum")
	})

	t.Run("GET_Pasien_Riwayat_No_Clinical_History_Returns_Empty_Array_200", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/pasien/%d/riwayat", pasienNoRM.ID), nil)
		req.AddCookie(dokterCookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var riwayat []handler.RiwayatKunjunganItem
		err := json.Unmarshal(rec.Body.Bytes(), &riwayat)
		require.NoError(t, err)

		assert.NotNil(t, riwayat, "Empty history should return non-nil JSON array []")
		assert.Equal(t, 0, len(riwayat))
	})

	t.Run("GET_Pasien_Riwayat_Non_Existent_Pasien_Returns_404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/pasien/999999/riwayat", nil)
		req.AddCookie(dokterCookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("GET_Pasien_Riwayat_Role_Petugas_Or_Admin_Returns_403", func(t *testing.T) {
		reqP, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/pasien/%d/riwayat", pasienWithRM.ID), nil)
		reqP.AddCookie(petugasCookie)
		recP := httptest.NewRecorder()
		router.ServeHTTP(recP, reqP)
		assert.Equal(t, http.StatusForbidden, recP.Code)

		reqA, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/pasien/%d/riwayat", pasienWithRM.ID), nil)
		reqA.AddCookie(adminCookie)
		recA := httptest.NewRecorder()
		router.ServeHTTP(recA, reqA)
		assert.Equal(t, http.StatusForbidden, recA.Code)
	})
}
