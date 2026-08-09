package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/api"
	"github.com/danisetiawan31/klinik-rme/internal/api/handler"
	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthEndpoint_NilPool(t *testing.T) {
	router := api.SetupRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var errResp middleware.ErrorEnvelope
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "DB_UNAVAILABLE", errResp.Error.Code)
	assert.Equal(t, "Layanan database tidak tersedia", errResp.Error.Message)
	assert.NotEmpty(t, errResp.Error.RequestID)
}

func TestHealthEndpoint_RealPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Spin up real PostgreSQL container using testcontainers-go
	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err, "failed to start postgres container")

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	router := api.SetupRouter(pool)

	// Skenario 1: DB Reachable -> 200 OK
	t.Run("DB Reachable -> 200 OK", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp handler.HealthResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, "ok", resp.DB)
	})

	// Skenario 2: DB Unreachable (pool closed & container terminated) -> 503 Service Unavailable
	t.Run("DB Unreachable -> 503 Service Unavailable", func(t *testing.T) {
		pool.Close()
		err := postgresContainer.Terminate(ctx)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var errResp middleware.ErrorEnvelope
		err = json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)

		assert.Equal(t, "DB_UNAVAILABLE", errResp.Error.Code)
		assert.Equal(t, "Layanan database tidak dapat dijangkau", errResp.Error.Message)
		assert.NotEmpty(t, errResp.Error.RequestID)
	})
}
