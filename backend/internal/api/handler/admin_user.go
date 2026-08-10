package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/mailer"
)

type CreateAdminUserRequest struct {
	Nama  string   `json:"nama" binding:"required"`
	Email string   `json:"email" binding:"required,email"`
	Roles []string `json:"roles" binding:"required,min=1"`
}

type AdminUserResponse struct {
	ID         int32    `json:"id"`
	Nama       string   `json:"nama"`
	Email      string   `json:"email"`
	Roles      []string `json:"roles"`
	InviteLink string   `json:"inviteLink"`
}

// CreateUser handles POST /api/v1/admin/users [admin]
func CreateUser(pool *pgxpool.Pool, q *dbgen.Queries, emailSender mailer.EmailSender, frontendBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAdminUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format input tidak valid", err)
			return
		}

		ctx := c.Request.Context()

		// Begin explicit database transaction
		tx, err := pool.Begin(ctx)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menginisialisasi transaksi database", err)
			return
		}
		defer func() {
			_ = tx.Rollback(ctx)
		}()

		qtx := q.WithTx(tx)

		// 1. Create user with null password_hash
		newUser, err := qtx.CreateUser(ctx, dbgen.CreateUserParams{
			Nama:  req.Nama,
			Email: req.Email,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				middleware.RespondError(c, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "Email sudah terdaftar", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal membuat pengguna baru", err)
			return
		}

		// 2. Insert user roles
		for _, role := range req.Roles {
			err = qtx.InsertUserRole(ctx, dbgen.InsertUserRoleParams{
				UserID: newUser.ID,
				Role:   role,
			})
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23514" {
					middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Peranan tidak valid", err)
					return
				}
				middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menambahkan peranan pengguna", err)
				return
			}
		}

		// 3. Generate invite token & insert into password_tokens
		rawToken, err := auth.GenerateToken()
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal membuat token undangan", err)
			return
		}

		tokenHash := auth.HashToken(rawToken)
		now := time.Now()
		expiresAt := now.Add(7 * 24 * time.Hour) // Invite token TTL = 7 days

		err = qtx.InsertPasswordToken(ctx, dbgen.InsertPasswordTokenParams{
			TokenHash: tokenHash,
			UserID:    newUser.ID,
			Type:      "invite",
			ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		})
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menyimpan token undangan", err)
			return
		}

		// Commit transaction
		if err := tx.Commit(ctx); err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menyimpan transaksi pengguna", err)
			return
		}

		// 4. Send email invite (Best Effort after Commit)
		inviteLink := fmt.Sprintf("%s/set-password?token=%s", frontendBaseURL, rawToken)
		if emailSender != nil {
			if sendErr := emailSender.SendInviteEmail(ctx, newUser.Email, inviteLink); sendErr != nil {
				log.Printf("[WARNING] Failed to send invite email to %s: %v", newUser.Email, sendErr)
			}
		}

		c.JSON(http.StatusCreated, AdminUserResponse{
			ID:         newUser.ID,
			Nama:       newUser.Nama,
			Email:      newUser.Email,
			Roles:      req.Roles,
			InviteLink: inviteLink,
		})
	}
}
