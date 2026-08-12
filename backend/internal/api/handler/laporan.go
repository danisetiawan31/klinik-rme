package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

type LaporanHarianResponse struct {
	Tanggal         string `json:"tanggal"`
	TotalKunjungan  int32  `json:"totalKunjungan"`
	TotalSelesai    int32  `json:"totalSelesai"`
	TotalTidakHadir int32  `json:"totalTidakHadir"`
}

// GetLaporanHarian handles GET /api/v1/laporan/harian [petugas, dokter, admin]
func GetLaporanHarian(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tanggalParam := c.Query("tanggal")
		var targetTime time.Time
		var tanggalStr string

		if tanggalParam == "" {
			loc, err := time.LoadLocation("Asia/Jakarta")
			if err != nil {
				loc = time.Local
			}
			targetTime = time.Now().In(loc)
			tanggalStr = targetTime.Format("2006-01-02")
		} else {
			parsedTime, err := time.Parse("2006-01-02", tanggalParam)
			if err != nil {
				middleware.RespondError(c, http.StatusBadRequest, "TANGGAL_INVALID", "Format tanggal tidak valid. Gunakan format YYYY-MM-DD (contoh: 2026-08-12).", err)
				return
			}
			targetTime = parsedTime
			tanggalStr = tanggalParam
		}

		ctx := c.Request.Context()
		klinik, err := q.GetSingleKlinik(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusInternalServerError, "KLINIK_NOT_CONFIGURED", "Klinik belum terkonfigurasi di sistem", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil data klinik", err)
			return
		}

		row, err := q.GetLaporanHarian(ctx, dbgen.GetLaporanHarianParams{
			KlinikID: klinik.ID,
			TanggalKunjungan: pgtype.Date{
				Time:  targetTime,
				Valid: true,
			},
		})
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal mengambil data laporan harian", err)
			return
		}

		c.JSON(http.StatusOK, LaporanHarianResponse{
			Tanggal:         tanggalStr,
			TotalKunjungan:  row.TotalKunjungan,
			TotalSelesai:    row.TotalSelesai,
			TotalTidakHadir: row.TotalTidakHadir,
		})
	}
}
