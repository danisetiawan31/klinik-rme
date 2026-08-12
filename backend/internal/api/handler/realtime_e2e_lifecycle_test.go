package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/danisetiawan31/klinik-rme/internal/api"
	"github.com/danisetiawan31/klinik-rme/internal/api/handler"
	"github.com/danisetiawan31/klinik-rme/internal/bootstrap"
	"github.com/danisetiawan31/klinik-rme/internal/config"
	"github.com/danisetiawan31/klinik-rme/internal/db"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/realtime"
)

func TestRealtime_E2ELifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("realtime_e2e_db"),
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
		KlinikNama:     "Klinik E2E Lifecycle Realtime",
		KlinikJamBuka:  "00:00",
		KlinikJamTutup: "23:59",
	}
	err = bootstrap.SeedKlinik(ctx, pool, q, cfg)
	require.NoError(t, err)

	klinik, err := q.GetSingleKlinik(ctx)
	require.NoError(t, err)
	klinikID := klinik.ID

	// Instansiasi Realtime Hub & Start Run loop
	hub := realtime.NewHub()
	hubCtx, hubCancel := context.WithCancel(context.Background())
	t.Cleanup(hubCancel)
	go hub.Run(hubCtx)

	router := api.SetupRouter(pool, hub, nil, "http://localhost:3000")
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	// User Sessions (Admin, Petugas, Dokter)
	adminCookie, _ := createDisplayTokenTestUser(t, ctx, pool, q, "admin_e2e@klinik.id", []string{"admin"})
	petugasCookie, _ := createDisplayTokenTestUser(t, ctx, pool, q, "petugas_e2e@klinik.id", []string{"petugas"})
	dokterCookie, _ := createDisplayTokenTestUser(t, ctx, pool, q, "dokter_e2e@klinik.id", []string{"dokter"})

	// Helper assert WS message received
	assertWSNotification := func(t *testing.T, conn *websocket.Conn, name string) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		msgType, msgBytes, err := conn.ReadMessage()
		require.NoError(t, err, "WS client [%s] gagal membaca notifikasi broadcast", name)
		assert.Equal(t, websocket.TextMessage, msgType)

		var payload map[string]string
		err = json.Unmarshal(msgBytes, &payload)
		require.NoError(t, err)
		assert.Equal(t, "queue_updated", payload["type"], "WS client [%s] harus menerima type=queue_updated", name)
	}

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// =========================================================================
	// Langkah a: Admin Regenerate Display Token 1
	// =========================================================================
	regenURL := fmt.Sprintf("%s/api/v1/admin/klinik/%d/display-token/regenerate", server.URL, klinikID)
	reqRegen1, _ := http.NewRequest(http.MethodPost, regenURL, nil)
	reqRegen1.AddCookie(adminCookie)

	clientHTTP := &http.Client{}
	resp1, err := clientHTTP.Do(reqRegen1)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp1.StatusCode)

	var regenResp1 handler.RegenerateDisplayTokenResponse
	err = json.NewDecoder(resp1.Body).Decode(&regenResp1)
	resp1.Body.Close()
	require.NoError(t, err)
	displayToken1 := regenResp1.DisplayToken
	require.NotEmpty(t, displayToken1)

	// =========================================================================
	// Langkah b: Buka Koneksi WS 1 (gorilla/websocket) via Display Token 1
	// =========================================================================
	dialer := websocket.Dialer{}
	wsURL1 := fmt.Sprintf("%s?klinikId=%d&displayToken=%s", wsURL, klinikID, displayToken1)
	wsConn1, respWS1, err := dialer.Dial(wsURL1, nil)
	require.NoError(t, err)
	defer wsConn1.Close()
	assert.Equal(t, http.StatusSwitchingProtocols, respWS1.StatusCode)
	time.Sleep(50 * time.Millisecond)

	// =========================================================================
	// Langkah c: Buka Koneksi WS 2 via Cookie Staff Dokter
	// =========================================================================
	wsHeader2 := http.Header{}
	wsHeader2.Add("Cookie", fmt.Sprintf("session=%s", dokterCookie.Value))
	wsURL2 := fmt.Sprintf("%s?klinikId=%d", wsURL, klinikID)
	wsConn2, respWS2, err := dialer.Dial(wsURL2, wsHeader2)
	require.NoError(t, err)
	defer wsConn2.Close()
	assert.Equal(t, http.StatusSwitchingProtocols, respWS2.StatusCode)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 2, hub.ClientCount(klinikID), "Harus ada 2 client WS terhubung di Hub")

	// =========================================================================
	// Langkah d: Petugas REST Call -> POST /pasien lalu POST /kunjungan
	// =========================================================================
	nikStr := "3271234567890009"
	consentVal := true
	pasienReqBody, _ := json.Marshal(handler.CreatePasienRequest{
		Nama:         "Pasien Lifecycle E2E",
		Nik:          &nikStr,
		TanggalLahir: "1990-01-01",
		JenisKelamin: "L",
		Alamat:       "Jl Lifecycle E2E",
		NoTelp:       "081299998888",
		Consent:      &consentVal,
	})
	reqPasien, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/pasien", bytes.NewBuffer(pasienReqBody))
	reqPasien.Header.Set("Content-Type", "application/json")
	reqPasien.AddCookie(petugasCookie)

	respPasien, err := clientHTTP.Do(reqPasien)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, respPasien.StatusCode)

	var pasienResp handler.PasienResponse
	_ = json.NewDecoder(respPasien.Body).Decode(&pasienResp)
	respPasien.Body.Close()

	kunjunganReqBody, _ := json.Marshal(handler.CreateKunjunganRequest{
		PasienID: pasienResp.ID,
	})
	reqKunjungan, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/kunjungan", bytes.NewBuffer(kunjunganReqBody))
	reqKunjungan.Header.Set("Content-Type", "application/json")
	reqKunjungan.AddCookie(petugasCookie)

	respKunjungan, err := clientHTTP.Do(reqKunjungan)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, respKunjungan.StatusCode)

	var kunjunganResp handler.CreateKunjunganResponse
	_ = json.NewDecoder(respKunjungan.Body).Decode(&kunjunganResp)
	respKunjungan.Body.Close()

	// =========================================================================
	// Langkah e: Assert KEDUA Client WS Menerima Notifikasi Broadcast
	// =========================================================================
	assertWSNotification(t, wsConn1, "Client 1 (Display Token)")
	assertWSNotification(t, wsConn2, "Client 2 (Staff Cookie)")

	// =========================================================================
	// Langkah f: Dokter REST Call -> POST /panggil-berikutnya
	// =========================================================================
	panggilURL := fmt.Sprintf("%s/api/v1/klinik/%d/panggil-berikutnya", server.URL, klinikID)
	reqPanggil, _ := http.NewRequest(http.MethodPost, panggilURL, nil)
	reqPanggil.AddCookie(dokterCookie)

	respPanggil, err := clientHTTP.Do(reqPanggil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, respPanggil.StatusCode)
	respPanggil.Body.Close()

	// Assert KEDUA Client WS Menerima Notifikasi lagi
	assertWSNotification(t, wsConn1, "Client 1 (Display Token)")
	assertWSNotification(t, wsConn2, "Client 2 (Staff Cookie)")

	// =========================================================================
	// Langkah g: Admin Regenerate Display Token 2 (Token Baru)
	// =========================================================================
	reqRegen2, _ := http.NewRequest(http.MethodPost, regenURL, nil)
	reqRegen2.AddCookie(adminCookie)

	resp2, err := clientHTTP.Do(reqRegen2)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var regenResp2 handler.RegenerateDisplayTokenResponse
	_ = json.NewDecoder(resp2.Body).Decode(&regenResp2)
	resp2.Body.Close()
	displayToken2 := regenResp2.DisplayToken
	require.NotEmpty(t, displayToken2)
	require.NotEqual(t, displayToken1, displayToken2)

	// Assert WS Client 1 (yang connect pakai Token 1 lama) TETAP terhubung (ping frame / no disconnect error)
	errPing := wsConn1.WriteMessage(websocket.PingMessage, nil)
	require.NoError(t, errPing, "WS Client 1 yang terhubung via token lama harus TETAP terhubung di socket level")

	// =========================================================================
	// Langkah h: Disconnect WS Client 1 & Perform Write -> Only Client 2 Receives Broadcast
	// =========================================================================
	_ = wsConn1.Close()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount(klinikID), "Client count di Hub harus berkurang jadi 1 setelah disconnect")

	// Lakukan 1 write lagi: Dokter lewati kunjungan
	lewatiURL := fmt.Sprintf("%s/api/v1/kunjungan/%d/lewati", server.URL, kunjunganResp.ID)
	reqLewati, _ := http.NewRequest(http.MethodPost, lewatiURL, nil)
	reqLewati.AddCookie(dokterCookie)

	respLewati, err := clientHTTP.Do(reqLewati)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, respLewati.StatusCode)
	respLewati.Body.Close()

	// Assert HANYA Client 2 yang menerima notifikasi (Client 1 sudah closed)
	assertWSNotification(t, wsConn2, "Client 2 (Staff Cookie)")

	// =========================================================================
	// Langkah i: REST GET /antrian dengan Token Baru vs Token Lama
	// =========================================================================
	antrianURL := fmt.Sprintf("%s/api/v1/klinik/%d/antrian", server.URL, klinikID)

	// i.1. Request dengan Token 2 (BARU) -> Sukses 200 OK & shape publik (tanpa id & tanpa pasienNama)
	reqAntrianBaru, _ := http.NewRequest(http.MethodGet, antrianURL, nil)
	reqAntrianBaru.Header.Set("X-Display-Token", displayToken2)

	respAntrianBaru, err := clientHTTP.Do(reqAntrianBaru)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, respAntrianBaru.StatusCode)

	bodyBytesBaru, _ := io.ReadAll(respAntrianBaru.Body)
	respAntrianBaru.Body.Close()

	var antrianPublikItems []handler.AntrianItemPublicResponse
	err = json.Unmarshal(bodyBytesBaru, &antrianPublikItems)
	require.NoError(t, err)

	// Pastikan shape JSON publik tidak mengandung field id & pasienNama
	var rawJSONList []map[string]interface{}
	_ = json.Unmarshal(bodyBytesBaru, &rawJSONList)
	if len(rawJSONList) > 0 {
		_, hasID := rawJSONList[0]["id"]
		_, hasPasienNama := rawJSONList[0]["pasienNama"]
		assert.False(t, hasID, "Payload publik TIDAK Boleh mengandung field 'id'")
		assert.False(t, hasPasienNama, "Payload publik TIDAK Boleh mengandung field 'pasienNama'")
	}

	// i.2. Request dengan Token 1 (LAMA / REVOKED) -> Gagal 401 Unauthorized
	reqAntrianLama, _ := http.NewRequest(http.MethodGet, antrianURL, nil)
	reqAntrianLama.Header.Set("X-Display-Token", displayToken1)

	respAntrianLama, err := clientHTTP.Do(reqAntrianLama)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, respAntrianLama.StatusCode)
	respAntrianLama.Body.Close()
}
