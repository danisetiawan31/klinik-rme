package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	"github.com/danisetiawan31/klinik-rme/internal/audit"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

type CreatePasienRequest struct {
	Nik          *string `json:"nik"`
	Nama         string  `json:"nama" binding:"required"`
	TanggalLahir string  `json:"tanggalLahir" binding:"required"`
	JenisKelamin string  `json:"jenisKelamin" binding:"required"`
	Alamat       string  `json:"alamat" binding:"required"`
	NoTelp       string  `json:"noTelp" binding:"required"`
	Consent      *bool   `json:"consent" binding:"required"`
}

type UpdatePasienRequest struct {
	Version      int32   `json:"version" binding:"required"`
	Nik          *string `json:"nik"`
	Nama         *string `json:"nama"`
	TanggalLahir *string `json:"tanggalLahir"`
	JenisKelamin *string `json:"jenisKelamin"`
	Alamat       *string `json:"alamat"`
	NoTelp       *string `json:"noTelp"`
}

type PasienResponse struct {
	ID           int32      `json:"id"`
	Nik          *string    `json:"nik"`
	Nama         string     `json:"nama"`
	TanggalLahir string     `json:"tanggalLahir"`
	JenisKelamin string     `json:"jenisKelamin"`
	Alamat       string     `json:"alamat"`
	NoTelp       string     `json:"noTelp"`
	ConsentAt    time.Time  `json:"consentAt"`
	Version      int32      `json:"version"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}

type PasienDetailResponse struct {
	PasienResponse
	RiwayatKunjunganRingkas []interface{} `json:"riwayatKunjunganRingkas"`
}

type PasienSearchItem struct {
	ID           int32   `json:"id"`
	Nik          *string `json:"nik"`
	Nama         string  `json:"nama"`
	TanggalLahir string  `json:"tanggalLahir"`
}

type auditPasienSnapshotCreate struct {
	Nik          *string `json:"nik"`
	Nama         string  `json:"nama"`
	TanggalLahir string  `json:"tanggalLahir"`
	JenisKelamin string  `json:"jenisKelamin"`
	Alamat       string  `json:"alamat"`
	NoTelp       string  `json:"noTelp"`
	ConsentAt    string  `json:"consentAt"`
}

type auditPasienSnapshotUpdate struct {
	Nik          *string `json:"nik"`
	Nama         string  `json:"nama"`
	TanggalLahir string  `json:"tanggalLahir"`
	JenisKelamin string  `json:"jenisKelamin"`
	Alamat       string  `json:"alamat"`
	NoTelp       string  `json:"noTelp"`
}

func textToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func formatPgDate(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

func mapPasienToResponse(p dbgen.Pasien) PasienResponse {
	var deletedAt *time.Time
	if p.DeletedAt.Valid {
		deletedAt = &p.DeletedAt.Time
	}
	return PasienResponse{
		ID:           p.ID,
		Nik:          textToPtr(p.Nik),
		Nama:         p.Nama,
		TanggalLahir: formatPgDate(p.TanggalLahir),
		JenisKelamin: p.JenisKelamin,
		Alamat:       p.Alamat,
		NoTelp:       p.NoTelp,
		ConsentAt:    p.ConsentAt.Time,
		Version:      p.Version,
		DeletedAt:    deletedAt,
	}
}

// CreatePasien handles POST /api/v1/pasien [petugas, admin]
func CreatePasien(pool *pgxpool.Pool, q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreatePasienRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format input tidak valid", err)
			return
		}

		if req.Consent == nil || !*req.Consent {
			middleware.RespondError(c, http.StatusBadRequest, "CONSENT_REQUIRED", "Persetujuan consent wajib disetujui", nil)
			return
		}

		dob, err := time.Parse("2006-01-02", req.TanggalLahir)
		if err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format tanggal lahir harus YYYY-MM-DD", err)
			return
		}

		if req.JenisKelamin != "L" && req.JenisKelamin != "P" {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Jenis kelamin harus 'L' atau 'P'", nil)
			return
		}

		user, ok := middleware.GetUserFromContext(c)
		if !ok {
			middleware.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak ditemukan atau telah berakhir", nil)
			return
		}

		ctx := c.Request.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menginisialisasi transaksi database", err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		qtx := q.WithTx(tx)
		consentAt := time.Now()

		var nikText pgtype.Text
		if req.Nik != nil {
			nikText = pgtype.Text{String: *req.Nik, Valid: true}
		}

		newPasien, err := qtx.InsertPasien(ctx, dbgen.InsertPasienParams{
			Nik:          nikText,
			Nama:         req.Nama,
			TanggalLahir: pgtype.Date{Time: dob, Valid: true},
			JenisKelamin: req.JenisKelamin,
			Alamat:       req.Alamat,
			NoTelp:       req.NoTelp,
			ConsentAt:    pgtype.Timestamptz{Time: consentAt, Valid: true},
		})
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menambahkan data pasien", err)
			return
		}

		snapshotAfter := auditPasienSnapshotCreate{
			Nik:          req.Nik,
			Nama:         newPasien.Nama,
			TanggalLahir: formatPgDate(newPasien.TanggalLahir),
			JenisKelamin: newPasien.JenisKelamin,
			Alamat:       newPasien.Alamat,
			NoTelp:       newPasien.NoTelp,
			ConsentAt:    consentAt.Format(time.RFC3339),
		}
		afterDataJSON, err := json.Marshal(snapshotAfter)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengolah snapshot audit", err)
			return
		}

		err = audit.Record(ctx, tx, q, user.ID, "pasien", newPasien.ID, "create", nil, afterDataJSON)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mencatat audit log", err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menyimpan transaksi pasien", err)
			return
		}

		c.JSON(http.StatusCreated, mapPasienToResponse(newPasien))
	}
}

// SearchPasien handles GET /api/v1/pasien/search [petugas, dokter, admin]
func SearchPasien(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		nik := c.Query("nik")
		nama := c.Query("nama")
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

		var nikText pgtype.Text
		if nik != "" {
			nikText = pgtype.Text{String: nik, Valid: true}
		}

		var namaText pgtype.Text
		if nama != "" {
			namaText = pgtype.Text{String: nama, Valid: true}
		}

		ctx := c.Request.Context()
		results, err := q.SearchPasien(ctx, dbgen.SearchPasienParams{
			Limit:  int32(limit),
			Offset: int32(offset),
			Nik:    nikText,
			Nama:   namaText,
		})
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mencari data pasien", err)
			return
		}

		var totalCount int64 = 0
		if len(results) > 0 {
			totalCount = results[0].TotalCount
		}
		c.Header("X-Total-Count", strconv.FormatInt(totalCount, 10))

		items := make([]PasienSearchItem, 0, len(results))
		for _, p := range results {
			items = append(items, PasienSearchItem{
				ID:           p.ID,
				Nik:          textToPtr(p.Nik),
				Nama:         p.Nama,
				TanggalLahir: formatPgDate(p.TanggalLahir),
			})
		}

		c.JSON(http.StatusOK, items)
	}
}

// GetPasienByID handles GET /api/v1/pasien/:id [petugas, dokter, admin]
func GetPasienByID(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil || id <= 0 {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "ID pasien tidak valid", nil)
			return
		}

		ctx := c.Request.Context()
		p, err := q.GetPasienByID(ctx, int32(id))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "PASIEN_NOT_FOUND", "Pasien tidak ditemukan", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengambil data pasien", err)
			return
		}

		c.JSON(http.StatusOK, PasienDetailResponse{
			PasienResponse:          mapPasienToResponse(p),
			RiwayatKunjunganRingkas: []interface{}{},
		})
	}
}

// UpdatePasien handles PATCH /api/v1/pasien/:id [petugas, admin]
func UpdatePasien(pool *pgxpool.Pool, q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		id, err := strconv.Atoi(idParam)
		if err != nil || id <= 0 {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "ID pasien tidak valid", nil)
			return
		}

		var req UpdatePasienRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format input tidak valid", err)
			return
		}

		ctx := c.Request.Context()

		// 1. Pre-fetch existing patient record for beforeData snapshot & existence check
		existing, err := q.GetPasienByID(ctx, int32(id))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "PASIEN_NOT_FOUND", "Pasien tidak ditemukan", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal membaca data pasien", err)
			return
		}

		// Prepare beforeData snapshot
		beforeSnapshot := auditPasienSnapshotUpdate{
			Nik:          textToPtr(existing.Nik),
			Nama:         existing.Nama,
			TanggalLahir: formatPgDate(existing.TanggalLahir),
			JenisKelamin: existing.JenisKelamin,
			Alamat:       existing.Alamat,
			NoTelp:       existing.NoTelp,
		}
		beforeDataJSON, err := json.Marshal(beforeSnapshot)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengolah snapshot audit", err)
			return
		}

		// Prepare update params
		var nikText pgtype.Text
		if req.Nik != nil {
			nikText = pgtype.Text{String: *req.Nik, Valid: true}
		}

		var namaText pgtype.Text
		if req.Nama != nil {
			namaText = pgtype.Text{String: *req.Nama, Valid: true}
		}

		var dobDate pgtype.Date
		if req.TanggalLahir != nil {
			dob, err := time.Parse("2006-01-02", *req.TanggalLahir)
			if err != nil {
				middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Format tanggal lahir harus YYYY-MM-DD", err)
				return
			}
			dobDate = pgtype.Date{Time: dob, Valid: true}
		}

		var jkText pgtype.Text
		if req.JenisKelamin != nil {
			if *req.JenisKelamin != "L" && *req.JenisKelamin != "P" {
				middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "Jenis kelamin harus 'L' atau 'P'", nil)
				return
			}
			jkText = pgtype.Text{String: *req.JenisKelamin, Valid: true}
		}

		var alamatText pgtype.Text
		if req.Alamat != nil {
			alamatText = pgtype.Text{String: *req.Alamat, Valid: true}
		}

		var noTelpText pgtype.Text
		if req.NoTelp != nil {
			noTelpText = pgtype.Text{String: *req.NoTelp, Valid: true}
		}

		user, ok := middleware.GetUserFromContext(c)
		if !ok {
			middleware.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak ditemukan atau telah berakhir", nil)
			return
		}

		// 2. Open transaction & attempt optimistic update
		tx, err := pool.Begin(ctx)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menginisialisasi transaksi database", err)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		qtx := q.WithTx(tx)

		updatedPasien, err := qtx.UpdatePasienOptimistic(ctx, dbgen.UpdatePasienOptimisticParams{
			ID:           int32(id),
			Version:      req.Version,
			Nik:          nikText,
			Nama:         namaText,
			TanggalLahir: dobDate,
			JenisKelamin: jkText,
			Alamat:       alamatText,
			NoTelp:       noTelpText,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Disambiguate: check if patient was deleted or doesn't exist vs version mismatch
				checkPasien, checkErr := q.GetPasienByIDIncludingDeleted(ctx, int32(id))
				if checkErr != nil || checkPasien.DeletedAt.Valid {
					middleware.RespondError(c, http.StatusNotFound, "PASIEN_NOT_FOUND", "Pasien tidak ditemukan", checkErr)
					return
				}
				middleware.RespondError(c, http.StatusConflict, "OPTIMISTIC_LOCK_FAILED", "Data pasien telah diubah oleh pengguna lain", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal memperbarui data pasien", err)
			return
		}

		// Prepare afterData snapshot
		afterSnapshot := auditPasienSnapshotUpdate{
			Nik:          textToPtr(updatedPasien.Nik),
			Nama:         updatedPasien.Nama,
			TanggalLahir: formatPgDate(updatedPasien.TanggalLahir),
			JenisKelamin: updatedPasien.JenisKelamin,
			Alamat:       updatedPasien.Alamat,
			NoTelp:       updatedPasien.NoTelp,
		}
		afterDataJSON, err := json.Marshal(afterSnapshot)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengolah snapshot audit", err)
			return
		}

		// Record audit log
		err = audit.Record(ctx, tx, q, user.ID, "pasien", updatedPasien.ID, "update", beforeDataJSON, afterDataJSON)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mencatat audit log", err)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal menyimpan transaksi pasien", err)
			return
		}

		c.JSON(http.StatusOK, mapPasienToResponse(updatedPasien))
	}
}
