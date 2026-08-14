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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/api"
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

type testErrorEnvelope struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	} `json:"error"`
}

// Helper function to create user & session in test database and return session cookie
func createTestUserWithSession(t *testing.T, ctx context.Context, pool *pgxpool.Pool, q *dbgen.Queries, email string, roles []string) (*http.Cookie, dbgen.User) {
	pwdHash, err := auth.Hash("TestPassword123!")
	require.NoError(t, err)

	var userID int32
	err = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Test User "+email, email, pwdHash).Scan(&userID)
	require.NoError(t, err)

	user, err := q.GetUserByID(ctx, userID)
	require.NoError(t, err)

	for _, r := range roles {
		err = q.InsertUserRole(ctx, dbgen.InsertUserRoleParams{
			UserID: user.ID,
			Role:   r,
		})
		require.NoError(t, err)
	}

	// Create valid session
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

func TestPasienEndpoints_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_pasien_handler_db"),
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
	router := api.SetupRouter(pool, nil, nil, "http://localhost:3000")

	// Create test users for authentication & authorization check
	petugasCookie, petugasUser := createTestUserWithSession(t, ctx, pool, q, "petugas@test.com", []string{"petugas"})
	dokterCookie, _ := createTestUserWithSession(t, ctx, pool, q, "dokter@test.com", []string{"dokter"})
	adminCookie, _ := createTestUserWithSession(t, ctx, pool, q, "admin@test.com", []string{"admin"})

	// 1. POST /api/v1/pasien
	t.Run("POST /api/v1/pasien", func(t *testing.T) {
		// a. Success (201, consent_at populated, version=1, audit_log created)
		bodySuccess := map[string]interface{}{
			"nik":          "3171010101900001",
			"nama":         "Budi Pasien Baru",
			"tanggalLahir": "1990-01-01",
			"jenisKelamin": "L",
			"alamat":       "Jl. Merdeka No. 10",
			"noTelp":       "08123456789",
			"consent":      true,
		}
		jsonBytes, _ := json.Marshal(bodySuccess)

		req := httptest.NewRequest("POST", "/api/v1/pasien", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(petugasCookie)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)

		var resObj map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &resObj)
		require.NoError(t, err)

		pasienID := int32(resObj["id"].(float64))
		assert.Equal(t, "3171010101900001", resObj["nik"])
		assert.Equal(t, "Budi Pasien Baru", resObj["nama"])
		assert.Equal(t, float64(1), resObj["version"])
		assert.NotEmpty(t, resObj["consentAt"])

		// Verify audit log entry in DB for create
		var auditCount int
		var aksi, tabelTarget string
		var actorUserID int32
		var afterDataRaw []byte

		err = pool.QueryRow(ctx, `SELECT COUNT(*), aksi, tabel_target, actor_user_id, after_data 
			FROM audit_log WHERE tabel_target = 'pasien' AND record_id = $1 GROUP BY aksi, tabel_target, actor_user_id, after_data`, pasienID).
			Scan(&auditCount, &aksi, &tabelTarget, &actorUserID, &afterDataRaw)
		require.NoError(t, err)
		assert.Equal(t, 1, auditCount)
		assert.Equal(t, "create", aksi)
		assert.Equal(t, "pasien", tabelTarget)
		assert.Equal(t, petugasUser.ID, actorUserID)

		var snapshot map[string]interface{}
		err = json.Unmarshal(afterDataRaw, &snapshot)
		require.NoError(t, err)
		assert.Equal(t, "3171010101900001", snapshot["nik"])
		assert.Equal(t, "Budi Pasien Baru", snapshot["nama"])
		assert.NotEmpty(t, snapshot["consentAt"])

		// b. consent=false / missing -> 400 CONSENT_REQUIRED, no pasien/audit inserted
		bodyNoConsent := map[string]interface{}{
			"nama":         "Pasien Tanpa Consent",
			"tanggalLahir": "1995-05-05",
			"jenisKelamin": "P",
			"alamat":       "Jl. Mawar",
			"noTelp":       "08999",
			"consent":      false,
		}
		jsonNoConsent, _ := json.Marshal(bodyNoConsent)

		req = httptest.NewRequest("POST", "/api/v1/pasien", bytes.NewBuffer(jsonNoConsent))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(petugasCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var errObj testErrorEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &errObj)
		assert.Equal(t, "CONSENT_REQUIRED", errObj.Error.Code)

		// Ensure no patient created with this name
		var countPasien int
		_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM pasien WHERE nama = 'Pasien Tanpa Consent'").Scan(&countPasien)
		assert.Equal(t, 0, countPasien)

		// c. Duplicate NIK -> 201 Created (both rows allowed with same NIK)
		req = httptest.NewRequest("POST", "/api/v1/pasien", bytes.NewBuffer(jsonBytes)) // Same NIK as first
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code, "duplicate NIK must be allowed and return 201")

		// d. Role dokter -> 403 Forbidden
		req = httptest.NewRequest("POST", "/api/v1/pasien", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokterCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)

		// e. Unauthenticated -> 401 Unauthorized
		req = httptest.NewRequest("POST", "/api/v1/pasien", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	// 2. GET /api/v1/pasien/search
	t.Run("GET /api/v1/pasien/search", func(t *testing.T) {
		// Clean table & insert controlled test dataset
		_, _ = pool.Exec(ctx, "TRUNCATE pasien RESTART IDENTITY CASCADE")

		// Insert 3 patients
		p1, _ := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "100001", Valid: true},
			Nama:         "Ahmad Yani",
			TanggalLahir: pgtype.Date{Time: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L", Alamat: "Jakarta", NoTelp: "0811", ConsentAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		_, _ = q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "100002", Valid: true},
			Nama:         "Ahmad Subardjo",
			TanggalLahir: pgtype.Date{Time: time.Date(1982, 2, 2, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L", Alamat: "Bogor", NoTelp: "0822", ConsentAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		p3, _ := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "200001", Valid: true},
			Nama:         "Dewi Sartika",
			TanggalLahir: pgtype.Date{Time: time.Date(1990, 3, 3, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "P", Alamat: "Bandung", NoTelp: "0833", ConsentAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})

		// Soft delete p3 manually
		_, _ = pool.Exec(ctx, "UPDATE pasien SET deleted_at = NOW() WHERE id = $1", p3.ID)

		// a. Search NIK exact match
		req := httptest.NewRequest("GET", "/api/v1/pasien/search?nik=100001", nil)
		req.AddCookie(dokterCookie)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "1", rec.Header().Get("X-Total-Count"))

		var searchRes []map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &searchRes)
		require.Len(t, searchRes, 1)
		assert.Equal(t, "Ahmad Yani", searchRes[0]["nama"])

		// b. Search Nama partial match
		req = httptest.NewRequest("GET", "/api/v1/pasien/search?nama=Ahmad", nil)
		req.AddCookie(petugasCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "2", rec.Header().Get("X-Total-Count"))

		_ = json.Unmarshal(rec.Body.Bytes(), &searchRes)
		assert.Len(t, searchRes, 2)

		// c. Combined NIK + Nama (AND logic)
		req = httptest.NewRequest("GET", "/api/v1/pasien/search?nik=100001&nama=Ahmad", nil)
		req.AddCookie(adminCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "1", rec.Header().Get("X-Total-Count"))

		_ = json.Unmarshal(rec.Body.Bytes(), &searchRes)
		require.Len(t, searchRes, 1)
		assert.Equal(t, float64(p1.ID), searchRes[0]["id"])

		// d. Pagination (total count must reflect total matched rows before limit/offset)
		req = httptest.NewRequest("GET", "/api/v1/pasien/search?page=1&limit=1", nil)
		req.AddCookie(petugasCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "2", rec.Header().Get("X-Total-Count"), "total count must be 2 even when limit=1")

		_ = json.Unmarshal(rec.Body.Bytes(), &searchRes)
		assert.Len(t, searchRes, 1)

		// e. Deleted record p3 should NOT appear in search
		req = httptest.NewRequest("GET", "/api/v1/pasien/search?nik=200001", nil)
		req.AddCookie(petugasCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "0", rec.Header().Get("X-Total-Count"), "total count must be 0 for empty search result")

		_ = json.Unmarshal(rec.Body.Bytes(), &searchRes)
		assert.Len(t, searchRes, 0, "soft-deleted patient must not appear in search")
	})

	// 3. GET /api/v1/pasien/:id
	t.Run("GET /api/v1/pasien/:id", func(t *testing.T) {
		p, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "555555", Valid: true},
			Nama:         "Pasien Detail Test",
			TanggalLahir: pgtype.Date{Time: time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "P", Alamat: "Alamat", NoTelp: "0855", ConsentAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)

		// a. Success 200 with riwayatKunjunganRingkas=[]
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/pasien/%d", p.ID), nil)
		req.AddCookie(dokterCookie)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		var detailRes map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &detailRes)
		assert.Equal(t, "Pasien Detail Test", detailRes["nama"])
		assert.NotNil(t, detailRes["riwayatKunjunganRingkas"])
		assert.Len(t, detailRes["riwayatKunjunganRingkas"], 0)

		// b. ID not found -> 404
		req = httptest.NewRequest("GET", "/api/v1/pasien/999999", nil)
		req.AddCookie(dokterCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		// c. Soft-deleted ID -> 404
		_, _ = pool.Exec(ctx, "UPDATE pasien SET deleted_at = NOW() WHERE id = $1", p.ID)
		req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/pasien/%d", p.ID), nil)
		req.AddCookie(petugasCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	// 4. PATCH /api/v1/pasien/:id
	t.Run("PATCH /api/v1/pasien/:id", func(t *testing.T) {
		p, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "777777", Valid: true},
			Nama:         "Pasien Patch Initial",
			TanggalLahir: pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L", Alamat: "Alamat Asli", NoTelp: "08777", ConsentAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)

		// a. Success with matching version (version 1 -> 2), audit log recorded with before & after snapshot
		newNama := "Pasien Patch Updated"
		newAlamat := "Alamat Baru Reborn"
		patchBody := map[string]interface{}{
			"version":   1,
			"nama":      newNama,
			"alamat":    newAlamat,
			"consent":   false,                             // Should be silently ignored
			"consentAt": time.Now().Add(24 * time.Hour), // Should be silently ignored
		}
		jsonPatch, _ := json.Marshal(patchBody)

		req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/pasien/%d", p.ID), bytes.NewBuffer(jsonPatch))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		var patchRes map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &patchRes)
		assert.Equal(t, float64(2), patchRes["version"], "version must increment to 2")
		assert.Equal(t, newNama, patchRes["nama"])
		assert.Equal(t, newAlamat, patchRes["alamat"])
		assert.Equal(t, "777777", patchRes["nik"], "unsent field nik must remain unchanged")

		// Verify audit log for update in DB
		var auditCount int
		var beforeRaw, afterRaw []byte
		err = pool.QueryRow(ctx, `SELECT COUNT(*), before_data, after_data FROM audit_log 
			WHERE tabel_target = 'pasien' AND record_id = $1 AND aksi = 'update' GROUP BY before_data, after_data`, p.ID).
			Scan(&auditCount, &beforeRaw, &afterRaw)
		require.NoError(t, err)
		assert.Equal(t, 1, auditCount)

		var bSnap, aSnap map[string]interface{}
		_ = json.Unmarshal(beforeRaw, &bSnap)
		_ = json.Unmarshal(afterRaw, &aSnap)
		assert.Equal(t, "Pasien Patch Initial", bSnap["nama"])
		assert.Equal(t, "Alamat Asli", bSnap["alamat"])
		assert.Equal(t, newNama, aSnap["nama"])
		assert.Equal(t, newAlamat, aSnap["alamat"])

		// b. Stale version (try version 1 when DB version is 2) -> 409 OPTIMISTIC_LOCK_FAILED
		staleBody := map[string]interface{}{
			"version": 1, // Stale version!
			"nama":    "Stale Attempt",
		}
		jsonStale, _ := json.Marshal(staleBody)

		req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/pasien/%d", p.ID), bytes.NewBuffer(jsonStale))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(petugasCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusConflict, rec.Code)

		var errRes testErrorEnvelope
		_ = json.Unmarshal(rec.Body.Bytes(), &errRes)
		assert.Equal(t, "OPTIMISTIC_LOCK_FAILED", errRes.Error.Code)

		// Verify DB record & audit_log were NOT changed by stale attempt
		pCurrent, _ := q.GetPasienByID(ctx, p.ID)
		assert.Equal(t, int32(2), pCurrent.Version)
		assert.Equal(t, newNama, pCurrent.Nama)

		// c. Non-existent ID -> 404 Not Found directly via pre-fetch without update
		req = httptest.NewRequest("PATCH", "/api/v1/pasien/999999", bytes.NewBuffer(jsonPatch))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(petugasCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)

		// d. Role dokter -> 403 Forbidden
		req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/pasien/%d", p.ID), bytes.NewBuffer(jsonPatch))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(dokterCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)

		// e. Soft-deleted patient -> 404 Not Found
		pDel, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "999999", Valid: true},
			Nama:         "Pasien Soft Delete Patch Test",
			TanggalLahir: pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L", Alamat: "Alamat", NoTelp: "08999", ConsentAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)

		_, _ = pool.Exec(ctx, "UPDATE pasien SET deleted_at = NOW() WHERE id = $1", pDel.ID)

		req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/pasien/%d", pDel.ID), bytes.NewBuffer(jsonPatch))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(petugasCookie)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
		_ = json.Unmarshal(rec.Body.Bytes(), &errRes)
		assert.Equal(t, "PASIEN_NOT_FOUND", errRes.Error.Code)
	})

	// 5. Concurrency Optimistic Lock Test (2 Parallel Goroutines with same version)
	t.Run("Concurrency Optimistic Lock Test", func(t *testing.T) {
		p, err := q.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          pgtype.Text{String: "888888", Valid: true},
			Nama:         "Pasien Concurrent Initial",
			TanggalLahir: pgtype.Date{Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			JenisKelamin: "L", Alamat: "Alamat Concurrent", NoTelp: "0888", ConsentAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		require.NoError(t, err)
		assert.Equal(t, int32(1), p.Version)

		var countBeforeAudit int
		_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_log WHERE tabel_target = 'pasien' AND record_id = $1 AND aksi = 'update'", p.ID).Scan(&countBeforeAudit)

		var wg sync.WaitGroup
		statusCodeChan := make(chan int, 2)

		for i := 1; i <= 2; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				body := map[string]interface{}{
					"version": 1, // BOTH goroutines use initial version 1!
					"nama":    fmt.Sprintf("Concurrent Update Worker %d", workerID),
				}
				jsonB, _ := json.Marshal(body)

				req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/pasien/%d", p.ID), bytes.NewBuffer(jsonB))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(petugasCookie)
				rec := httptest.NewRecorder()

				router.ServeHTTP(rec, req)
				statusCodeChan <- rec.Code
			}(i)
		}

		wg.Wait()
		close(statusCodeChan)

		codes := make([]int, 0, 2)
		for code := range statusCodeChan {
			codes = append(codes, code)
		}

		// Assert: EXACTLY 1 succeeds (200), EXACTLY 1 fails with 409
		assert.ElementsMatch(t, []int{http.StatusOK, http.StatusConflict}, codes, "concurrent patch with same version must result in 1 HTTP 200 and 1 HTTP 409")

		// Verify final state in DB
		pFinal, err := q.GetPasienByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, int32(2), pFinal.Version, "final version in DB must be initial+1 (2), NOT +2")

		// Verify audit log count: EXACTLY 1 new audit log row inserted for update (not 2)
		var countAfterAudit int
		_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM audit_log WHERE tabel_target = 'pasien' AND record_id = $1 AND aksi = 'update'", p.ID).Scan(&countAfterAudit)
		assert.Equal(t, countBeforeAudit+1, countAfterAudit, "only the successful patch must create an audit log entry")
	})
}
