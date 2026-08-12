package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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

func TestLaporanHarian_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_laporan_db"),
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

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	q := dbgen.New(pool)
	r := api.SetupRouter(pool, nil, nil, "")

	// 1. Seed Klinik
	var klinikID int32
	err = pool.QueryRow(ctx, `
		INSERT INTO klinik (nama, jam_buka, jam_tutup)
		VALUES ('Klinik Sehat Laporan', '08:00', '17:00')
		RETURNING id;
	`).Scan(&klinikID)
	require.NoError(t, err)

	// 2. Seed Users & Sessions
	petugasCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "petugas_lap@test.com", []string{"petugas"})
	dokterCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "dokter_lap@test.com", []string{"dokter"})
	adminCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "admin_lap@test.com", []string{"admin"})

	// 3. Seed Pasien
	pasien, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nama:         "Pasien Laporan",
		TanggalLahir: pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "L",
		Alamat:       "Jl. Test",
		NoTelp:       "08123456789",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)

	// 4. Seed Kunjungan for a specific test date: 2026-08-12
	// 5 visits total: 1 menunggu, 1 dipanggil, 2 selesai, 1 tidak_hadir
	targetDateStr := "2026-08-12"
	statuses := []string{"menunggu", "dipanggil", "selesai", "tidak_hadir", "selesai"}
	for i, status := range statuses {
		_, err := pool.Exec(ctx, `
			INSERT INTO kunjungan (pasien_id, klinik_id, tanggal_kunjungan, nomor_antrian, is_priority, skip_count, status)
			VALUES ($1, $2, $3, $4, false, 0, $5);
		`, pasien.ID, klinikID, targetDateStr, i+1, status)
		require.NoError(t, err)
	}

	t.Run("Mixed status kunjungan pada tanggal 2026-08-12", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/laporan/harian?tanggal=2026-08-12", nil)
		req.AddCookie(petugasCookie)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp handler.LaporanHarianResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "2026-08-12", resp.Tanggal)
		assert.Equal(t, int32(5), resp.TotalKunjungan)
		assert.Equal(t, int32(2), resp.TotalSelesai)
		assert.Equal(t, int32(1), resp.TotalTidakHadir)
	})

	t.Run("Tanggal tanpa kunjungan (2099-01-01) -> returns 0, bukan error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/laporan/harian?tanggal=2099-01-01", nil)
		req.AddCookie(dokterCookie)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp handler.LaporanHarianResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "2099-01-01", resp.Tanggal)
		assert.Equal(t, int32(0), resp.TotalKunjungan)
		assert.Equal(t, int32(0), resp.TotalSelesai)
		assert.Equal(t, int32(0), resp.TotalTidakHadir)
	})

	t.Run("Parameter ?tanggal= tidak dikirim -> default ke hari ini (Asia/Jakarta)", func(t *testing.T) {
		loc, err := time.LoadLocation("Asia/Jakarta")
		if err != nil {
			loc = time.Local
		}
		expectedToday := time.Now().In(loc).Format("2006-01-02")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/laporan/harian", nil)
		req.AddCookie(adminCookie)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp handler.LaporanHarianResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, expectedToday, resp.Tanggal)
		assert.GreaterOrEqual(t, resp.TotalKunjungan, int32(0))
	})

	t.Run("Format tanggal invalid -> 400 Bad Request dengan TANGGAL_INVALID", func(t *testing.T) {
		invalidDates := []string{"12-08-2026", "2026/08/12", "invalid-date", "2026-13-45"}
		for _, invDate := range invalidDates {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/laporan/harian?tanggal=%s", invDate), nil)
			req.AddCookie(petugasCookie)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			var errResp struct {
				Error struct {
					Code      string `json:"code"`
					Message   string `json:"message"`
					RequestID string `json:"requestId"`
				} `json:"error"`
			}
			err := json.Unmarshal(w.Body.Bytes(), &errResp)
			require.NoError(t, err)
			assert.Equal(t, "TANGGAL_INVALID", errResp.Error.Code)
			assert.Contains(t, errResp.Error.Message, "Format tanggal tidak valid")
		}
	})

	t.Run("Request tanpa auth cookie -> 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/laporan/harian", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
