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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

type KlinikAntrianHandler struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewKlinikAntrianHandler(pool *pgxpool.Pool) *KlinikAntrianHandler {
	return &KlinikAntrianHandler{
		pool: pool,
		q:    dbgen.New(pool),
	}
}

type KlinikResponse struct {
	ID       int32  `json:"id"`
	Nama     string `json:"nama"`
	JamBuka  string `json:"jamBuka"`
	JamTutup string `json:"jamTutup"`
}

type CreateKunjunganRequest struct {
	PasienID       int32   `json:"pasienId" binding:"required,gt=0"`
	IsPriority     *bool   `json:"isPriority"`
	PriorityReason *string `json:"priorityReason"`
}

type CreateKunjunganResponse struct {
	ID               int32  `json:"id"`
	NomorAntrian     int32  `json:"nomorAntrian"`
	Status           string `json:"status"`
	TanggalKunjungan string `json:"tanggalKunjungan"`
}

type GetKunjunganResponse struct {
	ID           int32   `json:"id"`
	PasienID     int32   `json:"pasienId"`
	NomorAntrian int32   `json:"nomorAntrian"`
	Status       string  `json:"status"`
	IsPriority   bool    `json:"isPriority"`
	DokterID     *int32  `json:"dokterId"`
	DipanggilAt  *string `json:"dipanggilAt"`
}

type AntrianItemResponse struct {
	ID           int32  `json:"id"`
	NomorAntrian int32  `json:"nomorAntrian"`
	Status       string `json:"status"`
	IsPriority   bool   `json:"isPriority"`
	PasienNama   string `json:"pasienNama"`
}

type AntrianItemPublicResponse struct {
	NomorAntrian int32  `json:"nomorAntrian"`
	Status       string `json:"status"`
	IsPriority   bool   `json:"isPriority"`
}

func formatPgTimeHHMM(t pgtype.Time) string {
	if !t.Valid {
		return "00:00"
	}
	totalSeconds := t.Microseconds / 1_000_000
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

// GetKlinikByID handles GET /api/v1/klinik/:id [petugas, dokter, admin]
func (h *KlinikAntrianHandler) GetKlinikByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID klinik tidak valid", err)
		return
	}

	klinik, err := h.q.GetKlinikByID(c.Request.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			middleware.RespondError(c, http.StatusNotFound, "KLINIK_NOT_FOUND", "Klinik tidak ditemukan", err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil data klinik", err)
		return
	}

	c.JSON(http.StatusOK, KlinikResponse{
		ID:       klinik.ID,
		Nama:     klinik.Nama,
		JamBuka:  formatPgTimeHHMM(klinik.JamBuka),
		JamTutup: formatPgTimeHHMM(klinik.JamTutup),
	})
}

// CreateKunjungan handles POST /api/v1/kunjungan [petugas, admin]
func (h *KlinikAntrianHandler) CreateKunjungan(c *gin.Context) {
	var req CreateKunjunganRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_BODY", "Body request tidak valid", err)
		return
	}

	klinik, err := h.q.GetSingleKlinik(c.Request.Context())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("[CREATE_KUNJUNGAN_ERROR] Klinik table is empty!");
			middleware.RespondError(c, http.StatusInternalServerError, "KLINIK_NOT_CONFIGURED", "Klinik belum terkonfigurasi di sistem", err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil data klinik", err)
		return
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.Local
	}
	now := time.Now().In(loc)

	nowMicroseconds := int64(now.Hour()*3600+now.Minute()*60+now.Second()) * 1_000_000
	if klinik.JamTutup.Valid && nowMicroseconds > klinik.JamTutup.Microseconds {
		middleware.RespondError(c, http.StatusBadRequest, "KLINIK_TUTUP", "Pendaftaran antrian sudah ditutup untuk hari ini", nil)
		return
	}

	pasien, err := h.q.GetPasienByID(c.Request.Context(), req.PasienID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			middleware.RespondError(c, http.StatusNotFound, "PASIEN_NOT_FOUND", "Pasien tidak ditemukan", err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal memverifikasi data pasien", err)
		return
	}

	todayDate := pgtype.Date{Time: now, Valid: true}
	nomorAntrian, err := h.q.UpsertQueueCounter(c.Request.Context(), dbgen.UpsertQueueCounterParams{
		KlinikID: klinik.ID,
		Tanggal:  todayDate,
	})
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal generate nomor antrian", err)
		return
	}

	isPriority := false
	if req.IsPriority != nil {
		isPriority = *req.IsPriority
	}
	var priorityReason pgtype.Text
	if req.PriorityReason != nil {
		priorityReason = pgtype.Text{String: *req.PriorityReason, Valid: true}
	}

	inserted, err := h.q.InsertKunjungan(c.Request.Context(), dbgen.InsertKunjunganParams{
		PasienID:         pasien.ID,
		KlinikID:         klinik.ID,
		DokterID:         pgtype.Int4{Valid: false},
		TanggalKunjungan: todayDate,
		NomorAntrian:     nomorAntrian,
		IsPriority:       isPriority,
		PriorityReason:   priorityReason,
		SkipCount:        0,
		Status:           "menunggu",
	})
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal membuat kunjungan", err)
		return
	}

	c.JSON(http.StatusCreated, CreateKunjunganResponse{
		ID:               inserted.ID,
		NomorAntrian:     inserted.NomorAntrian,
		Status:           inserted.Status,
		TanggalKunjungan: now.Format("2006-01-02"),
	})
}

