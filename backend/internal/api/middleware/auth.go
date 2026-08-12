package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/danisetiawan31/klinik-rme/internal/auth"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
)

const (
	SessionCookieName = "session"

	UserContextKey          = "user"
	RolesContextKey         = "roles"
	SessionIDHashContextKey = "session_id_hash"
	SessionRawTokenKey      = "session_raw_token"
)

// validateStaffSession memverifikasi token sesi cookie staff, mengecek kadaluarsa,
// mengambil data user & roles, serta memperpanjang sliding session expiry.
// Fungsi ini direuse oleh Authenticate dan DualAuth untuk menghindari duplikasi logic.
func validateStaffSession(c *gin.Context, q *dbgen.Queries, rawToken string) (dbgen.User, []string, string, bool) {
	if rawToken == "" {
		return dbgen.User{}, nil, "", false
	}

	idHash := auth.HashToken(rawToken)
	ctx := c.Request.Context()

	sess, err := q.GetSessionByIDHash(ctx, idHash)
	if err != nil {
		return dbgen.User{}, nil, "", false
	}

	now := time.Now()
	// Validate against sliding expiration and hard cap absolute expiration
	if now.After(sess.ExpiresAt.Time) || now.Equal(sess.ExpiresAt.Time) ||
		now.After(sess.AbsoluteExpiresAt.Time) || now.Equal(sess.AbsoluteExpiresAt.Time) {
		return dbgen.User{}, nil, "", false
	}

	user, err := q.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return dbgen.User{}, nil, "", false
	}

	roles, err := q.GetRolesByUserID(ctx, sess.UserID)
	if err != nil {
		return dbgen.User{}, nil, "", false
	}
	if roles == nil {
		roles = []string{}
	}

	// Extend sliding session duration by 2 hours for active valid requests
	newExpiresAt := now.Add(2 * time.Hour)
	_ = q.UpdateSessionExpiresAt(ctx, dbgen.UpdateSessionExpiresAtParams{
		IDHash: idHash,
		ExpiresAt: pgtype.Timestamptz{
			Time:  newExpiresAt,
			Valid: true,
		},
	})

	return user, roles, idHash, true
}

// Authenticate verifies the session cookie, checks expiry against DB, attaches user & roles to context,
// and extends the sliding session expiry by 2 hours on every valid request.
func Authenticate(q *dbgen.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		if q == nil {
			RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Database queries unit is not initialized", nil)
			c.Abort()
			return
		}

		rawToken, err := c.Cookie(SessionCookieName)
		if err != nil || rawToken == "" {
			RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak ditemukan atau telah berakhir", nil)
			c.Abort()
			return
		}

		user, roles, idHash, ok := validateStaffSession(c, q, rawToken)
		if !ok {
			RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Sesi tidak ditemukan atau telah berakhir", nil)
			c.Abort()
			return
		}

		// Attach user, roles, and session info to request context
		c.Set(UserContextKey, user)
		c.Set(RolesContextKey, roles)
		c.Set(SessionIDHashContextKey, idHash)
		c.Set(SessionRawTokenKey, rawToken)

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
