package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRole checks if the authenticated user possesses at least one of the specified allowed roles.
// Returns 403 Forbidden if the user lacks the required role.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := GetRolesFromContext(c)
		if !exists || len(userRoles) == 0 {
			RespondError(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak: peranan tidak sesuai", nil)
			c.Abort()
			return
		}

		for _, allowed := range allowedRoles {
			for _, userRole := range userRoles {
				if userRole == allowed {
					c.Next()
					return
				}
			}
		}

		RespondError(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak: peranan tidak sesuai", nil)
		c.Abort()
	}
}
