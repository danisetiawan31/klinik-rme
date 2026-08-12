package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
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

func TestCreateAddendum_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_rm_addendum_db"),
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

	// Setup users: Dokter 1, Dokter 2, Petugas, Admin
	dokter1Cookie, dokter1User := createKlinikAntrianTestUser(t, ctx, pool, q, "dr1.addendum@test.com", []string{"dokter"})
	dokter2Cookie, dokter2User := createKlinikAntrianTestUser(t, ctx, pool, q, "dr2.addendum@test.com", []string{"dokter"})
	petugasCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "petugas.addendum@test.com", []string{"petugas"})
	adminCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "admin.addendum@test.com", []string{"admin"})

	// Setup klinik & pasien
	var klinikID int32
	err = pool.QueryRow(ctx, `
		INSERT INTO klinik (nama, jam_buka, jam_tutup)
		VALUES ('Klinik Addendum Handler', '08:00', '23:59')
		RETURNING id
	`).Scan(&klinikID)
	require.NoError(t, err)

	pasien, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nama:         "Pasien Addendum Handler",
		TanggalLahir: pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "P",
		Alamat:       "Jl Addendum Handler",
		NoTelp:       "0888888888",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)

	todayDate := pgtype.Date{Time: time.Now(), Valid: true}
	router := api.SetupRouter(pool, nil, nil, "http://localhost:3000")

	// Helper to setup a parent rekam medis via HTTP
	createParentRM := func(noAntrian int32) (dbgen.Kunjungan, handler.RekamMedisResponse) {
		k, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasien.ID,
			KlinikID:         klinikID,
			DokterID:         pgtype.Int4{Int32: dokter1User.ID, Valid: true},
			TanggalKunjungan: todayDate,
			NomorAntrian:     noAntrian,
			IsPriority:       false,
			Status:           "dipanggil",
		})
		require.NoError(t, err)

		icd := "K29.7"
		tindakanArr := []handler.CreateTindakanItemRequest{
			{Jenis: "tindakan", Deskripsi: "Injeksi Antasida"},
			{Jenis: "resep", Deskripsi: "Omeprazole 20mg"},
		}
		body := handler.CreateRekamMedisRequest{
			Keluhan:          "Pusing dan mual awal",
			HasilPemeriksaan: "Tensi 120/80 mmHg",
			Diagnosis: []handler.CreateDiagnosisItemRequest{
				{KodeIcd: &icd, Deskripsi: "Gastritis Awal"},
			},
			Tindakan: &tindakanArr,
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", k.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokter1Cookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)

		var rmResp handler.RekamMedisResponse
		err = json.Unmarshal(rec.Body.Bytes(), &rmResp)
		require.NoError(t, err)

		return k, rmResp
	}

	// 1. Sukses dengan SEBAGIAN field dikirim (cuma keluhan) -> Carry-Over Verification
	t.Run("Sukses_Partial_Fields_Carry_Over_Verification", func(t *testing.T) {
		_, parentRM := createParentRM(1)

		newKeluhan := "Pusing hebat berkepanjangan (Updated)"
		addendumBody := handler.CreateAddendumRequest{
			AlasanAddendum: "Perkembangan gejala pasien",
			Keluhan:        &newKeluhan,
			// hasilPemeriksaan, diagnosis, tindakan nil -> Carry over!
		}
		jsonBody, _ := json.Marshal(addendumBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", parentRM.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokter2Cookie) // Dokter lain yang addend!

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var resp handler.AddendumResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, parentRM.ID, resp.AddendumOf)
		assert.Equal(t, "Pusing hebat berkepanjangan (Updated)", resp.Keluhan)
		// Assert carry-over values strictly match parent:
		assert.Equal(t, parentRM.HasilPemeriksaan, resp.HasilPemeriksaan)
		require.Len(t, resp.Diagnosis, len(parentRM.Diagnosis))
		assert.Equal(t, parentRM.Diagnosis[0].Deskripsi, resp.Diagnosis[0].Deskripsi)
		require.Len(t, resp.Tindakan, len(parentRM.Tindakan))
		assert.Equal(t, parentRM.Tindakan[0].Deskripsi, resp.Tindakan[0].Deskripsi)

		// Audit Log Verification:
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

		assert.Equal(t, int(dokter2User.ID), auditActorID, "Audit actor must be dokter 2")
		assert.Equal(t, "addendum", auditAksi, "Audit aksi MUST be 'addendum'")
		assert.NotNil(t, beforeDataJSON)
		assert.NotNil(t, afterDataJSON)

		var beforeSnap, afterSnap map[string]interface{}
		_ = json.Unmarshal(beforeDataJSON, &beforeSnap)
		_ = json.Unmarshal(afterDataJSON, &afterSnap)

		assert.Equal(t, parentRM.Keluhan, beforeSnap["keluhan"])
		assert.Equal(t, "Pusing hebat berkepanjangan (Updated)", afterSnap["keluhan"])
	})

	// 2. Diagnosis dikirim eksplisit [] TAPI field lain nil -> 400 Bad Request
	t.Run("Explicit_Empty_Diagnosis_Returns_400_Bad_Request", func(t *testing.T) {
		_, parentRM := createParentRM(2)

		emptyDiag := []handler.CreateDiagnosisItemRequest{}
		addendumBody := handler.CreateAddendumRequest{
			AlasanAddendum: "Mencoba kosongkan diagnosis",
			Diagnosis:      &emptyDiag, // Explicit empty slice -> 0 merged diagnosis
		}
		jsonBody, _ := json.Marshal(addendumBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", parentRM.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokter1Cookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)

		// Verify 0 addendum rows inserted for parentRM
		var addendumCount int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM rekam_medis WHERE addendum_of = $1`, parentRM.ID).Scan(&addendumCount)
		require.NoError(t, err)
		assert.Equal(t, 0, addendumCount)
	})

	// 3. Sukses dengan Diagnosis dikirim eksplisit BEDA dari parent -> Override Verification
	t.Run("Explicit_Diagnosis_Override_Verification", func(t *testing.T) {
		_, parentRM := createParentRM(3)

		newIcd := "K29.5"
		newDiags := []handler.CreateDiagnosisItemRequest{
			{KodeIcd: &newIcd, Deskripsi: "Chronic gastritis, unspecified (Overridden)"},
		}
		addendumBody := handler.CreateAddendumRequest{
			AlasanAddendum: "Revisi hasil lab",
			Diagnosis:      &newDiags,
		}
		jsonBody, _ := json.Marshal(addendumBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", parentRM.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokter1Cookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)

		var resp handler.AddendumResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		require.Len(t, resp.Diagnosis, 1)
		assert.Equal(t, "Chronic gastritis, unspecified (Overridden)", resp.Diagnosis[0].Deskripsi)
		assert.NotEqual(t, parentRM.Diagnosis[0].Deskripsi, resp.Diagnosis[0].Deskripsi)
	})

	// 4. Parent :id Tidak Ada / Soft-Deleted -> 404 Not Found
	t.Run("Non_Existent_Parent_Returns_404", func(t *testing.T) {
		addendumBody := handler.CreateAddendumRequest{
			AlasanAddendum: "Parent invalid",
		}
		jsonBody, _ := json.Marshal(addendumBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/rekam-medis/999999/addendum", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokter1Cookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	// 5. Concurrency Test: 2 Goroutines Addendum Simultaneously to SAME Parent -> 1x 201, 1x 409
	t.Run("Concurrency_Simultaneous_Addendum_1_Success_1_Conflict", func(t *testing.T) {
		_, parentRM := createParentRM(5)

		var wg sync.WaitGroup
		startGate := make(chan struct{})

		statusCodes := make([]int, 2)
		cookies := []*http.Cookie{dokter1Cookie, dokter2Cookie}

		wg.Add(2)
		for i := 0; i < 2; i++ {
			idx := i
			go func() {
				defer wg.Done()
				<-startGate

				kVal := fmt.Sprintf("Keluhan dokter %d", idx+1)
				body := handler.CreateAddendumRequest{
					AlasanAddendum: fmt.Sprintf("Addendum dari dokter %d", idx+1),
					Keluhan:        &kVal,
				}
				jsonBody, _ := json.Marshal(body)

				req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", parentRM.ID), bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(cookies[idx])

				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				statusCodes[idx] = rec.Code
			}()
		}

		close(startGate)
		wg.Wait()

		// Assert EXACTLY 1 success (201 Created) and EXACTLY 1 conflict (409 Conflict)
		has201 := (statusCodes[0] == http.StatusCreated || statusCodes[1] == http.StatusCreated)
		has409 := (statusCodes[0] == http.StatusConflict || statusCodes[1] == http.StatusConflict)

		assert.True(t, has201, "Exactly one addendum request must succeed with 201")
		assert.True(t, has409, "Exactly one addendum request must fail with 409 Conflict due to uq_addendum_of_active")

		// Assert EXACTLY 1 addendum row created in DB for parentRM
		var addendumCount int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM rekam_medis WHERE addendum_of = $1`, parentRM.ID).Scan(&addendumCount)
		require.NoError(t, err)
		assert.Equal(t, 1, addendumCount, "Only 1 addendum row must be created in DB")
	})

	// 6. Role Petugas / Admin -> 403 Forbidden
	t.Run("Role_Petugas_Or_Admin_Returns_403_Forbidden", func(t *testing.T) {
		_, parentRM := createParentRM(6)

		body := handler.CreateAddendumRequest{
			AlasanAddendum: "Role test",
		}
		jsonBody, _ := json.Marshal(body)

		reqPetugas, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", parentRM.ID), bytes.NewBuffer(jsonBody))
		reqPetugas.Header.Set("Content-Type", "application/json")
		reqPetugas.AddCookie(petugasCookie)
		recPetugas := httptest.NewRecorder()
		router.ServeHTTP(recPetugas, reqPetugas)
		assert.Equal(t, http.StatusForbidden, recPetugas.Code)

		reqAdmin, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", parentRM.ID), bytes.NewBuffer(jsonBody))
		reqAdmin.Header.Set("Content-Type", "application/json")
		reqAdmin.AddCookie(adminCookie)
		recAdmin := httptest.NewRecorder()
		router.ServeHTTP(recAdmin, reqAdmin)
		assert.Equal(t, http.StatusForbidden, recAdmin.Code)
	})

	// 7. AlasanAddendum Kosong -> 400 Bad Request
	t.Run("AlasanAddendum_Kosong_Returns_400", func(t *testing.T) {
		_, parentRM := createParentRM(7)

		body := handler.CreateAddendumRequest{
			AlasanAddendum: "   ", // Kosong
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/rekam-medis/%d/addendum", parentRM.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokter1Cookie)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
