package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
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

type UpdateAdminUserRequest struct {
	Nama  *string `json:"nama" binding:"omitempty,min=1"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type AdminUserResponse struct {
	ID         int32    `json:"id"`
	Nama       string   `json:"nama"`
	Email      string   `json:"email"`
	Roles      []string `json:"roles"`
	InviteLink string   `json:"inviteLink"`
}

type UserItemResponse struct {
	ID    int32    `json:"id"`
	Nama  string   `json:"nama"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// ListUsers handles GET /api/v1/admin/users?page=&limit= [admin]
func ListUsers(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		pageStr := c.DefaultQuery("page", "1")
		limitStr := c.DefaultQuery("limit", "10")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}
		offset := (page - 1) * limit

		ctx := c.Request.Context()
		rows, err := q.ListUsersWithRoles(ctx, dbgen.ListUsersWithRolesParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengambil daftar pengguna", err)
			return
		}

		res := make([]UserItemResponse, 0, len(rows))
		for _, row := range rows {
			roles := row.Roles
			if roles == nil {
				roles = []string{}
			}
			res = append(res, UserItemResponse{
				ID:    row.ID,
				Nama:  row.Nama,
				Email: row.Email,
				Roles: roles,
			})
		}

		c.JSON(http.StatusOK, res)
	}
}

// UpdateUser handles PATCH /api/v1/admin/users/:id [admin]
func UpdateUser(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		idParsed, err := strconv.Atoi(idParam)
		if err != nil || idParsed <= 0 {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "ID pengguna tidak valid", err)
			return
		}
		userID := int32(idParsed)

		var req UpdateAdminUserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format input tidak valid", err)
			return
		}

		if req.Nama == nil && req.Email == nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Tidak ada data yang diubah", nil)
			return
		}

		var namaText, emailText pgtype.Text
		if req.Nama != nil {
			namaText = pgtype.Text{String: *req.Nama, Valid: true}
		}
		if req.Email != nil {
			emailText = pgtype.Text{String: *req.Email, Valid: true}
		}

		ctx := c.Request.Context()
		updatedUser, err := q.UpdateUserBiodata(ctx, dbgen.UpdateUserBiodataParams{
			ID:    userID,
			Nama:  namaText,
			Email: emailText,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "USER_NOT_FOUND", "Pengguna tidak ditemukan", err)
				return
			}
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				middleware.RespondError(c, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "Email sudah terdaftar", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal memperbarui data pengguna", err)
			return
		}

		roles, err := q.GetRolesByUserID(ctx, updatedUser.ID)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengambil peranan pengguna", err)
			return
		}
		if roles == nil {
			roles = []string{}
		}

		c.JSON(http.StatusOK, UserItemResponse{
			ID:    updatedUser.ID,
			Nama:  updatedUser.Nama,
			Email: updatedUser.Email,
			Roles: roles,
		})
	}
}

// ResendInvite handles POST /api/v1/admin/users/:id/resend-invite [admin]
func ResendInvite(pool *pgxpool.Pool, q *dbgen.Queries, emailSender mailer.EmailSender, frontendBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		idParsed, err := strconv.Atoi(idParam)
		if err != nil || idParsed <= 0 {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "ID pengguna tidak valid", err)
			return
		}
		userID := int32(idParsed)

		ctx := c.Request.Context()

		// 1. Fetch user to check existence and password_hash status
		targetUser, err := q.GetUserByID(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "USER_NOT_FOUND", "Pengguna tidak ditemukan", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengambil data pengguna", err)
			return
		}

		// Reject if user already has password_hash (active user)
		if targetUser.PasswordHash.Valid && targetUser.PasswordHash.String != "" {
			middleware.RespondError(c, http.StatusBadRequest, "USER_ALREADY_ACTIVE", "Pengguna sudah aktif dan memiliki password", nil)
			return
		}

		// 2. Begin explicit database transaction
		tx, err := pool.Begin(ctx)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menginisialisasi transaksi database", err)
			return
		}
		defer func() {
			_ = tx.Rollback(ctx)
		}()

		qtx := q.WithTx(tx)

		// 2a. Invalidate previous active invite tokens for this user
		err = qtx.InvalidateActiveInviteTokensByUserID(ctx, userID)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menginvalidasi token undangan lama", err)
			return
		}

		// 2b. Generate new invite token
		rawToken, err := auth.GenerateToken()
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal membuat token undangan baru", err)
			return
		}

		tokenHash := auth.HashToken(rawToken)
		now := time.Now()
		expiresAt := now.Add(7 * 24 * time.Hour) // Invite token TTL = 7 days

		// 2c. Insert new invite token into password_tokens
		err = qtx.InsertPasswordToken(ctx, dbgen.InsertPasswordTokenParams{
			TokenHash: tokenHash,
			UserID:    userID,
			Type:      "invite",
			ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		})
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menyimpan token undangan baru", err)
			return
		}

		// Commit transaction
		if err := tx.Commit(ctx); err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menyimpan transaksi undangan", err)
			return
		}

		// 3. Send email invite (Best Effort after Commit)
		inviteLink := fmt.Sprintf("%s/set-password?token=%s", frontendBaseURL, rawToken)
		if emailSender != nil {
			if sendErr := emailSender.SendInviteEmail(ctx, targetUser.Email, inviteLink); sendErr != nil {
				log.Printf("[WARNING] Failed to resend invite email to %s: %v", targetUser.Email, sendErr)
			}
		}

		c.Status(http.StatusNoContent)
	}
}