// GetKunjunganByID handles GET /api/v1/kunjungan/:id [petugas, dokter, admin]
func (h *KlinikAntrianHandler) GetKunjunganByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID kunjungan tidak valid", err)
		return
	}

	kunjungan, err := h.q.GetKunjunganByID(c.Request.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			middleware.RespondError(c, http.StatusNotFound, "KUNJUNGAN_NOT_FOUND", "Kunjungan tidak ditemukan", err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil data kunjungan", err)
		return
	}

	var dokterID *int32
	if kunjungan.DokterID.Valid {
		dokterID = &kunjungan.DokterID.Int32
	}
	var dipanggilAt *string
	if kunjungan.DipanggilAt.Valid {
		formatted := kunjungan.DipanggilAt.Time.Format(time.RFC3339)
		dipanggilAt = &formatted
	}

	c.JSON(http.StatusOK, GetKunjunganResponse{
		ID:           kunjungan.ID,
		PasienID:     kunjungan.PasienID,
		NomorAntrian: kunjungan.NomorAntrian,
		Status:       kunjungan.Status,
		IsPriority:   kunjungan.IsPriority,
		DokterID:     dokterID,
		DipanggilAt:  dipanggilAt,
	})
}

// GetAntrianKlinik handles GET /api/v1/klinik/:id/antrian [petugas, dokter, admin]
func (h *KlinikAntrianHandler) GetAntrianKlinik(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID klinik tidak valid", err)
		return
	}

	_, err = h.q.GetKlinikByID(c.Request.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			middleware.RespondError(c, http.StatusNotFound, "KLINIK_NOT_FOUND", "Klinik tidak ditemukan", err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal memverifikasi klinik", err)
		return
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	todayDate := pgtype.Date{Time: now, Valid: true}

	rows, err := h.q.ListKunjunganWithPasienNamaByKlinikAndTanggal(c.Request.Context(), dbgen.ListKunjunganWithPasienNamaByKlinikAndTanggalParams{
		KlinikID:         int32(id),
		TanggalKunjungan: todayDate,
	})
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil daftar antrian", err)
		return
	}

	authChannel := middleware.GetAuthChannelFromContext(c)
	if authChannel == "display-token" {
		publicResp := make([]AntrianItemPublicResponse, 0, len(rows))
		for _, r := range rows {
			publicResp = append(publicResp, AntrianItemPublicResponse{
				NomorAntrian: r.NomorAntrian,
				Status:       r.Status,
				IsPriority:   r.IsPriority,
			})
		}
		c.JSON(http.StatusOK, publicResp)
		return
	}

	response := make([]AntrianItemResponse, 0, len(rows))
	for _, r := range rows {
		response = append(response, AntrianItemResponse{
			ID:           r.ID,
			NomorAntrian: r.NomorAntrian,
			Status:       r.Status,
			IsPriority:   r.IsPriority,
			PasienNama:   r.PasienNama,
		})
	}

	c.JSON(http.StatusOK, response)
}

