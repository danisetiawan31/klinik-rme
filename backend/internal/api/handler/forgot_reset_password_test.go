package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/danisetiawan31/klinik-rme/internal/api/handler"
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/mailer"
)

func TestForgotAndResetPassword_Integration(t *testing.T) {
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
	router := api.SetupRouter(pool, mockMailer, frontendBaseURL)

	// Seed existing user
	passHash, _ := auth.Hash("OldPassword!123")
	var userID int32
	err = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Existing User", "existing@test.com", passHash).Scan(&userID)
	require.NoError(t, err)

	// Seed invited user (null password_hash)
	var invitedUserID int32
	err = pool.QueryRow(ctx, "INSERT INTO users (nama, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"Invited User", "invited@test.com", nil).Scan(&invitedUserID)
	require.NoError(t, err)

	t.Run("POST /api/v1/auth/forgot-password - Registered vs Non-registered (Always 200 OK)", func(t *testing.T) {
		// 1. Registered Email
		bodyReg, _ := json.Marshal(handler.ForgotPasswordRequest{Email: "existing@test.com"})
		wReg := httptest.NewRecorder()
		reqReg, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBuffer(bodyReg))
		reqReg.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wReg, reqReg)

		assert.Equal(t, http.StatusOK, wReg.Code)
		assert.Contains(t, wReg.Body.String(), "Jika email terdaftar, instruksi reset password telah dikirim")
		assert.NotContains(t, wReg.Body.String(), "token") // Token is NEVER returned in response body

		// 2. Non-registered Email (Returns identical 200 OK message)
		bodyNonReg, _ := json.Marshal(handler.ForgotPasswordRequest{Email: "nonexistent@test.com"})
		wNonReg := httptest.NewRecorder()
		reqNonReg, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewBuffer(bodyNonReg))
		reqNonReg.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wNonReg, reqNonReg)

		assert.Equal(t, http.StatusOK, wNonReg.Code)
		assert.Contains(t, wNonReg.Body.String(), "Jika email terdaftar, instruksi reset password telah dikirim")
	})

	t.Run("POST /api/v1/auth/reset-password - Reset Token (type='reset')", func(t *testing.T) {
		rawToken, _ := auth.GenerateToken()
		tokenHash := auth.HashToken(rawToken)

		err := q.InsertPasswordToken(ctx, dbgen.InsertPasswordTokenParams{
			TokenHash: tokenHash,
			UserID:    userID,
			Type:      "reset",
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(1 * time.Hour), Valid: true},
		})
		require.NoError(t, err)

		newPass := "NewResetPass!123"
		body, _ := json.Marshal(handler.ResetPasswordRequest{
			Token:        rawToken,
			PasswordBaru: newPass,
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify new password allows login
		loginBody, _ := json.Marshal(handler.LoginRequest{Email: "existing@test.com", Password: newPass})
		wLogin := httptest.NewRecorder()
		reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
		reqLogin.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wLogin, reqLogin)
		assert.Equal(t, http.StatusOK, wLogin.Code)
	})

	t.Run("POST /api/v1/auth/reset-password - Invite Token Completion (type='invite')", func(t *testing.T) {
		rawToken, _ := auth.GenerateToken()
		tokenHash := auth.HashToken(rawToken)

		err := q.InsertPasswordToken(ctx, dbgen.InsertPasswordTokenParams{
			TokenHash: tokenHash,
			UserID:    invitedUserID,
			Type:      "invite",
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
		})
		require.NoError(t, err)

		newPass := "FirstTimeInvitePass!123"
		body, _ := json.Marshal(handler.ResetPasswordRequest{
			Token:        rawToken,
			PasswordBaru: newPass,
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify invited user can now login with new password
		loginBody, _ := json.Marshal(handler.LoginRequest{Email: "invited@test.com", Password: newPass})
		wLogin := httptest.NewRecorder()
		reqLogin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
		reqLogin.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wLogin, reqLogin)
		assert.Equal(t, http.StatusOK, wLogin.Code)
	})

	t.Run("POST /api/v1/auth/reset-password - Concurrency Test (2 Concurrent Requests, Real PostgreSQL)", func(t *testing.T) {
		rawToken, _ := auth.GenerateToken()
		tokenHash := auth.HashToken(rawToken)

		err := q.InsertPasswordToken(ctx, dbgen.InsertPasswordTokenParams{
			TokenHash: tokenHash,
			UserID:    userID,
			Type:      "reset",
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(1 * time.Hour), Valid: true},
		})
		require.NoError(t, err)

		var wg sync.WaitGroup
		statusCodes := make([]int, 2)

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				body, _ := json.Marshal(handler.ResetPasswordRequest{
					Token:        rawToken,
					PasswordBaru: "ConcurrentPassword!123",
				})

				w := httptest.NewRecorder()
				req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(w, req)

				statusCodes[idx] = w.Code
			}(i)
		}

		wg.Wait()

		// Assert exactly one request got 204 No Content, and the other got 400 Bad Request
		successCount := 0
		failCount := 0

		for _, code := range statusCodes {
			if code == http.StatusNoContent {
				successCount++
			} else if code == http.StatusBadRequest {
				failCount++
			}
		}

		assert.Equal(t, 1, successCount, "exactly 1 request must succeed in token consumption")
		assert.Equal(t, 1, failCount, "exactly 1 request must fail due to atomic token consumption constraint")
	})

	t.Run("POST /api/v1/auth/reset-password - Expired & Invalid Tokens", func(t *testing.T) {
		// 1. Expired token -> 400 Bad Request
		expiredToken, _ := auth.GenerateToken()
		expiredHash := auth.HashToken(expiredToken)

		_ = q.InsertPasswordToken(ctx, dbgen.InsertPasswordTokenParams{
			TokenHash: expiredHash,
			UserID:    userID,
			Type:      "reset",
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(-1 * time.Hour), Valid: true},
		})

		bodyExpired, _ := json.Marshal(handler.ResetPasswordRequest{
			Token:        expiredToken,
			PasswordBaru: "NewPassword!123",
		})
		wExpired := httptest.NewRecorder()
		reqExpired, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(bodyExpired))
		reqExpired.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wExpired, reqExpired)

		assert.Equal(t, http.StatusBadRequest, wExpired.Code)

		// 2. Invalid token -> 400 Bad Request
		bodyInvalid, _ := json.Marshal(handler.ResetPasswordRequest{
			Token:        "completely_invalid_token",
			PasswordBaru: "NewPassword!123",
		})
		wInvalid := httptest.NewRecorder()
		reqInvalid, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewBuffer(bodyInvalid))
		reqInvalid.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wInvalid, reqInvalid)

		assert.Equal(t, http.StatusBadRequest, wInvalid.Code)
	})
}
