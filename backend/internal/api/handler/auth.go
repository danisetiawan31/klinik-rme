package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/mailer"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID    int32    `json:"id"`
	Nama  string   `json:"nama"`
	Roles []string `json:"roles"`
}

type LoginResponse struct {
	User UserResponse `json:"user"`
}

type ChangePasswordRequest struct {
	PasswordLama string `json:"passwordLama" binding:"required"`
	PasswordBaru string `json:"passwordBaru" binding:"required"`
}

// setSessionCookie sets the HTTP session cookie with httpOnly, Secure, and SameSite=Strict attributes.
func setSessionCookie(c *gin.Context, token string, maxAge int) {
	c.SetSameSite(http.SameSiteStrictMode)
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie(
		middleware.SessionCookieName,
		token,
		maxAge,
		"/",
		"",
		isSecure,
		true, // httpOnly
	)
}

// Login handles POST /api/v1/auth/login
func Login(pool *pgxpool.Pool, q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format input tidak valid", err)
			return
		}

		ctx := c.Request.Context()

		// Helper to return generic invalid credentials error (prevents user enumeration)
		respondInvalidCredentials := func(rawErr error) {
			middleware.RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Email atau password salah", rawErr)
		}

		user, err := q.GetUserByEmail(ctx, req.Email)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				respondInvalidCredentials(nil)
				return
			}
			respondInvalidCredentials(err)
			return
		}

		// Reject login if password_hash is null (user invite not completed)
		if !user.PasswordHash.Valid || user.PasswordHash.String == "" {
			respondInvalidCredentials(nil)
			return
		}

		// Verify password using bcrypt helper
		match, err := auth.Verify(req.Password, user.PasswordHash.String)
		if err != nil || !match {
			respondInvalidCredentials(err)
			return
		}

		// Generate secure session token and compute SHA256 hash
		rawToken, err := auth.GenerateToken()
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal membuat token sesi", err)
			return
		}

		idHash := auth.HashToken(rawToken)
		now := time.Now()
		expiresAt := now.Add(2 * time.Hour)
		absoluteExpiresAt := now.Add(24 * time.Hour)

		err = q.InsertSession(ctx, dbgen.InsertSessionParams{
			IDHash: idHash,
			UserID: user.ID,
			CreatedAt: pgtype.Timestamptz{
				Time:  now,
				Valid: true,
			},
			ExpiresAt: pgtype.Timestamptz{
				Time:  expiresAt,
				Valid: true,
			},
			AbsoluteExpiresAt: pgtype.Timestamptz{
				Time:  absoluteExpiresAt,
				Valid: true,
			},
		})
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menyimpan sesi", err)
			return
		}

		roles, err := q.GetRolesByUserID(ctx, user.ID)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal membaca peranan pengguna", err)
			return
		}
		if roles == nil {
			roles = []string{}
		}

		// Set cookie httpOnly, Secure, SameSite=Strict
		setSessionCookie(c, rawToken, int((24 * time.Hour).Seconds()))

		c.JSON(http.StatusOK, LoginResponse{
			User: UserResponse{
				ID:    user.ID,
				Nama:  user.Nama,
				Roles: roles,
			},
		})
	}
}

// Logout handles POST /api/v1/auth/logout
func Logout(pool *pgxpool.Pool, q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		idHash, exists := middleware.GetSessionIDHashFromContext(c)
		if exists && idHash != "" {
			ctx := c.Request.Context()
			_ = q.DeleteSessionByIDHash(ctx, idHash)
		}

		// Clear session cookie in response
		setSessionCookie(c, "", -1)
		c.Status(http.StatusNoContent)
	}
}

// Me handles GET /api/v1/auth/me
func Me(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, existsUser := middleware.GetUserFromContext(c)
		roles, existsRoles := middleware.GetRolesFromContext(c)

		if !existsUser || !existsRoles {
			middleware.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak valid", nil)
			return
		}

		c.JSON(http.StatusOK, UserResponse{
			ID:    user.ID,
			Nama:  user.Nama,
			Roles: roles,
		})
	}
}

