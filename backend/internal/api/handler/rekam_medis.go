package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	"github.com/danisetiawan31/klinik-rme/internal/audit"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/realtime"
)

type CreateDiagnosisItemRequest struct {
	KodeIcd   *string `json:"kodeIcd"`
	Deskripsi string  `json:"deskripsi"`
}

type CreateTindakanItemRequest struct {
	Jenis     string `json:"jenis"`
	Deskripsi string `json:"deskripsi"`
}

type CreateRekamMedisRequest struct {
	Keluhan          string                       `json:"keluhan"`
	HasilPemeriksaan string                       `json:"hasilPemeriksaan"`
	Diagnosis        []CreateDiagnosisItemRequest `json:"diagnosis"`
	Tindakan         *[]CreateTindakanItemRequest `json:"tindakan"`
}

type CreateAddendumRequest struct {
	AlasanAddendum   string                        `json:"alasanAddendum"`
	Keluhan          *string                       `json:"keluhan"`
	HasilPemeriksaan *string                       `json:"hasilPemeriksaan"`
	Diagnosis        *[]CreateDiagnosisItemRequest `json:"diagnosis"`
	Tindakan         *[]CreateTindakanItemRequest  `json:"tindakan"`
}

type DiagnosisResponseItem struct {
	ID        int32   `json:"id"`
	KodeIcd   *string `json:"kodeIcd"`
	Deskripsi string  `json:"deskripsi"`
}

type TindakanResponseItem struct {
	ID        int32  `json:"id"`
	Jenis     string `json:"jenis"`
	Deskripsi string `json:"deskripsi"`
}

type RekamMedisResponse struct {
	ID               int32                   `json:"id"`
	Keluhan          string                  `json:"keluhan"`
	HasilPemeriksaan string                  `json:"hasilPemeriksaan"`
	Diagnosis        []DiagnosisResponseItem `json:"diagnosis"`
	Tindakan         []TindakanResponseItem  `json:"tindakan"`
	CreatedAt        string                  `json:"createdAt"`
}

type AddendumResponse struct {
	ID               int32                   `json:"id"`
	AddendumOf       int32                   `json:"addendumOf"`
	Keluhan          string                  `json:"keluhan"`
	HasilPemeriksaan string                  `json:"hasilPemeriksaan"`
	Diagnosis        []DiagnosisResponseItem `json:"diagnosis"`
	Tindakan         []TindakanResponseItem  `json:"tindakan"`
	CreatedAt        string                  `json:"createdAt"`
}

