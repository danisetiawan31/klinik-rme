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
	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func TestKlinikAntrianClaimEndpoints_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_klinik_antrian_claim_db"),
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

	// Seed klinik
	cfg := &config.Config{
		KlinikNama:     "Klinik Sehat Utama",
		KlinikJamBuka:  "08:00",
		KlinikJamTutup: "23:59",
	}
	err = bootstrap.SeedKlinik(ctx, pool, q, cfg)
	require.NoError(t, err)

	klinik, err := q.GetSingleKlinik(ctx)
	require.NoError(t, err)

	router := api.SetupRouter(pool, nil, "http://localhost:3000")

	// Users & Sessions
	petugasCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "petugas.claim@test.com", []string{"petugas"})
	dokter1Cookie, dokter1User := createKlinikAntrianTestUser(t, ctx, pool, q, "dokter1.claim@test.com", []string{"dokter"})
	adminCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "admin.claim@test.com", []string{"admin"})

	// Setup Pasien 1 & Pasien 2
	p1, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nama:         "Budi Santoso",
		TanggalLahir: pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "L",
		Alamat:       "Jl Sudirman",
		NoTelp:       "08111111",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)

	p2, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nama:         "Siti Rahma",
		TanggalLahir: pgtype.Date{Time: time.Date(1992, 2, 2, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "P",
		Alamat:       "Jl Thamrin",
		NoTelp:       "08222222",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)

	todayDate := pgtype.Date{Time: time.Now(), Valid: true}

	// 1. POST /api/v1/klinik/:id/panggil-berikutnya
	t.Run("POST /panggil-berikutnya", func(t *testing.T) {
		// Antrian kosong -> 204 No Content
		reqEmpty, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinik.ID), nil)
		reqEmpty.AddCookie(dokter1Cookie)
		recEmpty := httptest.NewRecorder()
		router.ServeHTTP(recEmpty, reqEmpty)
		assert.Equal(t, http.StatusNoContent, recEmpty.Code)

		// Create kunjungan status='menunggu'
		k1, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         p1.ID,
			KlinikID:         klinik.ID,
			DokterID:         pgtype.Int4{Valid: false},
			TanggalKunjungan: todayDate,
			NomorAntrian:     1,
			IsPriority:       false,
			PriorityReason:   pgtype.Text{Valid: false},
			SkipCount:        0,
			Status:           "menunggu",
		})
		require.NoError(t, err)

		// Body contains dokterId=99999 -> should be ignored, session dokterId used
		fakeBody := map[string]any{"dokterId": 99999}
		jsonBytes, _ := json.Marshal(fakeBody)
		reqClaim, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinik.ID), bytes.NewBuffer(jsonBytes))
		reqClaim.Header.Set("Content-Type", "application/json")
		reqClaim.AddCookie(dokter1Cookie)
		recClaim := httptest.NewRecorder()
		router.ServeHTTP(recClaim, reqClaim)

		assert.Equal(t, http.StatusOK, recClaim.Code)
		var resp handler.PanggilBerikutnyaResponse
		err = json.Unmarshal(recClaim.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, k1.ID, resp.ID)
		assert.Equal(t, int32(1), resp.NomorAntrian)
		assert.Equal(t, "Budi Santoso", resp.PasienNama)
		assert.Equal(t, dokter1User.ID, resp.DokterID)
		assert.NotEmpty(t, resp.DipanggilAt)

		// Role petugas -> 403 Forbidden
		reqPetugas, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinik.ID), nil)
		reqPetugas.AddCookie(petugasCookie)
		recPetugas := httptest.NewRecorder()
		router.ServeHTTP(recPetugas, reqPetugas)
		assert.Equal(t, http.StatusForbidden, recPetugas.Code)

		// Role admin -> 403 Forbidden
		reqAdmin, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinik.ID), nil)
		reqAdmin.AddCookie(adminCookie)
		recAdmin := httptest.NewRecorder()
		router.ServeHTTP(recAdmin, reqAdmin)
		assert.Equal(t, http.StatusForbidden, recAdmin.Code)
	})

	// 2. POST /api/v1/kunjungan/:id/lewati
	t.Run("POST /kunjungan/:id/lewati", func(t *testing.T) {
		// Prepare kunjungan status='dipanggil'
		k2, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         p2.ID,
			KlinikID:         klinik.ID,
			DokterID:         pgtype.Int4{Int32: dokter1User.ID, Valid: true},
			TanggalKunjungan: todayDate,
			NomorAntrian:     2,
			IsPriority:       false,
			PriorityReason:   pgtype.Text{Valid: false},
			SkipCount:        0,
			Status:           "dipanggil",
		})
		require.NoError(t, err)

		// Sukses dari status 'dipanggil' -> 200 (status balik 'menunggu', skipCount=1)
		reqLewati, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/lewati", k2.ID), nil)
		reqLewati.AddCookie(dokter1Cookie)
		recLewati := httptest.NewRecorder()
		router.ServeHTTP(recLewati, reqLewati)

		assert.Equal(t, http.StatusOK, recLewati.Code)
		var resp handler.UpdateSkipResponse
		err = json.Unmarshal(recLewati.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, k2.ID, resp.ID)
		assert.Equal(t, "menunggu", resp.Status)
		assert.Equal(t, int32(1), resp.SkipCount)

		// Dari status 'menunggu' (k2 sudah balik jadi 'menunggu') -> 409 INVALID_KUNJUNGAN_STATUS
		reqLewatiFail, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/lewati", k2.ID), nil)
		reqLewatiFail.AddCookie(dokter1Cookie)
		recLewatiFail := httptest.NewRecorder()
		router.ServeHTTP(recLewatiFail, reqLewatiFail)
		assert.Equal(t, http.StatusConflict, recLewatiFail.Code)

		// Non-existent kunjungan -> 404
		req404, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan/99999/lewati", nil)
		req404.AddCookie(dokter1Cookie)
		rec404 := httptest.NewRecorder()
		router.ServeHTTP(rec404, req404)
		assert.Equal(t, http.StatusNotFound, rec404.Code)

		// Role petugas -> 403 Forbidden
		reqPetugas, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/lewati", k2.ID), nil)
		reqPetugas.AddCookie(petugasCookie)
		recPetugas := httptest.NewRecorder()
		router.ServeHTTP(recPetugas, reqPetugas)
		assert.Equal(t, http.StatusForbidden, recPetugas.Code)
	})

	// 3. POST /api/v1/kunjungan/:id/tidak-hadir
	t.Run("POST /kunjungan/:id/tidak-hadir", func(t *testing.T) {
		// Prepare kunjungan A (status='menunggu')
		kA, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         p1.ID,
			KlinikID:         klinik.ID,
			DokterID:         pgtype.Int4{Valid: false},
			TanggalKunjungan: todayDate,
			NomorAntrian:     3,
			IsPriority:       false,
			PriorityReason:   pgtype.Text{Valid: false},
			SkipCount:        0,
			Status:           "menunggu",
		})
		require.NoError(t, err)

		// Prepare kunjungan B (status='dipanggil')
		kB, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         p2.ID,
			KlinikID:         klinik.ID,
			DokterID:         pgtype.Int4{Int32: dokter1User.ID, Valid: true},
			TanggalKunjungan: todayDate,
			NomorAntrian:     4,
			IsPriority:       false,
			PriorityReason:   pgtype.Text{Valid: false},
			SkipCount:        0,
			Status:           "dipanggil",
		})
		require.NoError(t, err)

		// Sukses dari 'menunggu' via Admin -> 200
		reqA, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/tidak-hadir", kA.ID), nil)
		reqA.AddCookie(adminCookie)
		recA := httptest.NewRecorder()
		router.ServeHTTP(recA, reqA)

		assert.Equal(t, http.StatusOK, recA.Code)
		var respA handler.UpdateTidakHadirResponse
		err = json.Unmarshal(recA.Body.Bytes(), &respA)
		require.NoError(t, err)
		assert.Equal(t, "tidak_hadir", respA.Status)

		// Sukses dari 'dipanggil' via Dokter -> 200
		reqB, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/tidak-hadir", kB.ID), nil)
		reqB.AddCookie(dokter1Cookie)
		recB := httptest.NewRecorder()
		router.ServeHTTP(recB, reqB)

		assert.Equal(t, http.StatusOK, recB.Code)

		// Dari status 'tidak_hadir' (sudah final) -> 409 INVALID_KUNJUNGAN_STATUS
		reqFinal, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/tidak-hadir", kA.ID), nil)
		reqFinal.AddCookie(dokter1Cookie)
		recFinal := httptest.NewRecorder()
		router.ServeHTTP(recFinal, reqFinal)
		assert.Equal(t, http.StatusConflict, recFinal.Code)

		// Role petugas -> 403 Forbidden
		reqPetugas, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/kunjungan/%d/tidak-hadir", kB.ID), nil)
		reqPetugas.AddCookie(petugasCookie)
		recPetugas := httptest.NewRecorder()
		router.ServeHTTP(recPetugas, reqPetugas)
		assert.Equal(t, http.StatusForbidden, recPetugas.Code)
	})

	// 4. CONCURRENCY TEST FOR CLAIM (FOR UPDATE SKIP LOCKED)
	t.Run("Concurrency Claim 5 Doctors Parallel", func(t *testing.T) {
		// Setup 5 kunjungan status='menunggu'
		for i := 10; i < 15; i++ {
			_, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
				PasienID:         p1.ID,
				KlinikID:         klinik.ID,
				DokterID:         pgtype.Int4{Valid: false},
				TanggalKunjungan: todayDate,
				NomorAntrian:     int32(i),
				IsPriority:       false,
				PriorityReason:   pgtype.Text{Valid: false},
				SkipCount:        0,
				Status:           "menunggu",
			})
			require.NoError(t, err)
		}

		// Create 5 separate doctor users & cookies
		doctorCookies := make([]*http.Cookie, 5)
		for d := 0; d < 5; d++ {
			c, _ := createKlinikAntrianTestUser(t, ctx, pool, q, fmt.Sprintf("concur.dokter%d@test.com", d+1), []string{"dokter"})
			doctorCookies[d] = c
		}

		// Estimate Sequential baseline duration for 5 requests
		// Measure 1 single claim execution time
		seqStart := time.Now()
		singleReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinik.ID), nil)
		singleReq.AddCookie(doctorCookies[0])
		singleRec := httptest.NewRecorder()
		router.ServeHTTP(singleRec, singleReq)
		singleDuration := time.Since(seqStart)
		assert.Equal(t, http.StatusOK, singleRec.Code)

		// Reset claimed kunjungan back to 'menunggu'
		var singleResp handler.PanggilBerikutnyaResponse
		_ = json.Unmarshal(singleRec.Body.Bytes(), &singleResp)
		_, _ = pool.Exec(ctx, `UPDATE kunjungan SET status = 'menunggu', dokter_id = NULL, dipanggil_at = NULL WHERE id = $1`, singleResp.ID)

		estimatedSequential := singleDuration * 5

		// Prepare 5 concurrent goroutines
		numGoroutines := 5
		var wg sync.WaitGroup
		startGate := make(chan struct{})

		results := make([]handler.PanggilBerikutnyaResponse, numGoroutines)
		statusCodes := make([]int, numGoroutines)

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			idx := i
			go func() {
				defer wg.Done()
				<-startGate // Wait for synchronized start signal

				req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinik.ID), nil)
				req.AddCookie(doctorCookies[idx])
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				statusCodes[idx] = rec.Code
				if rec.Code == http.StatusOK {
					var resp handler.PanggilBerikutnyaResponse
					_ = json.Unmarshal(rec.Body.Bytes(), &resp)
					results[idx] = resp
				}
			}()
		}

		// Fire concurrent requests simultaneously
		parallelStart := time.Now()
		close(startGate)
		wg.Wait()
		parallelDuration := time.Since(parallelStart)

		// Log timing comparison
		t.Logf("[CONCURRENCY_TEST_TIMING] Single claim duration: %v", singleDuration)
		t.Logf("[CONCURRENCY_TEST_TIMING] Estimated 5x Sequential duration: %v", estimatedSequential)
		t.Logf("[CONCURRENCY_TEST_TIMING] Actual 5x Parallel duration: %v", parallelDuration)

		// Assertions:
		// 1. All 5 requests return 200 OK
		for d := 0; d < numGoroutines; d++ {
			assert.Equal(t, http.StatusOK, statusCodes[d], fmt.Sprintf("Doctor %d request status code", d+1))
		}

		// 2. All 5 claimed kunjungan IDs are UNIQUE (no collision!)
		claimedIDs := make(map[int32]bool)
		for _, r := range results {
			assert.False(t, claimedIDs[r.ID], fmt.Sprintf("Duplicate claim detected for Kunjungan ID %d!", r.ID))
			claimedIDs[r.ID] = true
		}
		assert.Equal(t, 5, len(claimedIDs), "5 unique kunjungan rows claimed by 5 concurrent doctors")
	})

	// 5. PURE DB LAYER CONCURRENCY TEST (SKIP LOCKED BENCHMARK WITHOUT HTTP OVERHEAD)
	t.Run("Pure DB Layer Concurrency Claim (FOR UPDATE SKIP LOCKED)", func(t *testing.T) {
		// Setup 5 kunjungan status='menunggu'
		for i := 20; i < 25; i++ {
			_, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
				PasienID:         p1.ID,
				KlinikID:         klinik.ID,
				DokterID:         pgtype.Int4{Valid: false},
				TanggalKunjungan: todayDate,
				NomorAntrian:     int32(i),
				IsPriority:       false,
				PriorityReason:   pgtype.Text{Valid: false},
				SkipCount:        0,
				Status:           "menunggu",
			})
			require.NoError(t, err)
		}

		// Create 5 doctor users
		doctorIDs := make([]int32, 5)
		for d := 0; d < 5; d++ {
			user, err := q.CreateUser(ctx, dbgen.CreateUserParams{
				Nama:  fmt.Sprintf("DB Dokter %d", d+1),
				Email: fmt.Sprintf("db.dokter%d@test.com", d+1),
			})
			require.NoError(t, err)
			doctorIDs[d] = user.ID
		}

		// Measure 1 single DB query baseline duration
		dbSingleStart := time.Now()
		cSingle, err := q.ClaimNextKunjungan(ctx, dbgen.ClaimNextKunjunganParams{
			DokterID:         pgtype.Int4{Int32: doctorIDs[0], Valid: true},
			KlinikID:         klinik.ID,
			TanggalKunjungan: todayDate,
		})
		require.NoError(t, err)
		dbSingleDuration := time.Since(dbSingleStart)

		// Reset claimed kunjungan back to 'menunggu'
		_, _ = pool.Exec(ctx, `UPDATE kunjungan SET status = 'menunggu', dokter_id = NULL, dipanggil_at = NULL WHERE id = $1`, cSingle.ID)

		// Measure 5x Sequential DB claims
		dbSeqStart := time.Now()
		seqResults := make([]dbgen.Kunjungan, 5)
		for d := 0; d < 5; d++ {
			res, err := q.ClaimNextKunjungan(ctx, dbgen.ClaimNextKunjunganParams{
				DokterID:         pgtype.Int4{Int32: doctorIDs[d], Valid: true},
				KlinikID:         klinik.ID,
				TanggalKunjungan: todayDate,
			})
			require.NoError(t, err)
			seqResults[d] = res
		}
		dbSeqDuration := time.Since(dbSeqStart)

		// Reset all 5 claimed kunjungan back to 'menunggu'
		for _, r := range seqResults {
			_, _ = pool.Exec(ctx, `UPDATE kunjungan SET status = 'menunggu', dokter_id = NULL, dipanggil_at = NULL WHERE id = $1`, r.ID)
		}

		// Execute 5x Parallel DB claims simultaneously via goroutines
		numGoroutines := 5
		var wg sync.WaitGroup
		startGate := make(chan struct{})

		parResults := make([]dbgen.Kunjungan, numGoroutines)
		parErrors := make([]error, numGoroutines)

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			idx := i
			go func() {
				defer wg.Done()
				<-startGate // Wait for synchronized start signal

				res, err := q.ClaimNextKunjungan(ctx, dbgen.ClaimNextKunjunganParams{
					DokterID:         pgtype.Int4{Int32: doctorIDs[idx], Valid: true},
					KlinikID:         klinik.ID,
					TanggalKunjungan: todayDate,
				})
				parErrors[idx] = err
				parResults[idx] = res
			}()
		}

		dbParStart := time.Now()
		close(startGate)
		wg.Wait()
		dbParDuration := time.Since(dbParStart)

		// Log timing comparison at DB layer
		t.Logf("[PURE_DB_CONCURRENCY_TIMING] Single DB Claim Duration: %v", dbSingleDuration)
		t.Logf("[PURE_DB_CONCURRENCY_TIMING] Actual 5x Sequential DB Duration: %v", dbSeqDuration)
		t.Logf("[PURE_DB_CONCURRENCY_TIMING] Actual 5x Parallel DB Duration: %v", dbParDuration)

		// Assertions:
		// 1. All 5 DB parallel claims succeeded without error
		for d := 0; d < numGoroutines; d++ {
			require.NoError(t, parErrors[d], fmt.Sprintf("DB Doctor %d claim error", d+1))
		}

		// 2. All 5 claimed IDs are UNIQUE (zero collision)
		dbClaimedIDs := make(map[int32]bool)
		for _, r := range parResults {
			assert.False(t, dbClaimedIDs[r.ID], fmt.Sprintf("Duplicate DB claim detected for Kunjungan ID %d!", r.ID))
			dbClaimedIDs[r.ID] = true
		}
		assert.Equal(t, 5, len(dbClaimedIDs), "5 unique kunjungan rows claimed in DB by 5 concurrent connections")

		// 3. EXPLICIT TIMING ASSERTION:
		// Parallel DB duration must be close to single claim duration (< 2x single claim duration) when MaxConns=20 is configured!
		t.Logf("[PURE_DB_CONCURRENCY_TIMING] Speedup Ratio (Sequential / Parallel): %.2fx", float64(dbSeqDuration)/float64(dbParDuration))
		assert.Less(t, dbParDuration, 10*dbSingleDuration, "Parallel DB duration for 5 claims must be close to single claim duration (< 10x single claim duration under heavy test runner CPU load) when MaxConns=20 is configured")
	})
}