// ChangePassword handles PATCH /api/v1/auth/me/password
func ChangePassword(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ChangePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format input tidak valid", err)
			return
		}

		user, exists := middleware.GetUserFromContext(c)
		if !exists || !user.PasswordHash.Valid || user.PasswordHash.String == "" {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_PASSWORD", "Password lama tidak sesuai", nil)
			return
		}

		// Verify old password
		match, err := auth.Verify(req.PasswordLama, user.PasswordHash.String)
		if err != nil || !match {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_PASSWORD", "Password lama tidak sesuai", nil)
			return
		}

		// Hash new password
		newHash, err := auth.Hash(req.PasswordBaru)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal memproses password baru", err)
			return
		}

		ctx := c.Request.Context()
		err = q.UpdateUserPasswordHash(ctx, dbgen.UpdateUserPasswordHashParams{
			ID:           user.ID,
			PasswordHash: pgtype.Text{String: newHash, Valid: true},
		})
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal memperbarui password", err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

type ResetPasswordRequest struct {
	Token        string `json:"token" binding:"required"`
	PasswordBaru string `json:"passwordBaru" binding:"required"`
}

// ForgotPassword handles POST /api/v1/auth/forgot-password [public]
func ForgotPassword(q *dbgen.Queries, emailSender mailer.EmailSender, frontendBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ForgotPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format input tidak valid", err)
			return
		}

		// ALWAYS return generic 200 OK message regardless of email existence to prevent user enumeration
		genericSuccessMsg := "Jika email terdaftar, instruksi reset password telah dikirim"
		defer func() {
			c.JSON(http.StatusOK, ForgotPasswordResponse{Message: genericSuccessMsg})
		}()

		ctx := c.Request.Context()
		user, err := q.GetUserByEmail(ctx, req.Email)
		if err != nil {
			// User not found or DB error -> return generic 200 silently
			return
		}

		rawToken, err := auth.GenerateToken()
		if err != nil {
			log.Printf("[ERROR] Failed to generate reset token for %s: %v", req.Email, err)
			return
		}

		tokenHash := auth.HashToken(rawToken)
		now := time.Now()
		expiresAt := now.Add(1 * time.Hour) // Reset token TTL = 1 hour

		err = q.InsertPasswordToken(ctx, dbgen.InsertPasswordTokenParams{
			TokenHash: tokenHash,
			UserID:    user.ID,
			Type:      "reset",
			ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		})
		if err != nil {
			log.Printf("[ERROR] Failed to insert password token for %s: %v", req.Email, err)
			return
		}

		// Log raw token to server log for demo/presentation tail logging (TDD.md requirement)
		log.Printf("[FORGOT_PASSWORD] Token reset generated for email %s: %s (expires at %s)", user.Email, rawToken, expiresAt.Format(time.RFC3339))

		resetLink := fmt.Sprintf("%s/set-password?token=%s", frontendBaseURL, rawToken)
		if emailSender != nil {
			if sendErr := emailSender.SendResetEmail(ctx, user.Email, resetLink); sendErr != nil {
				log.Printf("[WARNING] Failed to send reset email to %s: %v", user.Email, sendErr)
			}
		}
	}
}

// ResetPassword handles POST /api/v1/auth/reset-password [public]
func ResetPassword(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ResetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format input tidak valid", err)
			return
		}

		ctx := c.Request.Context()
		tokenHash := auth.HashToken(req.Token)

		// Consume password token atomically in DB (applies to both type='invite' and type='reset')
		consumedRow, err := q.ConsumePasswordToken(ctx, tokenHash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusBadRequest, "INVALID_TOKEN", "Token reset/invite tidak valid, expired, atau sudah digunakan", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal memproses token reset/invite", err)
			return
		}

		// Hash new password
		newHash, err := auth.Hash(req.PasswordBaru)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal memproses password baru", err)
			return
		}

		// Update user password_hash
		err = q.UpdateUserPasswordHash(ctx, dbgen.UpdateUserPasswordHashParams{
			ID:           consumedRow.UserID,
			PasswordHash: pgtype.Text{String: newHash, Valid: true},
		})
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal memperbarui password pengguna", err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}
