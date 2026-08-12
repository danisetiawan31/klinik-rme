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
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func createDisplayTokenTestUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, q *dbgen.Queries, email string, roles []string) (*http.Cookie, dbgen.CreateUserRow) {
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

func TestDisplayTokenAndDualAuth_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("klinik_rme_test"),
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
		KlinikNama:     "Klinik Pratama Sehat",
		KlinikJamBuka:  "08:00",
		KlinikJamTutup: "23:59",
	}
	err = bootstrap.SeedKlinik(ctx, pool, q, cfg)
	require.NoError(t, err)

	router := api.SetupRouter(pool, nil, "http://localhost:4200")

	adminCookie, _ := createDisplayTokenTestUser(t, ctx, pool, q, "admin_dt@klinik.id", []string{"admin"})
	dokterCookie, _ := createDisplayTokenTestUser(t, ctx, pool, q, "dokter_dt@klinik.id", []string{"dokter"})
	petugasCookie, _ := createDisplayTokenTestUser(t, ctx, pool, q, "petugas_dt@klinik.id", []string{"petugas"})

	// Setup Data Pasien & Kunjungan untuk testing GET /klinik/:id/antrian
	pasien, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
		Nik:          pgtype.Text{String: "3201012345678999", Valid: true},
		Nama:         "Pasien Dual Auth Test",
		TanggalLahir: pgtype.Date{Time: time.Date(1995, 5, 20, 0, 0, 0, 0, time.UTC), Valid: true},
		JenisKelamin: "L",
		ConsentAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	require.NoError(t, err)

	singleKlinik, err := q.GetSingleKlinik(ctx)
	require.NoError(t, err)
	klinikID := singleKlinik.ID

	todayDate := pgtype.Date{Time: time.Now(), Valid: true}
	_, err = q.InsertKunjungan(ctx, dbgen.InsertKunjunganParams{
		PasienID:         pasien.ID,
		KlinikID:         klinikID,
		TanggalKunjungan: todayDate,
		NomorAntrian:     1,
		IsPriority:       true,
		Status:           "menunggu",
	})
	require.NoError(t, err)

	var activeDisplayToken string

	t.Run("POST /admin/klinik/:id/display-token/regenerate — Unauthenticated -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/klinik/%d/display-token/regenerate", klinikID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("POST /admin/klinik/:id/display-token/regenerate — Non-Admin (Petugas/Dokter) -> 403", func(t *testing.T) {
		for _, cookie := range []*http.Cookie{petugasCookie, dokterCookie} {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/klinik/%d/display-token/regenerate", klinikID), nil)
			req.AddCookie(cookie)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
		}
	})

	t.Run("POST /admin/klinik/:id/display-token/regenerate — Klinik Tidak Ditemukan -> 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/klinik/99999/display-token/regenerate", nil)
		req.AddCookie(adminCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GET /klinik/:id/antrian — Display token di DB masih NULL -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/klinik/%d/antrian", klinikID), nil)
		req.Header.Set("X-Display-Token", "some-random-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("POST /admin/klinik/:id/display-token/regenerate — Sukses 200 (Admin)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/klinik/%d/display-token/regenerate", klinikID), nil)
		req.AddCookie(adminCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		rawToken, ok := resp["displayToken"]
		require.True(t, ok)
		require.NotEmpty(t, rawToken)
		activeDisplayToken = rawToken

		// Verifikasi hash di DB match hasil HashToken manual
		hashInDB, err := q.GetKlinikDisplayTokenHash(ctx, klinikID)
		require.NoError(t, err)
		require.True(t, hashInDB.Valid)
		assert.Equal(t, auth.HashToken(rawToken), hashInDB.String)
	})

	t.Run("POST /admin/klinik/:id/display-token/regenerate — Regenerate Ulang Overwrite Hash & Token Lama Revoked", func(t *testing.T) {
		oldToken := activeDisplayToken

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/klinik/%d/display-token/regenerate", klinikID), nil)
		req.AddCookie(adminCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		newToken := resp["displayToken"]
		require.NotEmpty(t, newToken)
		require.NotEqual(t, oldToken, newToken)
		activeDisplayToken = newToken

		// Assert token lama sekarang 401 saat dipakai di DualAuth
		reqOld := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/klinik/%d/antrian", klinikID), nil)
		reqOld.Header.Set("X-Display-Token", oldToken)
		wOld := httptest.NewRecorder()
		router.ServeHTTP(wOld, reqOld)
		assert.Equal(t, http.StatusUnauthorized, wOld.Code)
	})

	t.Run("GET /klinik/:id/antrian — Via Cookie Staff (Petugas/Dokter/Admin) -> Shape Lama (Ada id & pasienNama)", func(t *testing.T) {
		for _, cookie := range []*http.Cookie{petugasCookie, dokterCookie, adminCookie} {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/klinik/%d/antrian", klinikID), nil)
			req.AddCookie(cookie)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var items []map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &items)
			require.NoError(t, err)
			require.Len(t, items, 1)

			item := items[0]
			_, hasID := item["id"]
			_, hasNama := item["pasienNama"]
			assert.True(t, hasID, "Jalur cookie staff WAJIB menyertakan field 'id'")
			assert.True(t, hasNama, "Jalur cookie staff WAJIB menyertakan field 'pasienNama'")
		}
	})

	t.Run("GET /klinik/:id/antrian — Via Header X-Display-Token Valid -> Shape Publik (Tanpa id & pasienNama)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/klinik/%d/antrian", klinikID), nil)
		req.Header.Set("X-Display-Token", activeDisplayToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var items []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &items)
		require.NoError(t, err)
		require.Len(t, items, 1)

		item := items[0]
		_, hasID := item["id"]
		_, hasNama := item["pasienNama"]
		assert.False(t, hasID, "Jalur display-token publik TIDAK BOLEH menyertakan field 'id'")
		assert.False(t, hasNama, "Jalur display-token publik TIDAK BOLEH menyertakan field 'pasienNama'")
		assert.Equal(t, float64(1), item["nomorAntrian"])
		assert.Equal(t, true, item["isPriority"])
		assert.Equal(t, "menunggu", item["status"])
	})

	t.Run("GET /klinik/:id/antrian — Via Query Param ?displayToken= Valid (WS Fallback) -> Shape Publik", func(t *testing.T) {
		urlStr := fmt.Sprintf("/api/v1/klinik/%d/antrian?displayToken=%s", klinikID, activeDisplayToken)
		req := httptest.NewRequest(http.MethodGet, urlStr, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var items []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &items)
		require.NoError(t, err)
		require.Len(t, items, 1)

		item := items[0]
		_, hasID := item["id"]
		_, hasNama := item["pasienNama"]
		assert.False(t, hasID, "Query param displayToken publik TIDAK BOLEH menyertakan field 'id'")
		assert.False(t, hasNama, "Query param displayToken publik TIDAK BOLEH menyertakan field 'pasienNama'")
	})

	t.Run("GET /klinik/:id/antrian — Dual-Auth Keduanya Gagal (Tanpa Cookie & Tanpa Token) -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/klinik/%d/antrian", klinikID), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("GET /klinik/:id/antrian — Token Salah -> 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/klinik/%d/antrian", klinikID), nil)
		req.Header.Set("X-Display-Token", "invalid_token_value_xyz")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
