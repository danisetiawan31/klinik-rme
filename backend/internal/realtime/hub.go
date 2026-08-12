package realtime

import (
	"context"
)

// DefaultSendBufferSize adalah ukuran default buffer channel Send milik Client.
const DefaultSendBufferSize = 16

// QueueUpdatedMessage adalah payload notifikasi invalidation ping.
var QueueUpdatedMessage = []byte(`{"type":"queue_updated"}`)

// Client mewakili satu koneksi realtime yang terikat ke klinikID tertentu.
type Client struct {
	KlinikID int32
	Send     chan []byte
}

// NewClient membuat instance Client baru dengan buffered channel Send.
func NewClient(klinikID int32) *Client {
	return &Client{
		KlinikID: klinikID,
		Send:     make(chan []byte, DefaultSendBufferSize),
	}
}

// Hub mengelola registrasi, unregistrasi, dan broadcast notifikasi ke Client secara in-memory.
type Hub struct {
	clients    map[int32]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan int32
	closed     chan struct{}
}

// NewHub membuat instance Hub baru.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int32]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan int32),
		closed:     make(chan struct{}),
	}
}

// Run menjalankan event loop utama Hub dalam 1 goroutine tunggal.
// Seluruh mutasi pada map clients strictly terjadi di goroutine ini (zero data race).
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown: close semua channel client dan bersihkan map
			for klinikID, clientSet := range h.clients {
				for client := range clientSet {
					close(client.Send)
				}
				delete(h.clients, klinikID)
			}
			close(h.closed)
			return

		case client := <-h.register:
			if client == nil {
				continue
			}
			clientSet, exists := h.clients[client.KlinikID]
			if !exists {
				clientSet = make(map[*Client]bool)
				h.clients[client.KlinikID] = clientSet
			}
			clientSet[client] = true

		case client := <-h.unregister:
			if client == nil {
				continue
			}
			h.removeClient(client)

		case klinikID := <-h.broadcast:
			clientSet, exists := h.clients[klinikID]
			if !exists || len(clientSet) == 0 {
				continue
			}

			// Non-blocking send ke seluruh client terdaftar untuk klinikID tsb
			for client := range clientSet {
				select {
				case client.Send <- QueueUpdatedMessage:
				default:
					// Channel penuh (client lambat/mati): unregister & close channel
					close(client.Send)
					delete(clientSet, client)
				}
			}

			if len(clientSet) == 0 {
				delete(h.clients, klinikID)
			}
		}
	}
}

// removeClient menghapus client dari map dan meng-close channel Send-nya jika client masih ada.
// Method ini internal dan hanya dipanggil di dalam goroutine Run.
func (h *Hub) removeClient(client *Client) {
	clientSet, exists := h.clients[client.KlinikID]
	if !exists {
		return
	}

	if _, found := clientSet[client]; found {
		delete(clientSet, client)
		close(client.Send)

		if len(clientSet) == 0 {
			delete(h.clients, client.KlinikID)
		}
	}
}

// RegisterClient mengirim permintaan registrasi client ke Hub secara thread-safe.
// Jika Hub telah di-shutdown (closed), panggilan ini akan no-op dan tidak blocking.
func (h *Hub) RegisterClient(c *Client) {
	select {
	case h.register <- c:
	case <-h.closed:
	}
}

// UnregisterClient mengirim permintaan unregistrasi client ke Hub secara thread-safe.
// Jika Hub telah di-shutdown (closed), panggilan ini akan no-op dan tidak blocking.
func (h *Hub) UnregisterClient(c *Client) {
	select {
	case h.unregister <- c:
	case <-h.closed:
	}
}

// BroadcastToKlinik mengirim sinyal broadcast untuk klinikID tertentu ke Hub secara thread-safe.
// Jika Hub telah di-shutdown (closed), panggilan ini akan no-op dan tidak blocking.
func (h *Hub) BroadcastToKlinik(klinikID int32) {
	select {
	case h.broadcast <- klinikID:
	case <-h.closed:
	}
}

// ClientCount returns the number of registered clients for a given klinikID.
// NOTE: Utama digunakan untuk pengujian (hanya aman jika dipanggil setelah event diproses).
func (h *Hub) ClientCount(klinikID int32) int {
	if clientSet, ok := h.clients[klinikID]; ok {
		return len(clientSet)
	}
	return 0
}