// CreateRekamMedisAwal handles POST /api/v1/kunjungan/:id/rekam-medis [dokter]
func CreateRekamMedisAwal(pool *pgxpool.Pool, q *dbgen.Queries, hub *realtime.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		kunjunganID, err := strconv.Atoi(idParam)
		if err != nil || kunjunganID <= 0 {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID kunjungan tidak valid", err)
			return
		}

		var req CreateRekamMedisRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Format payload request tidak valid", err)
			return
		}

		// Validasi input
		if strings.TrimSpace(req.Keluhan) == "" || strings.TrimSpace(req.HasilPemeriksaan) == "" {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Keluhan dan hasil pemeriksaan wajib diisi", nil)
			return
		}

		if req.Diagnosis == nil || len(req.Diagnosis) == 0 {
			middleware.RespondError(c, http.StatusBadRequest, "DIAGNOSIS_REQUIRED", "Diagnosis wajib diisi minimal 1 item", nil)
			return
		}

		for _, d := range req.Diagnosis {
			if strings.TrimSpace(d.Deskripsi) == "" {
				middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Deskripsi diagnosis wajib diisi", nil)
				return
			}
		}

		if req.Tindakan == nil {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Field tindakan wajib ada di request", nil)
			return
		}

		for _, tItem := range *req.Tindakan {
			if tItem.Jenis != "tindakan" && tItem.Jenis != "resep" {
				middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Jenis tindakan harus 'tindakan' atau 'resep'", nil)
				return
			}
			if strings.TrimSpace(tItem.Deskripsi) == "" {
				middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Deskripsi tindakan wajib diisi", nil)
				return
			}
		}

		// Dokter ID dari session context
		user, ok := middleware.GetUserFromContext(c)
		if !ok {
			middleware.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak ditemukan atau telah berakhir", nil)
			return
		}
		dokterID := user.ID

		ctx := c.Request.Context()

		// Cek kunjungan exist
		kunjungan, err := q.GetKunjunganByID(ctx, int32(kunjunganID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "KUNJUNGAN_NOT_FOUND", "Kunjungan tidak ditemukan", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil data kunjungan", err)
			return
		}

		// Transaksi eksplisit DB
		tx, err := pool.Begin(ctx)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal memulai transaksi database", err)
			return
		}
		defer func() {
			_ = tx.Rollback(ctx)
		}()

		qtx := q.WithTx(tx)

		// a. InsertRekamMedis
		rmRow, err := qtx.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      int32(kunjunganID),
			DokterID:         dokterID,
			Keluhan:          strings.TrimSpace(req.Keluhan),
			HasilPemeriksaan: strings.TrimSpace(req.HasilPemeriksaan),
			IsAddendum:       false,
			AddendumOf:       pgtype.Int4{Valid: false},
			AlasanAddendum:   pgtype.Text{Valid: false},
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				middleware.RespondError(c, http.StatusConflict, "REKAM_MEDIS_ALREADY_EXISTS", "Rekam medis awal untuk kunjungan me-refer ke kunjungan ini sudah ada", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal menyimpan rekam medis", err)
			return
		}

		// b. Loop InsertDiagnosis
		diagResponses := make([]DiagnosisResponseItem, 0, len(req.Diagnosis))
		diagSnapshots := make([]map[string]interface{}, 0, len(req.Diagnosis))
		for _, d := range req.Diagnosis {
			var icdText pgtype.Text
			if d.KodeIcd != nil && strings.TrimSpace(*d.KodeIcd) != "" {
				icdText = pgtype.Text{String: strings.TrimSpace(*d.KodeIcd), Valid: true}
			}
			diagRow, err := qtx.InsertDiagnosis(ctx, dbgen.InsertDiagnosisParams{
				RekamMedisID: rmRow.ID,
				KodeIcd:      icdText,
				Deskripsi:    strings.TrimSpace(d.Deskripsi),
			})
			if err != nil {
				middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal menyimpan diagnosis", err)
				return
			}

			var icdPtr *string
			if diagRow.KodeIcd.Valid {
				icdPtr = &diagRow.KodeIcd.String
			}
			diagResponses = append(diagResponses, DiagnosisResponseItem{
				ID:        diagRow.ID,
				KodeIcd:   icdPtr,
				Deskripsi: diagRow.Deskripsi,
			})
			diagSnapshots = append(diagSnapshots, map[string]interface{}{
				"id":        diagRow.ID,
				"kodeIcd":   icdPtr,
				"deskripsi": diagRow.Deskripsi,
			})
		}

		// c. Loop InsertTindakan
		tindakanResponses := make([]TindakanResponseItem, 0, len(*req.Tindakan))
		tindakanSnapshots := make([]map[string]interface{}, 0, len(*req.Tindakan))
		for _, tItem := range *req.Tindakan {
			tRow, err := qtx.InsertTindakan(ctx, dbgen.InsertTindakanParams{
				RekamMedisID: rmRow.ID,
				Jenis:        tItem.Jenis,
				Deskripsi:    strings.TrimSpace(tItem.Deskripsi),
			})
			if err != nil {
				middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal menyimpan tindakan", err)
				return
			}
			tindakanResponses = append(tindakanResponses, TindakanResponseItem{
				ID:        tRow.ID,
				Jenis:     tRow.Jenis,
				Deskripsi: tRow.Deskripsi,
			})
			tindakanSnapshots = append(tindakanSnapshots, map[string]interface{}{
				"id":        tRow.ID,
				"jenis":     tRow.Jenis,
				"deskripsi": tRow.Deskripsi,
			})
		}

		// d. UpdateKunjunganSelesai
		_, err = qtx.UpdateKunjunganSelesai(ctx, int32(kunjunganID))
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengupdate status kunjungan", err)
			return
		}

		// e. Snapshot JSON afterData
		afterData := map[string]interface{}{
			"kunjunganId":      int32(kunjunganID),
			"dokterId":         dokterID,
			"keluhan":          rmRow.Keluhan,
			"hasilPemeriksaan": rmRow.HasilPemeriksaan,
			"diagnosis":        diagSnapshots,
			"tindakan":         tindakanSnapshots,
		}
		afterDataJSON, err := json.Marshal(afterData)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal membuat snapshot audit log", err)
			return
		}

		// f. audit.Record(aksi="create")
		if err := audit.Record(ctx, tx, qtx, dokterID, "rekam_medis", rmRow.ID, "create", nil, afterDataJSON); err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mencatat audit log", err)
			return
		}

		// g. Commit
		if err := tx.Commit(ctx); err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal commit transaksi", err)
			return
		}

		if hub != nil {
			hub.BroadcastToKlinik(kunjungan.KlinikID)
		}

		c.JSON(http.StatusCreated, RekamMedisResponse{
			ID:               rmRow.ID,
			Keluhan:          rmRow.Keluhan,
			HasilPemeriksaan: rmRow.HasilPemeriksaan,
			Diagnosis:        diagResponses,
			Tindakan:         tindakanResponses,
			CreatedAt:        rmRow.CreatedAt.Time.Format(time.RFC3339),
		})
	}
}

