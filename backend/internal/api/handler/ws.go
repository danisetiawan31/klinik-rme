package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/danisetiawan31/klinik-rme/internal/api/middleware"
	dbgen "github.com/danisetiawan31/klinik-rme/internal/db/generated"
	"github.com/danisetiawan31/klinik-rme/internal/realtime"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// Default Gorilla Upgrader meng-enforce same-origin check (membandingkan header Origin vs Host).
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSHandler struct {
	hub *realtime.Hub
	q   *dbgen.Queries
}

func NewWSHandler(hub *realtime.Hub, q *dbgen.Queries) *WSHandler {
	return &WSHandler{
		hub: hub,
		q:   q,
	}
}

// ServeWS menangani koneksi WebSocket /ws?klinikId=X [Publik / Staff via Dual-Auth]
func (h *WSHandler) ServeWS(c *gin.Context) {
	if h.hub == nil {
		middleware.RespondError(c, http.StatusInternalServerError, "SERVER_ERROR", "Realtime hub unit is not initialized", nil)
		return
	}

	klinikIDParam := c.Query("klinikId")
	klinikIDParsed, err := strconv.Atoi(klinikIDParam)
	if err != nil || klinikIDParsed <= 0 {
		middleware.RespondError(c, http.StatusBadRequest, "INVALID_PARAMETER", "ID klinik tidak valid", err)
		return
	}
	klinikID := int32(klinikIDParsed)

	// Upgrade koneksi HTTP ke WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS_UPGRADE_ERROR] Failed to upgrade connection: %v", err)
		return
	}

	client := realtime.NewClient(klinikID)
	h.hub.RegisterClient(client)

	go h.writePump(client, conn)
	h.readPump(client, conn)
}

func (h *WSHandler) readPump(client *realtime.Client, conn *websocket.Conn) {
	defer func() {
		h.hub.UnregisterClient(client)
		_ = conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
		// Servis ini satu arah (server -> client notification). Discard isi pesan client.
	}
}

func (h *WSHandler) writePump(client *realtime.Client, conn *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed oleh Hub -> kirim CloseMessage
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
