package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
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
}