// CreateAddendum handles POST /api/v1/rekam-medis/:id/addendum [dokter]
func CreateAddendum(pool *pgxpool.Pool, q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		parentID, err := strconv.Atoi(idParam)
		if err != nil || parentID <= 0 {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID rekam medis tidak valid", err)
			return
		}

		var req CreateAddendumRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Format payload request tidak valid", err)
			return
		}

		if strings.TrimSpace(req.AlasanAddendum) == "" {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Alasan addendum wajib diisi", nil)
			return
		}

		ctx := c.Request.Context()

		// Dokter ID dari session context
		user, ok := middleware.GetUserFromContext(c)
		if !ok {
			middleware.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak ditemukan atau telah berakhir", nil)
			return
		}
		dokterID := user.ID

		// Fetch parent rekam_medis
		parentRM, err := q.GetRekamMedisByID(ctx, int32(parentID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "REKAM_MEDIS_NOT_FOUND", "Rekam medis tidak ditemukan", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil data rekam medis", err)
			return
		}

		// Fetch parent diagnosis & tindakan for carry-over & audit beforeData
		parentDiags, err := q.GetDiagnosisByRekamMedisID(ctx, parentRM.ID)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil diagnosis parent", err)
			return
		}

		parentTins, err := q.GetTindakanByRekamMedisID(ctx, parentRM.ID)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil tindakan parent", err)
			return
		}

		// MERGE DI BACKEND:
		// 1. Keluhan
		mergedKeluhan := parentRM.Keluhan
		if req.Keluhan != nil {
			mergedKeluhan = *req.Keluhan
		}

		// 2. Hasil Pemeriksaan
		mergedHasilPemeriksaan := parentRM.HasilPemeriksaan
		if req.HasilPemeriksaan != nil {
			mergedHasilPemeriksaan = *req.HasilPemeriksaan
		}

		// 3. Diagnosis
		var finalDiagnosis []CreateDiagnosisItemRequest
		if req.Diagnosis == nil {
			// Carry-over from parent
			finalDiagnosis = make([]CreateDiagnosisItemRequest, 0, len(parentDiags))
			for _, pd := range parentDiags {
				var icdPtr *string
				if pd.KodeIcd.Valid {
					icdPtr = &pd.KodeIcd.String
				}
				finalDiagnosis = append(finalDiagnosis, CreateDiagnosisItemRequest{
					KodeIcd:   icdPtr,
					Deskripsi: pd.Deskripsi,
				})
			}
		} else {
			// Override from request
			finalDiagnosis = *req.Diagnosis
		}

		// 4. Tindakan
		var finalTindakan []CreateTindakanItemRequest
		if req.Tindakan == nil {
			// Carry-over from parent
			finalTindakan = make([]CreateTindakanItemRequest, 0, len(parentTins))
			for _, pt := range parentTins {
				finalTindakan = append(finalTindakan, CreateTindakanItemRequest{
					Jenis:     pt.Jenis,
					Deskripsi: pt.Deskripsi,
				})
			}
		} else {
			// Override from request
			finalTindakan = *req.Tindakan
		}

		// Post-merge validation: diagnosis MUST be at least 1 item
		if len(finalDiagnosis) == 0 {
			middleware.RespondError(c, http.StatusBadRequest, "DIAGNOSIS_REQUIRED", "Hasil akhir diagnosis setelah merge tidak boleh kosong", nil)
			return
		}

		for _, d := range finalDiagnosis {
			if strings.TrimSpace(d.Deskripsi) == "" {
				middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Deskripsi diagnosis wajib diisi", nil)
				return
			}
		}

		for _, tItem := range finalTindakan {
			if tItem.Jenis != "tindakan" && tItem.Jenis != "resep" {
				middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Jenis tindakan harus 'tindakan' atau 'resep'", nil)
				return
			}
			if strings.TrimSpace(tItem.Deskripsi) == "" {
				middleware.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Deskripsi tindakan wajib diisi", nil)
				return
			}
		}

		// Transaksi eksplisit DB
		tx, err := pool.Begin(ctx)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal memulai transaksi database", err)
			return
		}
		defer func() {
			_ = tx.Rollback(ctx)
		}()

		qtx := q.WithTx(tx)

		// a. InsertRekamMedis (Addendum)
		newRM, err := qtx.InsertRekamMedis(ctx, dbgen.InsertRekamMedisParams{
			KunjunganID:      parentRM.KunjunganID,
			DokterID:         dokterID,
			Keluhan:          mergedKeluhan,
			HasilPemeriksaan: mergedHasilPemeriksaan,
			IsAddendum:       true,
			AddendumOf:       pgtype.Int4{Int32: parentRM.ID, Valid: true},
			AlasanAddendum:   pgtype.Text{String: strings.TrimSpace(req.AlasanAddendum), Valid: true},
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				middleware.RespondError(c, http.StatusConflict, "ADDENDUM_CONFLICT", "Versi rekam medis ini sudah tidak terkini, silakan refetch versi terbaru dan coba lagi", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal menyimpan addendum rekam medis", err)
			return
		}

		// b. Loop InsertDiagnosis
		diagResponses := make([]DiagnosisResponseItem, 0, len(finalDiagnosis))
		diagSnapshots := make([]map[string]interface{}, 0, len(finalDiagnosis))
		for _, d := range finalDiagnosis {
			var icdText pgtype.Text
			if d.KodeIcd != nil && strings.TrimSpace(*d.KodeIcd) != "" {
				icdText = pgtype.Text{String: strings.TrimSpace(*d.KodeIcd), Valid: true}
			}
			diagRow, err := qtx.InsertDiagnosis(ctx, dbgen.InsertDiagnosisParams{
				RekamMedisID: newRM.ID,
				KodeIcd:      icdText,
				Deskripsi:    strings.TrimSpace(d.Deskripsi),
			})
			if err != nil {
				middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal menyimpan diagnosis addendum", err)
				return
			}

			var icdPtr *string
			if diagRow.KodeIcd.Valid {
				icdPtr = &diagRow.KodeIcd.String
			}
			diagResponses = append(diagResponses, DiagnosisResponseItem{
				ID:        diagRow.ID,
				KodeIcd:   icdPtr,
				Deskripsi: diagRow.Deskripsi,
			})
			diagSnapshots = append(diagSnapshots, map[string]interface{}{
				"id":        diagRow.ID,
				"kodeIcd":   icdPtr,
				"deskripsi": diagRow.Deskripsi,
			})
		}

		// c. Loop InsertTindakan
		tindakanResponses := make([]TindakanResponseItem, 0, len(finalTindakan))
		tindakanSnapshots := make([]map[string]interface{}, 0, len(finalTindakan))
		for _, tItem := range finalTindakan {
			tRow, err := qtx.InsertTindakan(ctx, dbgen.InsertTindakanParams{
				RekamMedisID: newRM.ID,
				Jenis:        tItem.Jenis,
				Deskripsi:    strings.TrimSpace(tItem.Deskripsi),
			})
			if err != nil {
				middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal menyimpan tindakan addendum", err)
				return
			}
			tindakanResponses = append(tindakanResponses, TindakanResponseItem{
				ID:        tRow.ID,
				Jenis:     tRow.Jenis,
				Deskripsi: tRow.Deskripsi,
			})
			tindakanSnapshots = append(tindakanSnapshots, map[string]interface{}{
				"id":        tRow.ID,
				"jenis":     tRow.Jenis,
				"deskripsi": tRow.Deskripsi,
			})
		}

		// d. Build parent beforeData snapshot
		parentDiagSnapshots := make([]map[string]interface{}, 0, len(parentDiags))
		for _, pd := range parentDiags {
			var icdPtr *string
			if pd.KodeIcd.Valid {
				icdPtr = &pd.KodeIcd.String
			}
			parentDiagSnapshots = append(parentDiagSnapshots, map[string]interface{}{
				"id":        pd.ID,
				"kodeIcd":   icdPtr,
				"deskripsi": pd.Deskripsi,
			})
		}

		parentTinSnapshots := make([]map[string]interface{}, 0, len(parentTins))
		for _, pt := range parentTins {
			parentTinSnapshots = append(parentTinSnapshots, map[string]interface{}{
				"id":        pt.ID,
				"jenis":     pt.Jenis,
				"deskripsi": pt.Deskripsi,
			})
		}

		beforeData := map[string]interface{}{
			"id":               parentRM.ID,
			"kunjunganId":       parentRM.KunjunganID,
			"dokterId":          parentRM.DokterID,
			"keluhan":           parentRM.Keluhan,
			"hasilPemeriksaan":  parentRM.HasilPemeriksaan,
			"diagnosis":         parentDiagSnapshots,
			"tindakan":          parentTinSnapshots,
		}
		beforeDataJSON, err := json.Marshal(beforeData)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal membuat snapshot audit log parent", err)
			return
		}

		// e. Build afterData snapshot for new addendum record
		afterData := map[string]interface{}{
			"id":               newRM.ID,
			"kunjunganId":       parentRM.KunjunganID,
			"dokterId":          dokterID,
			"addendumOf":        parentRM.ID,
			"alasanAddendum":    strings.TrimSpace(req.AlasanAddendum),
			"keluhan":           newRM.Keluhan,
			"hasilPemeriksaan":  newRM.HasilPemeriksaan,
			"diagnosis":         diagSnapshots,
			"tindakan":          tindakanSnapshots,
		}
		afterDataJSON, err := json.Marshal(afterData)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal membuat snapshot audit log addendum", err)
			return
		}

		// f. audit.Record(aksi="addendum")
		if err := audit.Record(ctx, tx, qtx, dokterID, "rekam_medis", newRM.ID, "addendum", beforeDataJSON, afterDataJSON); err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mencatat audit log addendum", err)
			return
		}

		// g. Commit
		if err := tx.Commit(ctx); err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal commit transaksi addendum", err)
			return
		}

		c.JSON(http.StatusCreated, AddendumResponse{
			ID:               newRM.ID,
			AddendumOf:       parentRM.ID,
			Keluhan:          newRM.Keluhan,
			HasilPemeriksaan: newRM.HasilPemeriksaan,
			Diagnosis:        diagResponses,
			Tindakan:         tindakanResponses,
			CreatedAt:        newRM.CreatedAt.Time.Format(time.RFC3339),
		})
	}
}

