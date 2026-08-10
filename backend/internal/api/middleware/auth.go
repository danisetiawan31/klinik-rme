package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

const (
	SessionCookieName = "session"

	UserContextKey          = "user"
	RolesContextKey         = "roles"
	SessionIDHashContextKey = "session_id_hash"
	SessionRawTokenKey      = "session_raw_token"
)

// Authenticate verifies the session cookie, checks expiry against DB, attaches user & roles to context,
// and extends the sliding session expiry by 11 hours on every valid request.
func Authenticate(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken, err := c.Cookie(SessionCookieName)
		if err != nil || rawToken == "" {
			RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak ditemukan atau telah berakhir", nil)
			c.Abort()
			return
		}

		idHash := auth.HashToken(rawToken)
		ctx := c.Request.Context()

		sess, err := q.GetSessionByIDHash(ctx, idHash)
		if err != nil {
			RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak ditemukan atau telah berakhir", err)
			c.Abort()
			return
		}

		now := time.Now()
		// Validate against sliding expiration and hard cap absolute expiration
		if now.After(sess.ExpiresAt.Time) || now.Equal(sess.ExpiresAt.Time) ||
			now.After(sess.AbsoluteExpiresAt.Time) || now.Equal(sess.AbsoluteExpiresAt.Time) {
			RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak ditemukan atau telah berakhir", nil)
			c.Abort()
			return
		}

		user, err := q.GetUserByID(ctx, sess.UserID)
		if err != nil {
			RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Pengguna tidak ditemukan", err)
			c.Abort()
			return
		}

		roles, err := q.GetRolesByUserID(ctx, sess.UserID)
		if err != nil {
			RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Gagal membaca peranan pengguna", err)
			c.Abort()
			return
		}
		if roles == nil {
			roles = []string{}
		}

		// Attach user, roles, and session info to request context
		c.Set(UserContextKey, user)
		c.Set(RolesContextKey, roles)
		c.Set(SessionIDHashContextKey, idHash)
		c.Set(SessionRawTokenKey, rawToken)

		// Extend sliding session duration by 2 hours for every active valid request
		newExpiresAt := now.Add(2 * time.Hour)
		_ = q.UpdateSessionExpiresAt(ctx, dbgen.UpdateSessionExpiresAtParams{
			IDHash: idHash,
			ExpiresAt: pgtype.Timestamptz{
				Time:  newExpiresAt,
				Valid: true,
			},
		})

		c.Next()
	}
}

// GetUserFromContext extracts the authenticated dbgen.User from Gin context.
func GetUserFromContext(c *gin.Context) (dbgen.User, bool) {
	val, exists := c.Get(UserContextKey)
	if !exists {
		return dbgen.User{}, false
	}
	user, ok := val.(dbgen.User)
	return user, ok
}

// GetRolesFromContext extracts the user roles slice from Gin context.
func GetRolesFromContext(c *gin.Context) ([]string, bool) {
	val, exists := c.Get(RolesContextKey)
	if !exists {
		return nil, false
	}
	roles, ok := val.([]string)
	return roles, ok
}

// GetSessionIDHashFromContext extracts the session ID hash from Gin context.
func GetSessionIDHashFromContext(c *gin.Context) (string, bool) {
	val, exists := c.Get(SessionIDHashContextKey)
	if !exists {
		return "", false
	}
	idHash, ok := val.(string)
	return idHash, ok
}
