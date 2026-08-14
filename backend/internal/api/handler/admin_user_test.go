package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestAdminCreateUser_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
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

	// Seed admin user and staff user
	passHash, _ := auth.Hash("Password!123")
	var adminID, staffID int32

	_ = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Admin User", "admin@test.com", passHash).Scan(&adminID)
	_, _ = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", adminID, "admin")

	_ = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Staff User", "staff@test.com", passHash).Scan(&staffID)
	_, _ = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", staffID, "petugas")

	// Create valid login session for admin and staff
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

	adminCookie := loginSession("admin@test.com")
	staffCookie := loginSession("staff@test.com")

	t.Run("Success - Admin invites new user", func(t *testing.T) {
		reqBody, _ := json.Marshal(handler.CreateAdminUserRequest{
			Nama:  "Dokter Baru",
			Email: "dokter.baru@test.com",
			Roles: []string{"dokter"},
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp handler.AdminUserResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Greater(t, resp.ID, int32(0))
		assert.Equal(t, "Dokter Baru", resp.Nama)
		assert.Equal(t, "dokter.baru@test.com", resp.Email)
		assert.Equal(t, []string{"dokter"}, resp.Roles)
		assert.Contains(t, resp.InviteLink, "http://localhost:4200/set-password?token=")

		// Verify user in DB has password_hash NULL
		u, err := q.GetUserByID(ctx, resp.ID)
		require.NoError(t, err)
		assert.False(t, u.PasswordHash.Valid)

		// Verify roles in DB
		roles, err := q.GetRolesByUserID(ctx, resp.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"dokter"}, roles)
	})

	t.Run("Failure - Invalid Role (CHECK Constraint 23514 & DB Transaction Rollback)", func(t *testing.T) {
		reqBody, _ := json.Marshal(handler.CreateAdminUserRequest{
			Nama:  "Role Invalid User",
			Email: "invalidrole@test.com",
			Roles: []string{"superman"},
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "BAD_REQUEST", errResp.Error.Code)
		assert.Equal(t, "Peranan tidak valid", errResp.Error.Message)

		// Assert DB transaction rollback works cleanly (user is NOT created in DB)
		_, err = q.GetUserByEmail(ctx, "invalidrole@test.com")
		require.Error(t, err)
	})

	t.Run("Failure - Duplicate Email (Unique 23505 -> 409 Conflict)", func(t *testing.T) {
		reqBody, _ := json.Marshal(handler.CreateAdminUserRequest{
			Nama:  "Duplicate Admin",
			Email: "admin@test.com",
			Roles: []string{"admin"},
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "EMAIL_ALREADY_EXISTS", errResp.Error.Code)
		assert.Equal(t, "Email sudah terdaftar", errResp.Error.Message)
	})

	t.Run("Resend Fail -> User Creation Still Succeeds (201 Created)", func(t *testing.T) {
		failMailer := mailer.NewMockMailer(true)
		failRouter := api.SetupRouter(pool, nil, failMailer, frontendBaseURL)

		reqBody, _ := json.Marshal(handler.CreateAdminUserRequest{
			Nama:  "User Best Effort",
			Email: "besteffort@test.com",
			Roles: []string{"petugas"},
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		failRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Non-Admin -> 403 Forbidden", func(t *testing.T) {
		reqBody, _ := json.Marshal(handler.CreateAdminUserRequest{
			Nama:  "New Staff",
			Email: "newstaff@test.com",
			Roles: []string{"petugas"},
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(staffCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Unauthenticated -> 401 Unauthorized", func(t *testing.T) {
		reqBody, _ := json.Marshal(handler.CreateAdminUserRequest{
			Nama:  "New Staff",
			Email: "unauth@test.com",
			Roles: []string{"petugas"},
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Success - List Users (GET /admin/users)", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&limit=10", nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var users []handler.UserItemResponse
		err := json.Unmarshal(w.Body.Bytes(), &users)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 2)
		assert.Equal(t, strconv.Itoa(len(users)), w.Header().Get("X-Total-Count"))

		// Verify admin user is in the list
		foundAdmin := false
		for _, u := range users {
			if u.Email == "admin@test.com" {
				foundAdmin = true
				assert.Equal(t, []string{"admin"}, u.Roles)
			}
		}
		assert.True(t, foundAdmin, "admin user should be listed")

		// Verify X-Total-Count reflects total users before pagination limit=1
		wLimit := httptest.NewRecorder()
		reqLimit, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users?page=1&limit=1", nil)
		reqLimit.AddCookie(adminCookie)
		router.ServeHTTP(wLimit, reqLimit)
		assert.Equal(t, http.StatusOK, wLimit.Code)
		assert.Equal(t, w.Header().Get("X-Total-Count"), wLimit.Header().Get("X-Total-Count"), "total count must match even with limit=1")
	})

	t.Run("GET /admin/users - Non-Admin 403 Forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
		req.AddCookie(staffCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Success - Update User Nama Saja (PATCH /admin/users/:id)", func(t *testing.T) {
		newName := "Staff Nama Baru"
		body, _ := json.Marshal(map[string]string{"nama": newName})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d", staffID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp handler.UserItemResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, staffID, resp.ID)
		assert.Equal(t, "Staff Nama Baru", resp.Nama)
		assert.Equal(t, "staff@test.com", resp.Email)
		assert.Equal(t, []string{"petugas"}, resp.Roles)
	})

	t.Run("Success - Update User Email Saja (PATCH /admin/users/:id)", func(t *testing.T) {
		newEmail := "staff.diperbarui@test.com"
		body, _ := json.Marshal(map[string]string{"email": newEmail})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d", staffID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp handler.UserItemResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, staffID, resp.ID)
		assert.Equal(t, "Staff Nama Baru", resp.Nama)
		assert.Equal(t, "staff.diperbarui@test.com", resp.Email)
	})

	t.Run("Failure - Duplicate Email 409 Conflict (PATCH /admin/users/:id)", func(t *testing.T) {
		// Try to update staff's email to admin's email "admin@test.com"
		body, _ := json.Marshal(map[string]string{"email": "admin@test.com"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d", staffID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "EMAIL_ALREADY_EXISTS", errResp.Error.Code)
	})

	t.Run("Failure - User Not Found 404 (PATCH /admin/users/:id)", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"nama": "Ghost User"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/admin/users/99999", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "USER_NOT_FOUND", errResp.Error.Code)
	})

	t.Run("Failure - Non-Admin 403 Forbidden (PATCH /admin/users/:id)", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"nama": "Illegal Update"})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d", staffID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(staffCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Success - Resend Invite (POST /admin/users/:id/resend-invite)", func(t *testing.T) {
		// 1. Create a pending invite user
		createBody, _ := json.Marshal(handler.CreateAdminUserRequest{
			Nama:  "Pending User Resend",
			Email: "resend.pending@test.com",
			Roles: []string{"petugas"},
		})
		createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.AddCookie(adminCookie)
		createW := httptest.NewRecorder()
		router.ServeHTTP(createW, createReq)

		require.Equal(t, http.StatusCreated, createW.Code)
		var createResp handler.AdminUserResponse
		err := json.Unmarshal(createW.Body.Bytes(), &createResp)
		require.NoError(t, err)

		pendingUserID := createResp.ID
		oldInviteLink := createResp.InviteLink

		// Extract old raw token
		oldRawToken := oldInviteLink[bytes.Index([]byte(oldInviteLink), []byte("token="))+6:]
		oldTokenHash := auth.HashToken(string(oldRawToken))

		// 2. Perform Resend Invite
		resendW := httptest.NewRecorder()
		resendReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%d/resend-invite", pendingUserID), nil)
		resendReq.AddCookie(adminCookie)
		router.ServeHTTP(resendW, resendReq)

		assert.Equal(t, http.StatusNoContent, resendW.Code)

		// 3. Assert old token is invalidated/consumed and cannot be used
		_, err = q.ConsumePasswordToken(ctx, oldTokenHash)
		assert.Error(t, err, "old token should be consumed/invalidated and fail to consume")

		// 4. Assert new active invite token exists for pendingUserID
		newTokenRow, err := q.GetActiveInviteTokenByUserID(ctx, pendingUserID)
		require.NoError(t, err)
		assert.NotEmpty(t, newTokenRow.TokenHash)
		assert.NotEqual(t, oldTokenHash, newTokenRow.TokenHash)
	})

	t.Run("Failure - Resend Invite on Active User -> 400 USER_ALREADY_ACTIVE", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%d/resend-invite", adminID), nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "USER_ALREADY_ACTIVE", errResp.Error.Code)
	})

	t.Run("Failure - Resend Invite User Not Found -> 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/users/99999/resend-invite", nil)
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "USER_NOT_FOUND", errResp.Error.Code)
	})

	t.Run("Failure - Resend Invite Non-Admin -> 403 Forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%d/resend-invite", staffID), nil)
		req.AddCookie(staffCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// --- PATCH /admin/users/:id/roles Test Cases ---

	t.Run("Success - Update Roles Biasa (petugas -> dokter)", func(t *testing.T) {
		body, _ := json.Marshal(handler.UpdateUserRolesRequest{
			Roles: []string{"dokter"},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/roles", staffID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp handler.UserRolesResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, staffID, resp.ID)
		assert.Equal(t, []string{"dokter"}, resp.Roles)

		// Verify roles in DB
		rolesInDB, err := q.GetRolesByUserID(ctx, staffID)
		require.NoError(t, err)
		assert.Equal(t, []string{"dokter"}, rolesInDB)
	})

	t.Run("Success - Deduplicate Roles Input ([\"dokter\", \"dokter\"] -> [\"dokter\"])", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"roles": []string{"dokter", "dokter"},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/roles", staffID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp handler.UserRolesResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, []string{"dokter"}, resp.Roles)
	})

	t.Run("Failure - Roles Kosong [] -> 400 ROLES_CANNOT_BE_EMPTY", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"roles": []string{},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/roles", staffID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "ROLES_CANNOT_BE_EMPTY", errResp.Error.Code)
	})

	t.Run("Failure - Mutual Exclusion admin + dokter -> 400 MUTUAL_EXCLUSION_ROLES", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"roles": []string{"admin", "dokter"},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/roles", staffID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "MUTUAL_EXCLUSION_ROLES", errResp.Error.Code)
	})

	t.Run("Failure - Last Admin Guard (cuma 1 admin di sistem, hapus role admin) -> 400 LAST_ADMIN_GUARD", func(t *testing.T) {
		// Currently adminID is the ONLY admin in system
		body, _ := json.Marshal(map[string]interface{}{
			"roles": []string{"petugas"},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/roles", adminID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "LAST_ADMIN_GUARD", errResp.Error.Code)
	})

	t.Run("Success - Demote Admin ketika ada 2+ Admin di sistem", func(t *testing.T) {
		// Seed admin kedua (adminID2)
		var adminID2 int32
		_ = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
			"Admin Kedua", "admin2@test.com", passHash).Scan(&adminID2)
		_, _ = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", adminID2, "admin")

		// Sekarang ada 2 admin di sistem. Demote adminID2 ke petugas.
		body, _ := json.Marshal(map[string]interface{}{
			"roles": []string{"petugas"},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/roles", adminID2), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp handler.UserRolesResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, []string{"petugas"}, resp.Roles)
	})

	t.Run("Failure - User Not Found 404 (PATCH /admin/users/:id/roles)", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"roles": []string{"petugas"},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/admin/users/99999/roles", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "USER_NOT_FOUND", errResp.Error.Code)
	})

	t.Run("Failure - Non-Admin 403 Forbidden (PATCH /admin/users/:id/roles)", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"roles": []string{"petugas"},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/admin/users/%d/roles", staffID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(staffCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
