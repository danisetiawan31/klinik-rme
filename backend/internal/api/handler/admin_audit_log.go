package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

type AuditLogSummaryResponse struct {
	ID          int32  `json:"id"`
	TabelTarget string `json:"tabelTarget"`
	RecordID    int32  `json:"recordId"`
	ActorUserID int32  `json:"actorUserId"`
	Aksi        string `json:"aksi"`
	CreatedAt   string `json:"createdAt"`
}

type AuditLogDetailResponse struct {
	ID          int32           `json:"id"`
	TabelTarget string          `json:"tabelTarget"`
	RecordID    int32           `json:"recordId"`
	ActorUserID int32           `json:"actorUserId"`
	Aksi        string          `json:"aksi"`
	BeforeData  json.RawMessage `json:"beforeData"`
	AfterData   json.RawMessage `json:"afterData"`
	HashEntry   string          `json:"hashEntry"`
	CreatedAt   string          `json:"createdAt"`
}

// ListAuditLogs handles GET /api/v1/admin/audit-log?tabelTarget=&recordId=&actorId=&page=&limit= [admin]
func ListAuditLogs(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		tabelTarget := c.Query("tabelTarget")
		recordIdStr := c.Query("recordId")
		actorIdStr := c.Query("actorId")
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

		var tabelTargetText pgtype.Text
		if tabelTarget != "" {
			tabelTargetText = pgtype.Text{String: tabelTarget, Valid: true}
		}

		var recordIDInt pgtype.Int4
		if recordIdStr != "" {
			if rID, err := strconv.Atoi(recordIdStr); err == nil && rID > 0 {
				recordIDInt = pgtype.Int4{Int32: int32(rID), Valid: true}
			}
		}

		var actorIDInt pgtype.Int4
		if actorIdStr != "" {
			if aID, err := strconv.Atoi(actorIdStr); err == nil && aID > 0 {
				actorIDInt = pgtype.Int4{Int32: int32(aID), Valid: true}
			}
		}

		ctx := c.Request.Context()
		rows, err := q.ListAuditLogs(ctx, dbgen.ListAuditLogsParams{
			Limit:       int32(limit),
			Offset:      int32(offset),
			TabelTarget: tabelTargetText,
			RecordID:    recordIDInt,
			ActorUserID: actorIDInt,
		})
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengambil daftar audit log", err)
			return
		}

		var totalCount int64 = 0
		if len(rows) > 0 {
			totalCount = rows[0].TotalCount
		}
		c.Header("X-Total-Count", strconv.FormatInt(totalCount, 10))

		res := make([]AuditLogSummaryResponse, 0, len(rows))
		for _, row := range rows {
			createdAtStr := ""
			if row.CreatedAt.Valid {
				createdAtStr = row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
			}
			res = append(res, AuditLogSummaryResponse{
				ID:          row.ID,
				TabelTarget: row.TabelTarget,
				RecordID:    row.RecordID,
				ActorUserID: row.ActorUserID,
				Aksi:        row.Aksi,
				CreatedAt:   createdAtStr,
			})
		}

		c.JSON(http.StatusOK, res)
	}
}

// GetAuditLogByID handles GET /api/v1/admin/audit-log/:id [admin]
func GetAuditLogByID(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		idParsed, err := strconv.Atoi(idParam)
		if err != nil || idParsed <= 0 {
			middleware.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "ID audit log tidak valid", err)
			return
		}
		logID := int32(idParsed)

		ctx := c.Request.Context()
		row, err := q.GetAuditLogByID(ctx, logID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				middleware.RespondError(c, http.StatusNotFound, "AUDIT_LOG_NOT_FOUND", "Audit log tidak ditemukan", err)
				return
			}
			middleware.RespondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Gagal mengambil detail audit log", err)
			return
		}

		var beforeDataJSON json.RawMessage
		if len(row.BeforeData) > 0 {
			beforeDataJSON = json.RawMessage(row.BeforeData)
		}

		var afterDataJSON json.RawMessage
		if len(row.AfterData) > 0 {
			afterDataJSON = json.RawMessage(row.AfterData)
		}

		createdAtStr := ""
		if row.CreatedAt.Valid {
			createdAtStr = row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}

		c.JSON(http.StatusOK, AuditLogDetailResponse{
			ID:          row.ID,
			TabelTarget: row.TabelTarget,
			RecordID:    row.RecordID,
			ActorUserID: row.ActorUserID,
			Aksi:        row.Aksi,
			BeforeData:  beforeDataJSON,
			AfterData:   afterDataJSON,
			HashEntry:   row.HashEntry,
			CreatedAt:   createdAtStr,
		})
	}
}
