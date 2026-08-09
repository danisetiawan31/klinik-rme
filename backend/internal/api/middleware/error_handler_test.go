package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestIDMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())

	var capturedReqID string
	r.GET("/test", func(c *gin.Context) {
		capturedReqID = middleware.GetRequestID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, capturedReqID)
	assert.Equal(t, capturedReqID, w.Header().Get(middleware.RequestIDHeader))
}

func TestRespondError_Format(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())

	r.GET("/test-err", func(c *gin.Context) {
		rawErr := errors.Is(errors.New("db connection timeout: secret_credential_info"), errors.New("something"))
		_ = rawErr
		middleware.RespondError(
			c,
			http.StatusBadRequest,
			"BAD_REQUEST",
			"Input tidak valid",
			errors.New("raw db error details should not leak"),
		)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test-err", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp middleware.ErrorEnvelope
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "BAD_REQUEST", resp.Error.Code)
	assert.Equal(t, "Input tidak valid", resp.Error.Message)
	assert.NotEmpty(t, resp.Error.RequestID)

	// Ensure no raw error details leaked into response body
	assert.NotContains(t, w.Body.String(), "raw db error details")
	assert.NotContains(t, w.Body.String(), "secret_credential_info")
}

func TestGlobalRecovery_PanicHandler(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.GlobalRecovery())

	r.GET("/test-panic", func(c *gin.Context) {
		panic("database connection string postgres://user:password@host/db unexpected crash")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test-panic", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp middleware.ErrorEnvelope
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "INTERNAL_SERVER_ERROR", resp.Error.Code)
	assert.Equal(t, "Terjadi kesalahan internal pada server", resp.Error.Message)
	assert.NotEmpty(t, resp.Error.RequestID)

	// Ensure panic message and sensitive DSN details are NOT leaked to client response
	assert.NotContains(t, w.Body.String(), "postgres://user:password")
}
