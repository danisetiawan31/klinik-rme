package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func createKlinikAntrianTestUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, q *dbgen.Queries, email string, roles []string) (*http.Cookie, dbgen.CreateUserRow) {
	t.Helper()
	user, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		Nama:  "User " + roles[0],
		Email: email,
	})
	require.NoError(t, err)

	for _, role := range roles {
		err = q.InsertUserRole(ctx, dbgen.InsertUserRoleParams{
			UserID: user.ID,
			Role:   role,
		})
		require.NoError(t, err)
	}

	rawToken, err := auth.GenerateToken()
	require.NoError(t, err)
	idHash := auth.HashToken(rawToken)

	now := time.Now()
	exp := now.Add(2 * time.Hour)
	absExp := now.Add(24 * time.Hour)

	_, err = pool.Exec(ctx, `INSERT INTO sessions (id_hash, user_id, created_at, expires_at, absolute_expires_at) 
		VALUES ($1, $2, $3, $4, $5)`, idHash, user.ID, now, exp, absExp)
	require.NoError(t, err)

	cookie := &http.Cookie{
		Name:  "session",
		Value: rawToken,
	}

	return cookie, user
}

func TestKlinikAntrianEndpoints_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_klinik_antrian_handler_db"),
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

	router := api.SetupRouter(pool, nil, nil, "http://localhost:3000")

	petugasCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "petugas.antrian@test.com", []string{"petugas"})
	dokterCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "dokter.antrian@test.com", []string{"dokter"})
	adminCookie, _ := createKlinikAntrianTestUser(t, ctx, pool, q, "admin.antrian@test.com", []string{"admin"})

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

	// Soft-deleted pasien
	pDeleted, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nama:         "Pasien Terhapus",
		TanggalLahir: pgtype.Date{Time: time.Date(1995, 3, 3, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "L",
		Alamat:       "Jl Hapus",
		NoTelp:       "08333333",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE pasien SET deleted_at = now() WHERE id = $1`, pDeleted.ID)
	require.NoError(t, err)

	// 1. GET /api/v1/klinik/:id
	t.Run("GET /klinik/:id", func(t *testing.T) {
		// Sukses
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/klinik/1", nil)
		req.AddCookie(petugasCookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var kResp handler.KlinikResponse
		err := json.Unmarshal(rec.Body.Bytes(), &kResp)
		require.NoError(t, err)
		assert.Equal(t, klinik.ID, kResp.ID)
		assert.Equal(t, "Klinik Sehat Utama", kResp.Nama)
		assert.Equal(t, "08:00", kResp.JamBuka)
		assert.Equal(t, "23:59", kResp.JamTutup)

		// Non-existent klinik -> 404
		req404, _ := http.NewRequest(http.MethodGet, "/api/v1/klinik/9999", nil)
		req404.AddCookie(petugasCookie)
		rec404 := httptest.NewRecorder()
		router.ServeHTTP(rec404, req404)
		assert.Equal(t, http.StatusNotFound, rec404.Code)

		// Invalid ID param -> 400
		req400, _ := http.NewRequest(http.MethodGet, "/api/v1/klinik/invalid", nil)
		req400.AddCookie(petugasCookie)
		rec400 := httptest.NewRecorder()
		router.ServeHTTP(rec400, req400)
		assert.Equal(t, http.StatusBadRequest, rec400.Code)
	})

	// 2. POST /api/v1/kunjungan
	t.Run("POST /kunjungan", func(t *testing.T) {
		// Sukses (1st POST -> nomorAntrian=1)
		body1 := handler.CreateKunjunganRequest{
			PasienID: p1.ID,
		}
		jsonBytes1, _ := json.Marshal(body1)
		req1, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(jsonBytes1))
		req1.Header.Set("Content-Type", "application/json")
		req1.AddCookie(petugasCookie)
		rec1 := httptest.NewRecorder()
		router.ServeHTTP(rec1, req1)

		assert.Equal(t, http.StatusCreated, rec1.Code)
		var k1Resp handler.CreateKunjunganResponse
		err := json.Unmarshal(rec1.Body.Bytes(), &k1Resp)
		require.NoError(t, err)
		assert.Equal(t, int32(1), k1Resp.NomorAntrian)
		assert.Equal(t, "menunggu", k1Resp.Status)
		assert.NotEmpty(t, k1Resp.TanggalKunjungan)

		// Sukses (2nd POST -> nomorAntrian=2)
		isPriority := true
		reason := "Lansia"
		body2 := handler.CreateKunjunganRequest{
			PasienID:       p2.ID,
			IsPriority:     &isPriority,
			PriorityReason: &reason,
		}
		jsonBytes2, _ := json.Marshal(body2)
		req2, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(jsonBytes2))
		req2.Header.Set("Content-Type", "application/json")
		req2.AddCookie(adminCookie)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		assert.Equal(t, http.StatusCreated, rec2.Code)
		var k2Resp handler.CreateKunjunganResponse
		err = json.Unmarshal(rec2.Body.Bytes(), &k2Resp)
		require.NoError(t, err)
		assert.Equal(t, int32(2), k2Resp.NomorAntrian)

		// Pasien tidak ditemukan -> 404 PASIEN_NOT_FOUND
		body404 := handler.CreateKunjunganRequest{
			PasienID: 99999,
		}
		jsonBytes404, _ := json.Marshal(body404)
		req404, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(jsonBytes404))
		req404.Header.Set("Content-Type", "application/json")
		req404.AddCookie(petugasCookie)
		rec404 := httptest.NewRecorder()
		router.ServeHTTP(rec404, req404)
		assert.Equal(t, http.StatusNotFound, rec404.Code)

		// Soft-deleted pasien -> 404 PASIEN_NOT_FOUND
		bodyDel := handler.CreateKunjunganRequest{
			PasienID: pDeleted.ID,
		}
		jsonBytesDel, _ := json.Marshal(bodyDel)
		reqDel, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(jsonBytesDel))
		reqDel.Header.Set("Content-Type", "application/json")
		reqDel.AddCookie(petugasCookie)
		recDel := httptest.NewRecorder()
		router.ServeHTTP(recDel, reqDel)
		assert.Equal(t, http.StatusNotFound, recDel.Code)

		// Role dokter -> 403 Forbidden
		reqDoc, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(jsonBytes1))
		reqDoc.Header.Set("Content-Type", "application/json")
		reqDoc.AddCookie(dokterCookie)
		recDoc := httptest.NewRecorder()
		router.ServeHTTP(recDoc, reqDoc)
		assert.Equal(t, http.StatusForbidden, recDoc.Code)

		// Tanpa Auth -> 401 Unauthorized
		reqNoAuth, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(jsonBytes1))
		reqNoAuth.Header.Set("Content-Type", "application/json")
		recNoAuth := httptest.NewRecorder()
		router.ServeHTTP(recNoAuth, reqNoAuth)
		assert.Equal(t, http.StatusUnauthorized, recNoAuth.Code)

		// Test KLINIK_TUTUP -> Manipulasi klinik jam_tutup ke waktu lampau (00:01)
		_, err = pool.Exec(ctx, `UPDATE klinik SET jam_tutup = '00:01:00' WHERE id = $1`, klinik.ID)
		require.NoError(t, err)

		reqClosed, _ := http.NewRequest(http.MethodPost, "/api/v1/kunjungan", bytes.NewBuffer(jsonBytes1))
		reqClosed.Header.Set("Content-Type", "application/json")
		reqClosed.AddCookie(petugasCookie)
		recClosed := httptest.NewRecorder()
		router.ServeHTTP(recClosed, reqClosed)
		assert.Equal(t, http.StatusBadRequest, recClosed.Code)

		// Restore jam_tutup
		_, err = pool.Exec(ctx, `UPDATE klinik SET jam_tutup = '23:59:59' WHERE id = $1`, klinik.ID)
		require.NoError(t, err)
	})

	// 3. GET /api/v1/kunjungan/:id
	t.Run("GET /kunjungan/:id", func(t *testing.T) {
		// Sukses
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/kunjungan/1", nil)
		req.AddCookie(dokterCookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var kResp handler.GetKunjunganResponse
		err := json.Unmarshal(rec.Body.Bytes(), &kResp)
		require.NoError(t, err)
		assert.Equal(t, int32(1), kResp.ID)
		assert.Equal(t, p1.ID, kResp.PasienID)
		assert.Equal(t, int32(1), kResp.NomorAntrian)
		assert.Equal(t, "menunggu", kResp.Status)
		assert.False(t, kResp.IsPriority)
		assert.Nil(t, kResp.DokterID)
		assert.Nil(t, kResp.DipanggilAt)

		// Non-existent kunjungan -> 404
		req404, _ := http.NewRequest(http.MethodGet, "/api/v1/kunjungan/99999", nil)
		req404.AddCookie(dokterCookie)
		rec404 := httptest.NewRecorder()
		router.ServeHTTP(rec404, req404)
		assert.Equal(t, http.StatusNotFound, rec404.Code)
	})

	// 4. GET /api/v1/klinik/:id/antrian
	t.Run("GET /klinik/:id/antrian", func(t *testing.T) {
		// Sukses: return list dengan pasienNama benar urut nomor_antrian ASC
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/klinik/1/antrian", nil)
		req.AddCookie(petugasCookie)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var list []handler.AntrianItemResponse
		err := json.Unmarshal(rec.Body.Bytes(), &list)
		require.NoError(t, err)
		require.Len(t, list, 2)

		assert.Equal(t, int32(1), list[0].NomorAntrian)
		assert.Equal(t, "Budi Santoso", list[0].PasienNama)

		assert.Equal(t, int32(2), list[1].NomorAntrian)
		assert.Equal(t, "Siti Rahma", list[1].PasienNama)
		assert.True(t, list[1].IsPriority)

		// Test array kosong untuk klinik tanpa kunjungan
		// (buat klinik baru tanpa kunjungan)
		resNewKlinik, err := pool.Exec(ctx, `INSERT INTO klinik (nama, jam_buka, jam_tutup) VALUES ('Klinik Kosong', '08:00:00', '17:00:00')`)
		require.NoError(t, err)
		_ = resNewKlinik

		reqEmpty, _ := http.NewRequest(http.MethodGet, "/api/v1/klinik/2/antrian", nil)
		reqEmpty.AddCookie(petugasCookie)
		recEmpty := httptest.NewRecorder()
		router.ServeHTTP(recEmpty, reqEmpty)

		assert.Equal(t, http.StatusOK, recEmpty.Code)
		var emptyList []handler.AntrianItemResponse
		err = json.Unmarshal(recEmpty.Body.Bytes(), &emptyList)
		require.NoError(t, err)
		assert.NotNil(t, emptyList)
		assert.Len(t, emptyList, 0)
	})
}
