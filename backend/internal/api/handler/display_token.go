package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

type DisplayTokenHandler struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewDisplayTokenHandler(pool *pgxpool.Pool, q *dbgen.Queries) *DisplayTokenHandler {
	return &DisplayTokenHandler{
		pool: pool,
		q:    q,
	}
}

type RegenerateDisplayTokenResponse struct {
	DisplayToken string `json:"displayToken"`
}

// RegenerateDisplayToken menangani POST /admin/klinik/:id/display-token/regenerate [admin].
// Meng-generate token mentah baru, me-overwrite display_token_hash di DB dengan hash SHA256 baru,
// dan mengembalikan token mentah ke admin (token mentah hanya terlihat 1x ini).
func (h *DisplayTokenHandler) RegenerateDisplayToken(c *gin.Context) {
	idParam := c.Param("id")
	klinikIDParsed, err := strconv.Atoi(idParam)
	if err != nil || klinikIDParsed <= 0 {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID klinik tidak valid", err)
		return
	}
	klinikID := int32(klinikIDParsed)

	ctx := c.Request.Context()

	// Cek keberadaan klinik
	_, err = h.q.GetKlinikByID(ctx, klinikID)
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, "KLINIK_NOT_FOUND", "Klinik tidak ditemukan", err)
		return
	}

	// Generate raw token mentah & hash SHA256
	rawToken, err := auth.GenerateToken()
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal me-generate display token", err)
		return
	}
	hashedToken := auth.HashToken(rawToken)

	// Overwrite display_token_hash di DB
	_, err = h.q.UpdateKlinikDisplayTokenHash(ctx, dbgen.UpdateKlinikDisplayTokenHashParams{
		DisplayTokenHash: pgtype.Text{String: hashedToken, Valid: true},
		ID:               klinikID,
	})
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Gagal memperbarui display token di database", err)
		return
	}

	c.JSON(http.StatusOK, RegenerateDisplayTokenResponse{
		DisplayToken: rawToken,
	})
}
