package middleware

import (
	"crypto/subtle"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/danisetiawan31/klinik-rme/internal/auth"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

const (
	AuthChannelContextKey = "auth_channel"
	KlinikIDContextKey    = "klinik_id"
)

// DualAuth menyediakan skema autentikasi ganda untuk endpoint antrian / papan antrian (REST & WS).
// Alur evaluasi:
// 1. Cek cookie staff session ("session") -> jika valid (via validateStaffSession), authorized via channel "cookie" + user & roles ter-attach.
// 2. Jika cookie tidak ada/invalid -> cek header "X-Display-Token" (REST) atau query param "?displayToken=" (WS fallback).
// 3. Jika display token dikirim -> hash token, cocokkan dengan klinik.display_token_hash di DB via constant-time comparison.
// 4. Jika keduanya gagal -> 401 UNAUTHORIZED.
func DualAuth(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		if q == nil {
			RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Database queries unit is not initialized", nil)
			c.Abort()
			return
		}

		ctx := c.Request.Context()

		// A. Evaluasi Sesi Cookie Staff via Shared Helper validateStaffSession
		rawToken, err := c.Cookie(SessionCookieName)
		if err == nil && rawToken != "" {
			if user, roles, idHash, ok := validateStaffSession(c, q, rawToken); ok {
				c.Set(AuthChannelContextKey, "cookie")
				c.Set(UserContextKey, user)
				c.Set(RolesContextKey, roles)
				c.Set(SessionIDHashContextKey, idHash)
				c.Set(SessionRawTokenKey, rawToken)

				c.Next()
				return
			}
		}

		// B. Evaluasi Display Token (Papan Antrian Publik) jika Cookie tidak ada/invalid
		// Browser WebSocket API tidak dapat menyetel custom HTTP Header,
		// sehingga query param ?displayToken= dipakai sebagai fallback wajib untuk koneksi WS.
		displayToken := c.GetHeader("X-Display-Token")
		if displayToken == "" {
			displayToken = c.Query("displayToken")
		}

		if displayToken != "" {
			klinikParam := c.Param("id")
			if klinikParam == "" {
				klinikParam = c.Query("klinikId")
			}

			klinikIDParsed, err := strconv.Atoi(klinikParam)
			if err == nil && klinikIDParsed > 0 {
				klinikID := int32(klinikIDParsed)

				hashInDB, err := q.GetKlinikDisplayTokenHash(ctx, klinikID)
				// Jika display_token_hash di DB masih NULL (!Valid / kosong) -> SELALU gagal match
				if err == nil && hashInDB.Valid && hashInDB.String != "" {
					hashedToken := auth.HashToken(displayToken)
					// Constant-time compare untuk mencegah timing-attack pada hash token
					if subtle.ConstantTimeCompare([]byte(hashedToken), []byte(hashInDB.String)) == 1 {
						// Authorized via display-token (tanpa user / roles)
						c.Set(AuthChannelContextKey, "display-token")
						c.Set(KlinikIDContextKey, klinikID)

						c.Next()
						return
					}
				}
			}
		}

		// C. Kedua jalur gagal -> 401 UNAUTHORIZED
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi atau display token tidak ditemukan atau telah berakhir", nil)
		c.Abort()
	}
}

// GetAuthChannelFromContext mengambil nama channel autentikasi ("cookie" / "display-token") dari Gin context.
func GetAuthChannelFromContext(c *gin.Context) string {
	val, exists := c.Get(AuthChannelContextKey)
	if !exists {
		return ""
	}
	ch, ok := val.(string)
	if !ok {
		return ""
	}
	return ch
}
