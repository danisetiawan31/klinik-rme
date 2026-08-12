package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/api"
	"github.com/danisetiawan31/klinik-rme/internal/auth"
	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/realtime"
)

func TestWSHandler_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("ws_test_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := db.NewPool(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
	})

	err = db.RunMigrations("../../../migrations", connStr)
	require.NoError(t, err)

	q := dbgen.New(pool)
	cfg := &config.Config{
		KlinikNama:     "Klinik Realtime Sehat",
		KlinikJamBuka:  "08:00",
		KlinikJamTutup: "23:59",
	}
	err = bootstrap.SeedKlinik(ctx, pool, q, cfg)
	require.NoError(t, err)

	singleKlinik, err := q.GetSingleKlinik(ctx)
	require.NoError(t, err)
	klinikID := singleKlinik.ID

	// Generate & Simpan display token valid di DB
	rawDisplayToken, err := auth.GenerateToken()
	require.NoError(t, err)
	hashedDisplayToken := auth.HashToken(rawDisplayToken)
	_, err = q.UpdateKlinikDisplayTokenHash(ctx, dbgen.UpdateKlinikDisplayTokenHashParams{
		DisplayTokenHash: pgtype.Text{String: hashedDisplayToken, Valid: true},
		ID:               klinikID,
	})
	require.NoError(t, err)

	// Instansiasi Hub & Run loop
	hub := realtime.NewHub()
	hubCtx, hubCancel := context.WithCancel(context.Background())
	t.Cleanup(hubCancel)
	go hub.Run(hubCtx)

	router := api.SetupRouter(pool, hub, nil, "http://localhost:4200")
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	// User Cookie
	staffCookie, _ := createDisplayTokenTestUser(t, ctx, pool, q, "petugas_ws@klinik.id", []string{"petugas"})

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	t.Run("1. Auth via Cookie Staff Valid -> Upgrade Sukses & Ter-register di Hub", func(t *testing.T) {
		dialer := websocket.Dialer{}
		header := http.Header{}
		header.Add("Cookie", fmt.Sprintf("session=%s", staffCookie.Value))

		conn, resp, err := dialer.Dial(fmt.Sprintf("%s?klinikId=%d", wsURL, klinikID), header)
		require.NoError(t, err)
		defer conn.Close()

		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, hub.ClientCount(klinikID), "Client staff harus ter-register di Hub")
	})

	t.Run("2. Auth via Display-Token Valid (?displayToken=) -> Upgrade Sukses & Ter-register", func(t *testing.T) {
		dialer := websocket.Dialer{}
		urlStr := fmt.Sprintf("%s?klinikId=%d&displayToken=%s", wsURL, klinikID, rawDisplayToken)

		conn, resp, err := dialer.Dial(urlStr, nil)
		require.NoError(t, err)
		defer conn.Close()

		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		time.Sleep(50 * time.Millisecond)

		assert.GreaterOrEqual(t, hub.ClientCount(klinikID), 1, "Client display-token harus ter-register di Hub")
	})

	t.Run("3. Auth Gagal (Tanpa Auth / Token Invalid) -> 401 & Unregistered", func(t *testing.T) {
		dialer := websocket.Dialer{}

		// Client tanpa auth sama sekali
		_, resp, err := dialer.Dial(fmt.Sprintf("%s?klinikId=%d", wsURL, klinikID), nil)
		require.Error(t, err)
		if resp != nil {
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		}

		// Client dengan token invalid
		urlInvalid := fmt.Sprintf("%s?klinikId=%d&displayToken=invalid_token", wsURL, klinikID)
		_, resp2, err2 := dialer.Dial(urlInvalid, nil)
		require.Error(t, err2)
		if resp2 != nil {
			assert.Equal(t, http.StatusUnauthorized, resp2.StatusCode)
		}
	})

	t.Run("4. Disconnect Handling -> Ter-unregister Otomatis dari Hub", func(t *testing.T) {
		dialer := websocket.Dialer{}
		urlStr := fmt.Sprintf("%s?klinikId=%d&displayToken=%s", wsURL, klinikID, rawDisplayToken)

		conn, _, err := dialer.Dial(urlStr, nil)
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)
		initialCount := hub.ClientCount(klinikID)

		// Close koneksi dari sisi client
		_ = conn.Close()
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, initialCount-1, hub.ClientCount(klinikID), "Client harus ter-unregister dari Hub setelah disconnect")
	})

	t.Run("5. Multiple Clients -> Ter-register Independen", func(t *testing.T) {
		dialer := websocket.Dialer{}
		urlStr := fmt.Sprintf("%s?klinikId=%d&displayToken=%s", wsURL, klinikID, rawDisplayToken)

		conn1, _, err1 := dialer.Dial(urlStr, nil)
		require.NoError(t, err1)
		defer conn1.Close()

		conn2, _, err2 := dialer.Dial(urlStr, nil)
		require.NoError(t, err2)
		defer conn2.Close()

		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 2, hub.ClientCount(klinikID), "Dua client terhubung harus ter-register independen (count=2)")
	})

	t.Run("6. Broadcast Ping Notification -> Client Menerima Payload QueueUpdatedMessage", func(t *testing.T) {
		dialer := websocket.Dialer{}
		urlStr := fmt.Sprintf("%s?klinikId=%d&displayToken=%s", wsURL, klinikID, rawDisplayToken)

		conn, _, err := dialer.Dial(urlStr, nil)
		require.NoError(t, err)
		defer conn.Close()

		time.Sleep(50 * time.Millisecond)

		// Panggil broadcast manual dari test ke Hub
		hub.BroadcastToKlinik(klinikID)

		// Read message di sisi WS client test
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		msgType, msgBytes, err := conn.ReadMessage()
		require.NoError(t, err)

		assert.Equal(t, websocket.TextMessage, msgType)

		var payload map[string]string
		err = json.Unmarshal(msgBytes, &payload)
		require.NoError(t, err)

		assert.Equal(t, "queue_updated", payload["type"], "Client WS harus menerima payload {\"type\":\"queue_updated\"}")
	})
}
