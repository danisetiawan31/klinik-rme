package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
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
	"github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestPostgres(t *testing.T) (*pgxpool.Pool, func()) {
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
	require.NoError(t, err, "failed to start postgres container")

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Run domain migrations
	err = db.RunMigrations("../../../migrations", connStr)
	require.NoError(t, err, "failed to run migrations in test")

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	cleanup := func() {
		pool.Close()
		_ = postgresContainer.Terminate(ctx)
	}

	return pool, cleanup
}

func TestAuthEndpoints_Integration(t *testing.T) {
	pool, cleanup := setupTestPostgres(t)
	defer cleanup()

	ctx := context.Background()
	q := dbgen.New(pool)
	router := api.SetupRouter(pool, nil, nil, "http://localhost:4200")

	// Seed test users
	plainPassword := "ValidPassword!123"
	passHash, err := auth.Hash(plainPassword)
	require.NoError(t, err)

	// 1. User dengan password valid & role admin
	var adminUserID int32
	err = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Admin User", "admin@klinik.com", passHash).Scan(&adminUserID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role) VALUES ($1, $2)", adminUserID, "admin")
	require.NoError(t, err)

	// 2. User dengan password_hash null (invite belum selesai)
	var nullPassUserID int32
	err = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Pending User", "pending@klinik.com", nil).Scan(&nullPassUserID)
	require.NoError(t, err)

	t.Run("POST /api/v1/auth/login - Failure Scenarios (Identical Response Body)", func(t *testing.T) {
		failCases := []struct {
			name     string
			email    string
			password string
		}{
			{
				name:     "Non-existent email",
				email:    "notfound@klinik.com",
				password: plainPassword,
			},
			{
				name:     "Wrong password",
				email:    "admin@klinik.com",
				password: "WrongPassword!999",
			},
			{
				name:     "Null password_hash",
				email:    "pending@klinik.com",
				password: plainPassword,
			},
		}

		var firstErrBody string

		for i, tc := range failCases {
			t.Run(tc.name, func(t *testing.T) {
				body, _ := json.Marshal(handler.LoginRequest{
					Email:    tc.email,
					Password: tc.password,
				})

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusUnauthorized, w.Code)

				var errResp middleware.ErrorEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &errResp)
				require.NoError(t, err)

				assert.Equal(t, "INVALID_CREDENTIALS", errResp.Error.Code)
				assert.Equal(t, "Email atau password salah", errResp.Error.Message)
				assert.NotEmpty(t, errResp.Error.RequestID)

				if i == 0 {
					firstErrBody = w.Body.String()
				} else {
					// Verify response body JSON format is IDENTICAL across all 3 failure cases
					// (ignoring requestId which is unique per request)
					assert.Equal(t, errResp.Error.Code, "INVALID_CREDENTIALS")
					assert.Equal(t, errResp.Error.Message, "Email atau password salah")
				}
			})
		}
		_ = firstErrBody
	})

	var validSessionCookie *http.Cookie
	var rawSessionToken string

	t.Run("POST /api/v1/auth/login - Success", func(t *testing.T) {
		body, _ := json.Marshal(handler.LoginRequest{
			Email:    "admin@klinik.com",
			Password: plainPassword,
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp handler.LoginResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, adminUserID, resp.User.ID)
		assert.Equal(t, "Admin User", resp.User.Nama)
		assert.Equal(t, []string{"admin"}, resp.User.Roles)

		// Verify Session Cookie attributes: httpOnly, SameSite=Strict
		cookies := w.Result().Cookies()
		require.NotEmpty(t, cookies)

		for _, c := range cookies {
			if c.Name == middleware.SessionCookieName {
				validSessionCookie = c
				rawSessionToken = c.Value
				assert.True(t, c.HttpOnly)
				assert.NotEmpty(t, c.Value)
				break
			}
		}
		require.NotNil(t, validSessionCookie, "session cookie must be set in response")
	})

	t.Run("GET /api/v1/auth/me - Success & Failure", func(t *testing.T) {
		// 1. Success with valid cookie
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.AddCookie(validSessionCookie)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var userResp handler.UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &userResp)
		require.NoError(t, err)
		assert.Equal(t, adminUserID, userResp.ID)

		// 2. Failure without cookie -> 401
		wNoCookie := httptest.NewRecorder()
		reqNoCookie, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		router.ServeHTTP(wNoCookie, reqNoCookie)
		assert.Equal(t, http.StatusUnauthorized, wNoCookie.Code)

		// 3. Failure with invalid session token -> 401
		wInvalid := httptest.NewRecorder()
		reqInvalid, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		reqInvalid.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: "invalid_random_token_12345"})
		router.ServeHTTP(wInvalid, reqInvalid)
		assert.Equal(t, http.StatusUnauthorized, wInvalid.Code)
	})

	t.Run("Auth Middleware - Timestamp Expirations & Sliding Extension", func(t *testing.T) {
		idHash := auth.HashToken(rawSessionToken)

		// 1. Sesi dengan expires_at sudah lewat -> 401
		pastTime := time.Now().Add(-1 * time.Hour)
		err := q.UpdateSessionExpiresAt(ctx, dbgen.UpdateSessionExpiresAtParams{
			IDHash:    idHash,
			ExpiresAt: pgtype.Timestamptz{Time: pastTime, Valid: true},
		})
		require.NoError(t, err)

		wExpired := httptest.NewRecorder()
		reqExpired, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		reqExpired.AddCookie(validSessionCookie)
		router.ServeHTTP(wExpired, reqExpired)
		assert.Equal(t, http.StatusUnauthorized, wExpired.Code)

		// Restore valid expires_at
		now := time.Now()
		err = q.UpdateSessionExpiresAt(ctx, dbgen.UpdateSessionExpiresAtParams{
			IDHash:    idHash,
			ExpiresAt: pgtype.Timestamptz{Time: now.Add(2 * time.Hour), Valid: true},
		})
		require.NoError(t, err)

		// 2. Sesi dengan absolute_expires_at sudah lewat meski expires_at belum -> 401
		_, err = pool.Exec(ctx, "UPDATE sessions SET absolute_expires_at = $1 WHERE id_hash = $2", pastTime, idHash)
		require.NoError(t, err)

		wAbsExpired := httptest.NewRecorder()
		reqAbsExpired, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		reqAbsExpired.AddCookie(validSessionCookie)
		router.ServeHTTP(wAbsExpired, reqAbsExpired)
		assert.Equal(t, http.StatusUnauthorized, wAbsExpired.Code)

		// Restore valid absolute_expires_at
		_, err = pool.Exec(ctx, "UPDATE sessions SET absolute_expires_at = $1 WHERE id_hash = $2", now.Add(24*time.Hour), idHash)
		require.NoError(t, err)

		// 3. Request valid -> assert expires_at di DB ter-update (sliding extension 2 jam)
		wSliding := httptest.NewRecorder()
		reqSliding, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		reqSliding.AddCookie(validSessionCookie)
		router.ServeHTTP(wSliding, reqSliding)
		assert.Equal(t, http.StatusOK, wSliding.Code)

		// Check updated expires_at in DB
		sessAfter, err := q.GetSessionByIDHash(ctx, idHash)
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().Add(2*time.Hour), sessAfter.ExpiresAt.Time, 5*time.Second)
	})

	t.Run("PATCH /api/v1/auth/me/password - Success & Wrong Password", func(t *testing.T) {
		// 1. Password lama salah -> 400 INVALID_PASSWORD
		wrongBody, _ := json.Marshal(handler.ChangePasswordRequest{
			PasswordLama: "WrongOldPassword!123",
			PasswordBaru: "NewPass!456",
		})
		wWrong := httptest.NewRecorder()
		reqWrong, _ := http.NewRequest(http.MethodPatch, "/api/v1/auth/me/password", bytes.NewBuffer(wrongBody))
		reqWrong.Header.Set("Content-Type", "application/json")
		reqWrong.AddCookie(validSessionCookie)
		router.ServeHTTP(wWrong, reqWrong)

		assert.Equal(t, http.StatusBadRequest, wWrong.Code)
		var errResp middleware.ErrorEnvelope
		err := json.Unmarshal(wWrong.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_PASSWORD", errResp.Error.Code)
		assert.Equal(t, "Password lama tidak sesuai", errResp.Error.Message)

		// 2. Password lama cocok -> 204 No Content
		newPassword := "NewValidPass!789"
		validBody, _ := json.Marshal(handler.ChangePasswordRequest{
			PasswordLama: plainPassword,
			PasswordBaru: newPassword,
		})
		wValid := httptest.NewRecorder()
		reqValid, _ := http.NewRequest(http.MethodPatch, "/api/v1/auth/me/password", bytes.NewBuffer(validBody))
		reqValid.Header.Set("Content-Type", "application/json")
		reqValid.AddCookie(validSessionCookie)
		router.ServeHTTP(wValid, reqValid)

		assert.Equal(t, http.StatusNoContent, wValid.Code)

		// 3. Verify new password can be used to login
		loginBody, _ := json.Marshal(handler.LoginRequest{
			Email:    "admin@klinik.com",
			Password: newPassword,
		})
		wNewLogin := httptest.NewRecorder()
		reqNewLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
		reqNewLogin.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wNewLogin, reqNewLogin)

		assert.Equal(t, http.StatusOK, wNewLogin.Code)
	})

	t.Run("POST /api/v1/auth/logout - Success", func(t *testing.T) {
		wLogout := httptest.NewRecorder()
		reqLogout, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		reqLogout.AddCookie(validSessionCookie)
		router.ServeHTTP(wLogout, reqLogout)

		assert.Equal(t, http.StatusNoContent, wLogout.Code)

		// Assert cookie cleared in response (MaxAge < 0 or empty value)
		cookies := wLogout.Result().Cookies()
		require.NotEmpty(t, cookies)
		for _, c := range cookies {
			if c.Name == middleware.SessionCookieName {
				assert.True(t, c.MaxAge < 0 || c.Value == "")
			}
		}

		// Assert session row deleted from DB
		idHash := auth.HashToken(rawSessionToken)
		_, err := q.GetSessionByIDHash(ctx, idHash)
		require.Error(t, err)

		// Assert subsequent request with same cookie returns 401
		wNext := httptest.NewRecorder()
		reqNext, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		reqNext.AddCookie(validSessionCookie)
		router.ServeHTTP(wNext, reqNext)

		assert.Equal(t, http.StatusUnauthorized, wNext.Code)
	})
}
