package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/api"
	"github.com/danisetiawan31/klinik-rme/internal/api/handler"
	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/mailer"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func captureLogOutput(fn func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	fn()
	return buf.String()
}

func TestAuthAndRBACFoundation_FullLifecycleE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full lifecycle E2E integration test in short mode")
	}

	ctx := context.Background()

	// Spin up a single PostgreSQL container shared across all E2E steps (a-m)
	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_e2e_db"),
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

	err = db.RunMigrations("../../migrations", connStr)
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	q := dbgen.New(pool)
	mockMailer := mailer.NewMockMailer(false)
	frontendBaseURL := "http://localhost:4200"
	adminEmail := "admin.e2e@klinik.local"

	router := api.SetupRouter(pool, mockMailer, frontendBaseURL)

	var adminRawInviteToken string
	var adminCookie *http.Cookie
	var doctorRawInviteToken string
	var doctorCookie *http.Cookie
	var doctorResetToken string

	// Helper for HTTP requests
	doRequest := func(method, path string, bodyObj interface{}, cookie *http.Cookie) (*httptest.ResponseRecorder, error) {
		var reqBody []byte
		if bodyObj != nil {
			reqBody, _ = json.Marshal(bodyObj)
		}
		w := httptest.NewRecorder()
		req, err := http.NewRequest(method, path, bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, err
		}
		if bodyObj != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		router.ServeHTTP(w, req)
		return w, nil
	}

	// -------------------------------------------------------------------------
	// Step a: Bootstrap (SeedAdmin on fresh DB)
	// -------------------------------------------------------------------------
	t.Run("Step a - Bootstrap SeedAdmin on fresh DB", func(t *testing.T) {
		logText := captureLogOutput(func() {
			err := bootstrap.SeedAdmin(ctx, pool, q, adminEmail)
			require.NoError(t, err)
		})

		assert.Contains(t, logText, "[ADMIN_BOOTSTRAP] Created initial admin account")
		assert.Contains(t, logText, "[ADMIN_BOOTSTRAP] Generated new admin invite token")

		// Intercept and parse raw token from server log
		re := regexp.MustCompile(`Generated new admin invite token for [^:]+: ([A-Za-z0-9_-]+)`)
		matches := re.FindStringSubmatch(logText)
		require.Len(t, matches, 2, "must successfully extract admin invite token from log")
		adminRawInviteToken = matches[1]
		assert.NotEmpty(t, adminRawInviteToken)
	})

	// -------------------------------------------------------------------------
	// Step b: Complete Admin Password Setup via POST /auth/reset-password
	// -------------------------------------------------------------------------
	t.Run("Step b - Complete Admin Password Setup", func(t *testing.T) {
		w, err := doRequest(http.MethodPost, "/api/v1/auth/reset-password", handler.ResetPasswordRequest{
			Token:        adminRawInviteToken,
			PasswordBaru: "AdminPass!123",
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	// -------------------------------------------------------------------------
	// Step c: Login as Admin with New Password
	// -------------------------------------------------------------------------
	t.Run("Step c - Login as Admin", func(t *testing.T) {
		w, err := doRequest(http.MethodPost, "/api/v1/auth/login", handler.LoginRequest{
			Email:    adminEmail,
			Password: "AdminPass!123",
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)

		for _, c := range w.Result().Cookies() {
			if c.Name == middleware.SessionCookieName {
				adminCookie = c
				break
			}
		}
		require.NotNil(t, adminCookie, "session cookie must be set on login")
		assert.NotEmpty(t, adminCookie.Value)
	})

	// -------------------------------------------------------------------------
	// Step d: GET /auth/me with Admin Cookie
	// -------------------------------------------------------------------------
	t.Run("Step d - GET /auth/me as Admin", func(t *testing.T) {
		w, err := doRequest(http.MethodGet, "/api/v1/auth/me", nil, adminCookie)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp handler.UserResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp.Roles, "admin")
	})

	// -------------------------------------------------------------------------
	// Step e: Admin Invites New Doctor User via POST /admin/users
	// -------------------------------------------------------------------------
	t.Run("Step e - Admin Invites Doctor User", func(t *testing.T) {
		doctorEmail := "dokter.e2e@klinik.local"
		w, err := doRequest(http.MethodPost, "/api/v1/admin/users", handler.CreateAdminUserRequest{
			Nama:  "Dokter E2E",
			Email: doctorEmail,
			Roles: []string{"dokter"},
		}, adminCookie)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp handler.AdminUserResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, doctorEmail, resp.Email)
		assert.Contains(t, resp.InviteLink, "http://localhost:4200/set-password?token=")

		// Parse token parameter from inviteLink URL
		u, err := url.Parse(resp.InviteLink)
		require.NoError(t, err)
		doctorRawInviteToken = u.Query().Get("token")
		require.NotEmpty(t, doctorRawInviteToken)
	})

	// -------------------------------------------------------------------------
	// Step f: Complete Doctor Password Setup via POST /auth/reset-password
	// -------------------------------------------------------------------------
	t.Run("Step f - Complete Doctor Password Setup", func(t *testing.T) {
		w, err := doRequest(http.MethodPost, "/api/v1/auth/reset-password", handler.ResetPasswordRequest{
			Token:        doctorRawInviteToken,
			PasswordBaru: "DokterInitialPass!123",
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	// -------------------------------------------------------------------------
	// Step g: Login as Doctor User
	// -------------------------------------------------------------------------
	t.Run("Step g - Login as Doctor User", func(t *testing.T) {
		w, err := doRequest(http.MethodPost, "/api/v1/auth/login", handler.LoginRequest{
			Email:    "dokter.e2e@klinik.local",
			Password: "DokterInitialPass!123",
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, w.Code)

		for _, c := range w.Result().Cookies() {
			if c.Name == middleware.SessionCookieName {
				doctorCookie = c
				break
			}
		}
		require.NotNil(t, doctorCookie, "doctor session cookie must be set")
		assert.NotEqual(t, adminCookie.Value, doctorCookie.Value, "doctor session token must differ from admin session token")
	})

	// -------------------------------------------------------------------------
	// Step h: RBAC Check — Doctor User Attempts Admin-Only Endpoint (POST /admin/users)
	// -------------------------------------------------------------------------
	t.Run("Step h - Doctor Attempts Admin Endpoint (403 Forbidden)", func(t *testing.T) {
		w, err := doRequest(http.MethodPost, "/api/v1/admin/users", handler.CreateAdminUserRequest{
			Nama:  "Unauthorized Staff",
			Email: "unauth@klinik.local",
			Roles: []string{"petugas"},
		}, doctorCookie)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, w.Code, "Doctor user must be rejected with 403 Forbidden when calling admin-only route")
	})

	// -------------------------------------------------------------------------
	// Step i: Forgot Password for Doctor User via POST /auth/forgot-password
	// -------------------------------------------------------------------------
	t.Run("Step i - Forgot Password Doctor", func(t *testing.T) {
		var w *httptest.ResponseRecorder
		logText := captureLogOutput(func() {
			var reqErr error
			w, reqErr = doRequest(http.MethodPost, "/api/v1/auth/forgot-password", handler.ForgotPasswordRequest{
				Email: "dokter.e2e@klinik.local",
			}, nil)
			require.NoError(t, reqErr)
		})

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Jika email terdaftar, instruksi reset password telah dikirim")
		assert.NotContains(t, w.Body.String(), "token")

		// Intercept and parse raw reset token from server log
		re := regexp.MustCompile(`Token reset generated for email [^:]+: ([A-Za-z0-9_-]+)`)
		matches := re.FindStringSubmatch(logText)
		require.Len(t, matches, 2, "must successfully extract reset token from server log")
		doctorResetToken = matches[1]
		assert.NotEmpty(t, doctorResetToken)
	})

	// -------------------------------------------------------------------------
	// Step j: Reset Password Doctor with Different New Password
	// -------------------------------------------------------------------------
	t.Run("Step j - Reset Password Doctor with New Password", func(t *testing.T) {
		w, err := doRequest(http.MethodPost, "/api/v1/auth/reset-password", handler.ResetPasswordRequest{
			Token:        doctorResetToken,
			PasswordBaru: "DokterBrandNewPass!456",
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	// -------------------------------------------------------------------------
	// Step k: Login with Old vs New Password
	// -------------------------------------------------------------------------
	t.Run("Step k - Login Doctor Old vs New Password", func(t *testing.T) {
		// 1. Old password must fail with 401
		wOld, err := doRequest(http.MethodPost, "/api/v1/auth/login", handler.LoginRequest{
			Email:    "dokter.e2e@klinik.local",
			Password: "DokterInitialPass!123",
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, wOld.Code)

		// 2. New password must succeed with 200
		wNew, err := doRequest(http.MethodPost, "/api/v1/auth/login", handler.LoginRequest{
			Email:    "dokter.e2e@klinik.local",
			Password: "DokterBrandNewPass!456",
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, wNew.Code)

		// Update doctorCookie for logout test step l
		for _, c := range wNew.Result().Cookies() {
			if c.Name == middleware.SessionCookieName {
				doctorCookie = c
				break
			}
		}
	})

	// -------------------------------------------------------------------------
	// Step l: Logout & Session Invalidation Check
	// -------------------------------------------------------------------------
	t.Run("Step l - Logout and Session Invalidation", func(t *testing.T) {
		// Logout -> 204
		wLogout, err := doRequest(http.MethodPost, "/api/v1/auth/logout", nil, doctorCookie)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, wLogout.Code)

		// Subsequent GET /auth/me with same cookie -> 401 Unauthorized (session row deleted in DB)
		wMe, err := doRequest(http.MethodGet, "/api/v1/auth/me", nil, doctorCookie)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, wMe.Code, "Session must be invalidated in DB after logout")
	})

	// -------------------------------------------------------------------------
	// Step m: Server Restart Simulation & SeedAdmin Idempotency Check
	// -------------------------------------------------------------------------
	t.Run("Step m - Server Restart Simulation & SeedAdmin Idempotency", func(t *testing.T) {
		var tokenCountBefore int
		_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM password_tokens WHERE user_id = (SELECT id FROM users WHERE email = $1)", adminEmail).Scan(&tokenCountBefore)

		logText := captureLogOutput(func() {
			err := bootstrap.SeedAdmin(ctx, pool, q, adminEmail)
			require.NoError(t, err)
		})

		assert.Contains(t, logText, "Admin account admin.e2e@klinik.local has already completed setup. Skipping bootstrap.")
		assert.NotContains(t, logText, "Generated new admin invite token")

		var tokenCountAfter int
		_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM password_tokens WHERE user_id = (SELECT id FROM users WHERE email = $1)", adminEmail).Scan(&tokenCountAfter)

		assert.Equal(t, tokenCountBefore, tokenCountAfter, "No new password_tokens row must be generated after admin setup is complete")
	})
}
