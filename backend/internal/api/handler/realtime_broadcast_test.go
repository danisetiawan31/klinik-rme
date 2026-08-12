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
	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/realtime"
)

func TestRealtimeBroadcast_5TriggersAndNilHub(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("realtime_broadcast_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := db.NewPool(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
	})

	err = db.RunMigrations("../../../migrations", connStr)
	require.NoError(t, err)

	q := dbgen.New(pool)
	cfg := &config.Config{
		KlinikNama:     "Klinik Realtime Broadcast",
		KlinikJamBuka:  "00:00",
		KlinikJamTutup: "23:59",
	}
	err = bootstrap.SeedKlinik(ctx, pool, q, cfg)
	require.NoError(t, err)

	klinik, err := q.GetSingleKlinik(ctx)
	require.NoError(t, err)
	klinikID := klinik.ID

	// Setup Hub
	hub := realtime.NewHub()
	hubCtx, hubCancel := context.WithCancel(context.Background())
	t.Cleanup(hubCancel)
	go hub.Run(hubCtx)

	router := api.SetupRouter(pool, hub, nil, "http://localhost:3000")

	// Users & Sessions
	petugasCookie, _ := createDisplayTokenTestUser(t, ctx, pool, q, "petugas.bc@klinik.id", []string{"petugas"})
	dokterCookie, _ := createDisplayTokenTestUser(t, ctx, pool, q, "dokter.bc@klinik.id", []string{"dokter"})

	// Helper register WS Client manual ke Hub
	registerMockWSClient := func() *realtime.Client {
		client := realtime.NewClient(klinikID)
		hub.RegisterClient(client)
		time.Sleep(20 * time.Millisecond)
		return client
	}

	// Helper assert receive notification
	assertNotificationReceived := func(t *testing.T, client *realtime.Client) {
		select {
		case msg, ok := <-client.Send:
			require.True(t, ok, "Channel client.Send harus terbuka")
			var payload map[string]string
			err := json.Unmarshal(msg, &payload)
			require.NoError(t, err)
			assert.Equal(t, "queue_updated", payload["type"])
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout: Client WS tidak menerima notifikasi broadcast dalam 1 detik")
		}
	}

	// Helper assert NO notification received
	assertNoNotificationReceived := func(t *testing.T, client *realtime.Client) {
		select {
		case msg := <-client.Send:
			t.Fatalf("Seharusnya TIDAK ada broadcast, tapi menerima message: %s", string(msg))
		case <-time.After(300 * time.Millisecond):
			// Sukses: tidak ada pesan diterima
		}
	}

	// Helper create Pasien
	pasien, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nama:         "Pasien Broadcast Test",
		Nik:          pgtype.Text{String: "3271234567890001", Valid: true},
		TanggalLahir: pgtype.Date{Time: time.Date(1995, 5, 20, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "L",
		Alamat:       "Jl Broadcast 1",
		NoTelp:       "081234567890",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)

	t.Run("1. Trigger Broadcast: CreateKunjungan [Sukses]", func(t *testing.T) {
		client := registerMockWSClient()
		defer hub.UnregisterClient(client)

		reqBody, _ := json.Marshal(handler.CreateKunjunganRequest{
			PasienID: pasien.ID,
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(petugasCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		assertNotificationReceived(t, client)
	})

	t.Run("2. Trigger Broadcast: PanggilBerikutnya [Sukses]", func(t *testing.T) {
		client := registerMockWSClient()
		defer hub.UnregisterClient(client)

		urlStr := fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinikID)
		req, _ := http.NewRequest(http.MethodPost, urlStr, nil)
		req.AddCookie(dokterCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		assertNotificationReceived(t, client)
	})

	t.Run("3. Trigger Broadcast: Lewati [Sukses]", func(t *testing.T) {
		// Ambil kunjungan yang baru dipanggil
		kunjunganDipanggil, err := q.GetKunjunganByID(ctx, 1)
		require.NoError(t, err)

		client := registerMockWSClient()
		defer hub.UnregisterClient(client)

		urlStr := fmt.Sprintf("/api/v1/kunjungan/%d/lewati", kunjunganDipanggil.ID)
		req, _ := http.NewRequest(http.MethodPost, urlStr, nil)
		req.AddCookie(dokterCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		assertNotificationReceived(t, client)
	})

	t.Run("4. Trigger Broadcast: TidakHadir [Sukses]", func(t *testing.T) {
		// Panggil antrian berikutnya dulu supaya statusnya 'dipanggil'
		reqPanggil, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/klinik/%d/panggil-berikutnya", klinikID), nil)
		reqPanggil.AddCookie(dokterCookie)
		wP := httptest.NewRecorder()
		router.ServeHTTP(wP, reqPanggil)
		require.Equal(t, http.StatusOK, wP.Code)

		client := registerMockWSClient()
		defer hub.UnregisterClient(client)

		urlStr := fmt.Sprintf("/api/v1/kunjungan/%d/tidak-hadir", 1)
		req, _ := http.NewRequest(http.MethodPost, urlStr, nil)
		req.AddCookie(dokterCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		assertNotificationReceived(t, client)
	})

	t.Run("5. Trigger Broadcast: CreateRekamMedisAwal [Sukses]", func(t *testing.T) {
		// Buat kunjungan baru & panggil
		newKunjungan, err := q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
			PasienID:         pasien.ID,
			KlinikID:         klinikID,
			TanggalKunjungan: pgtype.Date{Time: time.Now(), Valid: true},
			NomorAntrian:     99,
			Status:           "dipanggil",
		})
		require.NoError(t, err)

		client := registerMockWSClient()
		defer hub.UnregisterClient(client)

		rmReq := handler.CreateRekamMedisRequest{
			Keluhan:          "Batuk berdahak 3 hari",
			HasilPemeriksaan: "TDS 120/80 mmHg, Suhu 37.2C",
			Diagnosis: []handler.CreateDiagnosisItemRequest{
				{Deskripsi: "Acute Bronchitis"},
			},
			Tindakan: &[]handler.CreateTindakanItemRequest{
				{Jenis: "resep", Deskripsi: "Amoxicillin 500mg 3x1"},
			},
		}
		reqBody, _ := json.Marshal(rmReq)
		urlStr := fmt.Sprintf("/api/v1/kunjungan/%d/rekam-medis", newKunjungan.ID)
		req, _ := http.NewRequest(http.MethodPost, urlStr, bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokterCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		assertNotificationReceived(t, client)
	})

	t.Run("6. Negative Test: Failure / Invalid State -> TIDAK ADA Broadcast", func(t *testing.T) {
		client := registerMockWSClient()
		defer hub.UnregisterClient(client)

		// Lewati kunjungan yang statusnya sudah 'selesai' -> Conflict (409)
		urlStr := fmt.Sprintf("/api/v1/kunjungan/%d/lewati", 1)
		req, _ := http.NewRequest(http.MethodPost, urlStr, nil)
		req.AddCookie(dokterCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusConflict, w.Code)

		assertNoNotificationReceived(t, client)
	})

	t.Run("7. Test Khusus Guard Nil Hub: hub = nil -> Endpoint Berfungsi Normal & Tidak Panic", func(t *testing.T) {
		nilHubRouter := api.SetupRouter(pool, nil, nil, "http://localhost:3000")

		// Create kunjungan via router tanpa Hub
		reqBody, _ := json.Marshal(handler.CreateKunjunganRequest{
			PasienID: pasien.ID,
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(petugasCookie)

		w := httptest.NewRecorder()
		assert.NotPanics(t, func() {
			nilHubRouter.ServeHTTP(w, req)
		})
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}
