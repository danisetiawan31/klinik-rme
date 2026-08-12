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
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/mailer"
)

func TestAdminFullLifecycle_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_admin_e2e_db"),
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
	mockMailer := mailer.NewMockMailer(false)
	frontendBaseURL := "http://localhost:4200"
	router := api.SetupRouter(pool, nil, mockMailer, frontendBaseURL)

	// --- Step a. Seed and Login Admin & Staff User ---
	passHash, _ := auth.Hash("Password!123")
	var adminID, staffID, pasienID int32

	_ = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Super Admin", "admin.e2e@test.com", passHash).Scan(&adminID)
	_, _ = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", adminID, "admin")

	_ = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Staff Petugas", "petugas.e2e@test.com", passHash).Scan(&staffID)
	_, _ = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", staffID, "petugas")

	// Seed 1 Pasien for testing RBAC on [dokter] endpoint later (step e)
	_ = pool.QueryRow(ctx, "INSERT INTO pasien (nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at) VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING id",
		"Pasien E2E", "1990-01-01", "L", "Jl. Test", "08123456789").Scan(&pasienID)

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

	adminCookie := loginSession("admin.e2e@test.com")
	staffCookie := loginSession("petugas.e2e@test.com")

	// --- Step b. GET /admin/users - Verifikasi User Awal ---
	t.Run("Step b. GET /admin/users - Verifikasi user awal", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&limit=10", nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var users []handler.UserItemResponse
		err := json.Unmarshal(w.Body.Bytes(), &users)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 2)
	})

	// --- Step c. PATCH /admin/users/:id - Update Biodata User & Reflect di GET /admin/users ---
	t.Run("Step c. PATCH /admin/users/:id - Update biodata user", func(t *testing.T) {
		patchBody, _ := json.Marshal(map[string]string{
			"nama":  "Staff Petugas Terkoreksi",
			"email": "petugas.terkoreksi@test.com",
		})
		patchW := httptest.NewRecorder()
		patchReq, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d", staffID), bytes.NewBuffer(patchBody))
		patchReq.Header.Set("Content-Type", "application/json")
		patchReq.AddCookie(adminCookie)
		router.ServeHTTP(patchW, patchReq)

		assert.Equal(t, http.StatusOK, patchW.Code)

		// Re-fetch via GET /admin/users to verify changes reflect
		getW := httptest.NewRecorder()
		getReq, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&limit=10", nil)
		getReq.AddCookie(adminCookie)
		router.ServeHTTP(getW, getReq)

		assert.Equal(t, http.StatusOK, getW.Code)
		var users []handler.UserItemResponse
		_ = json.Unmarshal(getW.Body.Bytes(), &users)

		found := false
		for _, u := range users {
			if u.ID == staffID {
				found = true
				assert.Equal(t, "Staff Petugas Terkoreksi", u.Nama)
				assert.Equal(t, "petugas.terkoreksi@test.com", u.Email)
			}
		}
		assert.True(t, found, "updated staff user should be found in GET /admin/users")
	})

	// --- Step d. POST /admin/users/:id/resend-invite - Resend Invite & Assert Token Invalidated ---
	var pendingUserID int32
	var oldTokenHash string

	t.Run("Step d1. Create pending user for invite test", func(t *testing.T) {
		createBody, _ := json.Marshal(handler.CreateAdminUserRequest{
			Nama:  "Pending Invite E2E",
			Email: "pending.e2e@test.com",
			Roles: []string{"petugas"},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(createBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp handler.AdminUserResponse
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		pendingUserID = resp.ID

		oldRawToken := resp.InviteLink[bytes.Index([]byte(resp.InviteLink), []byte("token="))+6:]
		oldTokenHash = auth.HashToken(string(oldRawToken))
	})

	t.Run("Step d2. Resend invite & verify old token invalidated", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%d/resend-invite", pendingUserID), nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Assert old token fails to consume
		_, err := q.ConsumePasswordToken(ctx, oldTokenHash)
		assert.Error(t, err, "old invite token must be invalidated")

		// Assert new active invite token exists
		newTokenRow, err := q.GetActiveInviteTokenByUserID(ctx, pendingUserID)
		require.NoError(t, err)
		assert.NotEmpty(t, newTokenRow.TokenHash)
		assert.NotEqual(t, oldTokenHash, newTokenRow.TokenHash)
	})

	// --- Step e. Real-time RBAC Enforcement without re-login ---
	t.Run("Step e. Change role petugas -> dokter and verify immediate RBAC access without re-login", func(t *testing.T) {
		// 1. Verify staff user currently FAILS accessing [dokter] endpoint GET /api/v1/pasien/:id/riwayat -> 403 Forbidden
		wBefore := httptest.NewRecorder()
		reqBefore, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/pasien/%d/riwayat", pasienID), nil)
		reqBefore.AddCookie(staffCookie)
		router.ServeHTTP(wBefore, reqBefore)
		assert.Equal(t, http.StatusForbidden, wBefore.Code, "staff with 'petugas' role should be rejected from [dokter] endpoint")

		// 2. Admin changes staff user's role from petugas -> dokter
		roleBody, _ := json.Marshal(handler.UpdateUserRolesRequest{Roles: []string{"dokter"}})
		roleW := httptest.NewRecorder()
		roleReq, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/roles", staffID), bytes.NewBuffer(roleBody))
		roleReq.Header.Set("Content-Type", "application/json")
		roleReq.AddCookie(adminCookie)
		router.ServeHTTP(roleW, roleReq)
		assert.Equal(t, http.StatusOK, roleW.Code)

		// 3. Staff user attempts accessing [dokter] endpoint AGAIN using the SAME session cookie (WITHOUT re-logging in) -> 200 OK
		wAfter := httptest.NewRecorder()
		reqAfter, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/pasien/%d/riwayat", pasienID), nil)
		reqAfter.AddCookie(staffCookie)
		router.ServeHTTP(wAfter, reqAfter)

		assert.Equal(t, http.StatusOK, wAfter.Code, "staff with newly assigned 'dokter' role must IMMEDIATELY pass RBAC check without re-logging in")
	})

	// --- Step f. LAST_ADMIN_GUARD Protection ---
	t.Run("Step f. Attempt deleting sole admin -> 400 LAST_ADMIN_GUARD", func(t *testing.T) {
		body, _ := json.Marshal(handler.UpdateUserRolesRequest{Roles: []string{"petugas"}})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/roles", adminID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var errResp middleware.ErrorEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		assert.Equal(t, "LAST_ADMIN_GUARD", errResp.Error.Code)
	})

	// --- Step g & h. GET /admin/audit-log & GET /admin/audit-log/:id ---
	var selectedAuditID int32

	t.Run("Step g1. Create Pasien via API to generate real audit log entry", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"nama":         "Pasien Audit E2E",
			"tanggalLahir": "1995-05-05",
			"jenisKelamin": "P",
			"alamat":       "Jl. Audit E2E",
			"noTelp":       "08987654321",
			"consent":      true,
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/pasien", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Step g2. GET /admin/audit-log filter kombinasi & no beforeData/afterData", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/audit-log?tabelTarget=pasien&page=1&limit=10", nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var list []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &list)
		require.NoError(t, err)
		assert.NotEmpty(t, list)

		for _, item := range list {
			_, hasBefore := item["beforeData"]
			_, hasAfter := item["afterData"]
			assert.False(t, hasBefore, "Summary list MUST NOT contain beforeData")
			assert.False(t, hasAfter, "Summary list MUST NOT contain afterData")
		}

		selectedAuditID = int32(list[0]["id"].(float64))
	})

	t.Run("Step h. GET /admin/audit-log/:id detail match record", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/audit-log/%d", selectedAuditID), nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var detail handler.AuditLogDetailResponse
		err := json.Unmarshal(w.Body.Bytes(), &detail)
		require.NoError(t, err)

		assert.Equal(t, selectedAuditID, detail.ID)
		assert.Equal(t, "pasien", detail.TabelTarget)
		assert.NotEmpty(t, detail.AfterData, "Detail response MUST contain afterData")
		assert.NotEmpty(t, detail.HashEntry, "Detail response MUST contain hashEntry")
	})
}
