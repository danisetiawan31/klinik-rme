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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/api"
	"github.com/danisetiawan31/klinik-rme/internal/api/handler"
	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	"github.com/danisetiawan31/klinik-rme/internal/audit"
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func TestAdminAuditLog_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_audit_db"),
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
	router := api.SetupRouter(pool, nil, nil, "")

	// Seed admin user & staff user
	passHash, _ := auth.Hash("Password!123")
	var adminID, staffID int32

	_ = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Admin Audit", "admin.audit@test.com", passHash).Scan(&adminID)
	_, _ = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", adminID, "admin")

	_ = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Staff Audit", "staff.audit@test.com", passHash).Scan(&staffID)
	_, _ = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", staffID, "petugas")

	loginSession := func(email string) *http.Cookie {
		body, _ := json.Marshal(handler.LoginRequest{Email: email, Password: "Password!123"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		for _, c := range w.Result().Cookies() {
			if c.Name == middleware.SessionCookieName {
				return c
			}
		}
		t.Fatal("cookie not found")
		return nil
	}

	adminCookie := loginSession("admin.audit@test.com")
	staffCookie := loginSession("staff.audit@test.com")

	// Seed audit log entries via audit.Record helper
	tx1, err := pool.Begin(ctx)
	require.NoError(t, err)
	afterData1, _ := json.Marshal(map[string]interface{}{"nama": "Pasien Test 1", "nik": "1234567890123456"})
	err = audit.Record(ctx, tx1, q, adminID, "pasien", 101, "create", nil, afterData1)
	require.NoError(t, err)
	err = tx1.Commit(ctx)
	require.NoError(t, err)

	tx2, err := pool.Begin(ctx)
	require.NoError(t, err)
	beforeData2, _ := json.Marshal(map[string]interface{}{"nama": "Pasien Test 1"})
	afterData2, _ := json.Marshal(map[string]interface{}{"nama": "Pasien Test 1 Update"})
	err = audit.Record(ctx, tx2, q, staffID, "pasien", 101, "update", beforeData2, afterData2)
	require.NoError(t, err)
	err = tx2.Commit(ctx)
	require.NoError(t, err)

	tx3, err := pool.Begin(ctx)
	require.NoError(t, err)
	afterData3, _ := json.Marshal(map[string]interface{}{"keluhan": "Demam tinggi"})
	err = audit.Record(ctx, tx3, q, staffID, "rekam_medis", 501, "create", nil, afterData3)
	require.NoError(t, err)
	err = tx3.Commit(ctx)
	require.NoError(t, err)

	// Retrieve latest audit log entry IDs for testing detail
	rows, err := q.ListAuditLogs(ctx, dbgen.ListAuditLogsParams{Limit: 10, Offset: 0})
	require.NoError(t, err)
	require.Len(t, rows, 3)
	latestAuditID := rows[0].ID

	t.Run("1. List tanpa filter (pagination benar & X-Total-Count)", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/audit-log?page=1&limit=2", nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "3", w.Header().Get("X-Total-Count"), "X-Total-Count must be 3 even when limit=2")
		var list []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &list)
		require.NoError(t, err)
		assert.Len(t, list, 2, "page=1 & limit=2 should return exactly 2 items in body")
	})

	t.Run("1b. List filter kosong (X-Total-Count=0)", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/audit-log?tabelTarget=nonexistent", nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "0", w.Header().Get("X-Total-Count"))
		var list []map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &list)
		assert.Len(t, list, 0)
	})

	t.Run("2. List dengan kombinasi filter (tabelTarget + recordId + actorId sekaligus)", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/admin/audit-log?tabelTarget=pasien&recordId=101&actorId=%d", staffID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "1", w.Header().Get("X-Total-Count"))
		var list []handler.AuditLogSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &list)
		require.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, "pasien", list[0].TabelTarget)
		assert.Equal(t, int32(101), list[0].RecordID)
		assert.Equal(t, staffID, list[0].ActorUserID)
		assert.Equal(t, "update", list[0].Aksi)
	})

	t.Run("2b. Filter invalid int params (recordId=abc, actorId=xyz) -> diabaikan sebagai filter", func(t *testing.T) {
		url := "/api/v1/admin/audit-log?tabelTarget=pasien&recordId=abc&actorId=xyz"
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "2", w.Header().Get("X-Total-Count"))
		var list []handler.AuditLogSummaryResponse
		err := json.Unmarshal(w.Body.Bytes(), &list)
		require.NoError(t, err)
		// Filter recordId=abc & actorId=xyz diabaikan, filter tabelTarget=pasien tetap bekerja -> 2 rows (create & update)
		assert.Len(t, list, 2)
	})

	t.Run("3. List response TIDAK mengandung beforeData/afterData sama sekali", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/audit-log", nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var rawList []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &rawList)
		require.NoError(t, err)
		assert.NotEmpty(t, rawList)

		for _, item := range rawList {
			_, hasBefore := item["beforeData"]
			_, hasAfter := item["afterData"]
			assert.False(t, hasBefore, "List summary must NOT contain beforeData")
			assert.False(t, hasAfter, "List summary must NOT contain afterData")
		}
	})

	t.Run("4. Detail sukses, response mengandung beforeData/afterData/hashEntry", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/audit-log/%d", latestAuditID), nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var detail handler.AuditLogDetailResponse
		err := json.Unmarshal(w.Body.Bytes(), &detail)
		require.NoError(t, err)

		assert.Equal(t, latestAuditID, detail.ID)
		assert.Equal(t, "rekam_medis", detail.TabelTarget)
		assert.Equal(t, int32(501), detail.RecordID)
		assert.Equal(t, staffID, detail.ActorUserID)
		assert.NotEmpty(t, detail.AfterData, "Detail response MUST contain afterData")
		assert.NotEmpty(t, detail.HashEntry, "Detail response MUST contain hashEntry")
	})

	t.Run("5. Detail 404 id tidak ada", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/audit-log/99999", nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "AUDIT_LOG_NOT_FOUND", errResp.Error.Code)
	})

	t.Run("6. 403 non-admin untuk kedua endpoint", func(t *testing.T) {
		// GET /admin/audit-log
		w1 := httptest.NewRecorder()
		req1, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/audit-log", nil)
		req1.AddCookie(staffCookie)
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusForbidden, w1.Code)

		// GET /admin/audit-log/:id
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/audit-log/%d", latestAuditID), nil)
		req2.AddCookie(staffCookie)
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusForbidden, w2.Code)
	})
}
