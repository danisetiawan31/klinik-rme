package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
)

func TestRequireRoleMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		contextRoles   []string
		allowedRoles   []string
		expectedStatus int
	}{
		{
			name:           "User has required role -> 200 OK",
			contextRoles:   []string{"petugas", "admin"},
			allowedRoles:   []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User has one of multiple allowed roles -> 200 OK",
			contextRoles:   []string{"dokter"},
			allowedRoles:   []string{"petugas", "dokter"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User does not have required role -> 403 Forbidden",
			contextRoles:   []string{"petugas"},
			allowedRoles:   []string{"admin", "dokter"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "User has empty roles -> 403 Forbidden",
			contextRoles:   []string{},
			allowedRoles:   []string{"admin"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(middleware.RequestID())

			// Mock auth context injector
			r.Use(func(c *gin.Context) {
				c.Set(middleware.RolesContextKey, tt.contextRoles)
				c.Next()
			})

			r.GET("/protected", middleware.RequireRole(tt.allowedRoles...), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusForbidden {
				var errResp middleware.ErrorEnvelope
				err := json.Unmarshal(w.Body.Bytes(), &errResp)
				require.NoError(t, err)

				assert.Equal(t, "FORBIDDEN", errResp.Error.Code)
				assert.Equal(t, "Akses ditolak: peranan tidak sesuai", errResp.Error.Message)
				assert.NotEmpty(t, errResp.Error.RequestID)
			}
		})
	}
}