type PanggilBerikutnyaResponse struct {
	ID           int32  `json:"id"`
	NomorAntrian int32  `json:"nomorAntrian"`
	PasienNama   string `json:"pasienNama"`
	DokterID     int32  `json:"dokterId"`
	DipanggilAt  string `json:"dipanggilAt"`
}

type UpdateSkipResponse struct {
	ID        int32  `json:"id"`
	Status    string `json:"status"`
	SkipCount int32  `json:"skipCount"`
}

type UpdateTidakHadirResponse struct {
	ID     int32  `json:"id"`
	Status string `json:"status"`
}

// PanggilBerikutnya handles POST /api/v1/klinik/:id/panggil-berikutnya [dokter]
func (h *KlinikAntrianHandler) PanggilBerikutnya(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID klinik tidak valid", err)
		return
	}

	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		middleware.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Pengguna tidak terautentikasi", nil)
		return
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.Local
	}
	now := time.Now().In(loc)
	todayDate := pgtype.Date{Time: now, Valid: true}

	claimed, err := h.q.ClaimNextKunjungan(c.Request.Context(), dbgen.ClaimNextKunjunganParams{
		DokterID:         pgtype.Int4{Int32: user.ID, Valid: true},
		KlinikID:         int32(id),
		TanggalKunjungan: todayDate,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Status(http.StatusNoContent)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal memanggil antrian berikutnya", err)
		return
	}

	pasien, err := h.q.GetPasienByID(c.Request.Context(), claimed.PasienID)
	if err != nil {
		pasien.Nama = ""
	}

	dipanggilAtStr := now.Format(time.RFC3339)
	if claimed.DipanggilAt.Valid {
		dipanggilAtStr = claimed.DipanggilAt.Time.Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, PanggilBerikutnyaResponse{
		ID:           claimed.ID,
		NomorAntrian: claimed.NomorAntrian,
		PasienNama:   pasien.Nama,
		DokterID:     user.ID,
		DipanggilAt:  dipanggilAtStr,
	})
}

// Lewati handles POST /api/v1/kunjungan/:id/lewati [dokter]
func (h *KlinikAntrianHandler) Lewati(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID kunjungan tidak valid", err)
		return
	}

	updated, err := h.q.UpdateKunjunganSkip(c.Request.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kunjungan, getErr := h.q.GetKunjunganByID(c.Request.Context(), int32(id))
			if getErr != nil && errors.Is(getErr, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "KUNJUNGAN_NOT_FOUND", "Kunjungan tidak ditemukan", getErr)
				return
			}
			middleware.RespondError(c, http.StatusConflict, "INVALID_KUNJUNGAN_STATUS", fmt.Sprintf("Status kunjungan (%s) bukan dipanggil", kunjungan.Status), err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal melewati kunjungan", err)
		return
	}

	c.JSON(http.StatusOK, UpdateSkipResponse{
		ID:        updated.ID,
		Status:    updated.Status,
		SkipCount: updated.SkipCount,
	})
}

// TidakHadir handles POST /api/v1/kunjungan/:id/tidak-hadir [dokter, admin]
func (h *KlinikAntrianHandler) TidakHadir(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID kunjungan tidak valid", err)
		return
	}

	updated, err := h.q.UpdateKunjunganTidakHadir(c.Request.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kunjungan, getErr := h.q.GetKunjunganByID(c.Request.Context(), int32(id))
			if getErr != nil && errors.Is(getErr, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "KUNJUNGAN_NOT_FOUND", "Kunjungan tidak ditemukan", getErr)
				return
			}
			middleware.RespondError(c, http.StatusConflict, "INVALID_KUNJUNGAN_STATUS", fmt.Sprintf("Status kunjungan (%s) sudah final", kunjungan.Status), err)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengubah status kunjungan menjadi tidak hadir", err)
		return
	}

	c.JSON(http.StatusOK, UpdateTidakHadirResponse{
		ID:     updated.ID,
		Status: updated.Status,
	})
}