type GetRekamMedisResponse struct {
	ID               int32                   `json:"id"`
	Keluhan          string                  `json:"keluhan"`
	HasilPemeriksaan string                  `json:"hasilPemeriksaan"`
	Diagnosis        []DiagnosisResponseItem `json:"diagnosis"`
	Tindakan         []TindakanResponseItem  `json:"tindakan"`
	IsAddendum       bool                    `json:"isAddendum"`
	CreatedAt        string                  `json:"createdAt"`
}

type RiwayatKunjunganItem struct {
	KunjunganID int32                 `json:"kunjunganId"`
	Tanggal     string                `json:"tanggal"`
	RekamMedis  GetRekamMedisResponse `json:"rekamMedis"`
}

// GetRekamMedisKunjungan handles GET /api/v1/kunjungan/:id/rekam-medis [dokter]
func GetRekamMedisKunjungan(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		kunjunganID, err := strconv.Atoi(idParam)
		if err != nil || kunjunganID <= 0 {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID kunjungan tidak valid", err)
			return
		}

		ctx := c.Request.Context()
		leafRM, err := q.GetLeafRekamMedisByKunjunganID(ctx, int32(kunjunganID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "REKAM_MEDIS_NOT_FOUND", "Rekam medis tidak ditemukan untuk kunjungan ini", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil rekam medis", err)
			return
		}

		diags, err := q.GetDiagnosisByRekamMedisID(ctx, leafRM.ID)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil diagnosis", err)
			return
		}

		tins, err := q.GetTindakanByRekamMedisID(ctx, leafRM.ID)
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil tindakan", err)
			return
		}

		diagResponses := make([]DiagnosisResponseItem, 0, len(diags))
		for _, d := range diags {
			var icdPtr *string
			if d.KodeIcd.Valid {
				icdPtr = &d.KodeIcd.String
			}
			diagResponses = append(diagResponses, DiagnosisResponseItem{
				ID:        d.ID,
				KodeIcd:   icdPtr,
				Deskripsi: d.Deskripsi,
			})
		}

		tindakanResponses := make([]TindakanResponseItem, 0, len(tins))
		for _, tItem := range tins {
			tindakanResponses = append(tindakanResponses, TindakanResponseItem{
				ID:        tItem.ID,
				Jenis:     tItem.Jenis,
				Deskripsi: tItem.Deskripsi,
			})
		}

		c.JSON(http.StatusOK, GetRekamMedisResponse{
			ID:               leafRM.ID,
			Keluhan:          leafRM.Keluhan,
			HasilPemeriksaan: leafRM.HasilPemeriksaan,
			Diagnosis:        diagResponses,
			Tindakan:         tindakanResponses,
			IsAddendum:       leafRM.IsAddendum,
			CreatedAt:        leafRM.CreatedAt.Time.Format(time.RFC3339),
		})
	}
}