type UpdateUserRolesRequest struct {
	Roles []string `json:"roles" binding:"required"`
}

type UserRolesResponse struct {
	ID    int32    `json:"id"`
	Roles []string `json:"roles"`
}

func dedupeStringSlice(input []string) []string {
	seen := make(map[string]bool, len(input))
	result := make([]string, 0, len(input))
	for _, item := range input {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// UpdateUserRoles handles PATCH /api/v1/admin/users/:id/roles [admin]
func UpdateUserRoles(pool *pgxpool.Pool, q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		idParsed, err := strconv.Atoi(idParam)
		if err != nil || idParsed <= 0 {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "ID pengguna tidak valid", err)
			return
		}
		userID := int32(idParsed)

		var req UpdateUserRolesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format input tidak valid", err)
			return
		}

		dedupedRoles := dedupeStringSlice(req.Roles)
		ctx := c.Request.Context()

		// 1. Cek user exist di DB (404 jika tidak ditemukan)
		_, err = q.GetUserByID(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "USER_NOT_FOUND", "Pengguna tidak ditemukan", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengambil data pengguna", err)
			return
		}

		// 2. Cek roles kosong (400 jika [])
		if len(dedupedRoles) == 0 {
			middleware.RespondError(c, http.StatusBadRequest, "ROLES_CANNOT_BE_EMPTY", "Minimal 1 peranan harus dipilih", nil)
			return
		}

		// 3. Cek mutual exclusion: "admin" DAN "dokter" tidak boleh sekaligus
		hasAdmin := false
		hasDokter := false
		for _, r := range dedupedRoles {
			if r == "admin" {
				hasAdmin = true
			}
			if r == "dokter" {
				hasDokter = true
			}
		}
		if hasAdmin && hasDokter {
			middleware.RespondError(c, http.StatusBadRequest, "MUTUAL_EXCLUSION_ROLES", "Peranan admin dan dokter tidak boleh dimiliki sekaligus oleh satu pengguna", nil)
			return
		}

		// 4. Cek last-admin guard (pre-check)
		currentRoles, err := q.GetRolesByUserID(ctx, userID)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengambil peranan pengguna", err)
			return
		}
		currentlyIsAdmin := false
		for _, r := range currentRoles {
			if r == "admin" {
				currentlyIsAdmin = true
				break
			}
		}

		if currentlyIsAdmin && !hasAdmin {
			adminCount, err := q.CountUsersWithRole(ctx, "admin")
			if err != nil {
				middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menghitung jumlah admin di sistem", err)
				return
			}
			if adminCount <= 1 {
				middleware.RespondError(c, http.StatusBadRequest, "LAST_ADMIN_GUARD", "Tidak dapat menghapus peranan admin terakhir di sistem", nil)
				return
			}
		}

		// 5. Eksekusi replace roles dalam 1 transaksi eksplisit
		tx, err := pool.Begin(ctx)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menginisialisasi transaksi database", err)
			return
		}
		defer func() {
			_ = tx.Rollback(ctx)
		}()

		qtx := q.WithTx(tx)

		if err := qtx.DeleteUserRolesByUserID(ctx, userID); err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menghapus peranan lama pengguna", err)
			return
		}

		for _, role := range dedupedRoles {
			err = qtx.InsertUserRole(ctx, dbgen.InsertUserRoleParams{
				UserID: userID,
				Role:   role,
			})
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23514" {
					middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Peranan tidak valid", err)
					return
				}
				middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menambahkan peranan pengguna baru", err)
				return
			}
		}

		if err := tx.Commit(ctx); err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menyimpan transaksi peranan pengguna", err)
			return
		}

		c.JSON(http.StatusOK, UserRolesResponse{
			ID:    userID,
			Roles: dedupedRoles,
		})
	}
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