// GetRiwayatRekamMedisPasien handles GET /api/v1/pasien/:id/riwayat [dokter]
func GetRiwayatRekamMedisPasien(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		pasienID, err := strconv.Atoi(idParam)
		if err != nil || pasienID <= 0 {
			middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID pasien tidak valid", err)
			return
		}

		ctx := c.Request.Context()

		// Cek pasien ada
		_, err = q.GetPasienByID(ctx, int32(pasienID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "PASIEN_NOT_FOUND", "Pasien tidak ditemukan", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil data pasien", err)
			return
		}

		rows, err := q.ListLeafRekamMedisWithKunjunganByPasienID(ctx, int32(pasienID))
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil riwayat rekam medis", err)
			return
		}

		result := make([]RiwayatKunjunganItem, 0, len(rows))
		for _, r := range rows {
			diags, err := q.GetDiagnosisByRekamMedisID(ctx, r.RekamMedisID)
			if err != nil {
				middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil diagnosis riwayat", err)
				return
			}

			tins, err := q.GetTindakanByRekamMedisID(ctx, r.RekamMedisID)
			if err != nil {
				middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil tindakan riwayat", err)
				return
			}

			diagResponses := make([]DiagnosisResponseItem, 0, len(diags))
			for _, d := range diags {
				var icdPtr *string
				if d.KodeIcd.Valid {
					icdPtr = &d.KodeIcd.String
				}
				diagResponses = append(diagResponses, DiagnosisResponseItem{
					ID:        d.ID,
					KodeIcd:   icdPtr,
					Deskripsi: d.Deskripsi,
				})
			}

			tindakanResponses := make([]TindakanResponseItem, 0, len(tins))
			for _, tItem := range tins {
				tindakanResponses = append(tindakanResponses, TindakanResponseItem{
					ID:        tItem.ID,
					Jenis:     tItem.Jenis,
					Deskripsi: tItem.Deskripsi,
				})
			}

			tglStr := ""
			if r.TanggalKunjungan.Valid {
				tglStr = r.TanggalKunjungan.Time.Format("2006-01-02")
			}

			createdAtStr := ""
			if r.RekamMedisCreatedAt.Valid {
				createdAtStr = r.RekamMedisCreatedAt.Time.Format(time.RFC3339)
			}

			result = append(result, RiwayatKunjunganItem{
				KunjunganID: r.KunjunganID,
				Tanggal:     tglStr,
				RekamMedis: GetRekamMedisResponse{
					ID:               r.RekamMedisID,
					Keluhan:          r.Keluhan,
					HasilPemeriksaan: r.HasilPemeriksaan,
					Diagnosis:        diagResponses,
					Tindakan:         tindakanResponses,
					IsAddendum:       r.IsAddendum,
					CreatedAt:        createdAtStr,
				},
			})
		}

		c.JSON(http.StatusOK, result)
	}
}

