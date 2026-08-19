package realtime_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/danisetiawan31/klinik-rme/internal/realtime"
)

func TestHub_Concurrency(t *testing.T) {
	hub := realtime.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Jalankan Hub.Run di goroutine terpisah
	go hub.Run(ctx)

	const numGoroutines = 25
	const operationsPerGoroutine = 50
	const numKlinik = 3

	var registeredCount int64
	var unregisteredCount int64

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		goroutineID := g
		go func() {
			defer wg.Done()

			// Setiap goroutine mengelola set client tersendiri
			localClients := make([]*realtime.Client, 0)

			for op := 0; op < operationsPerGoroutine; op++ {
				klinikID := int32((goroutineID+op)%numKlinik + 1)

				switch op % 3 {
				case 0:
					// Register
					client := realtime.NewClient(klinikID)
					localClients = append(localClients, client)
					hub.RegisterClient(client)
					atomic.AddInt64(&registeredCount, 1)

				case 1:
					// Unregister (jika ada client di localClients)
					if len(localClients) > 0 {
						targetIndex := op % len(localClients)
						targetClient := localClients[targetIndex]

						hub.UnregisterClient(targetClient)
						atomic.AddInt64(&unregisteredCount, 1)

						// Hapus dari localClients
						localClients = append(localClients[:targetIndex], localClients[targetIndex+1:]...)
					}

				case 2:
					// Broadcast
					hub.BroadcastToKlinik(klinikID)
				}

				// Small sleep to mix concurrency interleaving
				time.Sleep(10 * time.Microsecond)
			}

			// Clean up sisanya
			for _, client := range localClients {
				hub.UnregisterClient(client)
				atomic.AddInt64(&unregisteredCount, 1)
			}
		}()
	}

	wg.Wait()

	// Berikan sedikit waktu agar saluran channel di Hub selesai memproses seluruh pesan pending
	time.Sleep(100 * time.Millisecond)

	expectedRemaining := registeredCount - unregisteredCount
	assert.GreaterOrEqual(t, registeredCount, int64(numGoroutines), "Registered count minimal harus sesuai")
	assert.Equal(t, registeredCount, unregisteredCount, "Seluruh client yang terdaftar telah di-unregister")

	// Pastikan total client terdaftar di semua klinik adalah 0
	var actualRemaining int
	for k := int32(1); k <= numKlinik; k++ {
		actualRemaining += hub.ClientCount(k)
	}

	assert.Equal(t, int(expectedRemaining), actualRemaining, "State akhir map client di Hub konsisten dengan perhitungan manual")
}

func TestHub_NonBlockingBroadcast(t *testing.T) {
	hub := realtime.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	const klinikID = int32(10)

	// Buat client dengan buffer Send berukuran 1
	client := &realtime.Client{
		KlinikID: klinikID,
		Send:     make(chan []byte, 1),
	}

	hub.RegisterClient(client)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount(klinikID))

	// Isi channel client sampai penuh tanpa konsumsi
	client.Send <- []byte("message_1")

	// Channel client sekarang PENUH (size=1, len=1).
	// Broadcast berikutnya HARUS non-blocking dan meng-unregister client yang lambat/mati tersebut.
	done := make(chan bool)
	go func() {
		hub.BroadcastToKlinik(klinikID)
		done <- true
	}()

	select {
	case <-done:
		// PASS: Broadcast tidak blocking / hang
	case <-time.After(1 * time.Second):
		t.Fatal("FAIL: Hub.BroadcastToKlinik blocking / hang saat channel client penuh!")
	}

	time.Sleep(50 * time.Millisecond)

	// Assert: Client ter-unregister otomatis karena channel penuh saat broadcast
	assert.Equal(t, 0, hub.ClientCount(klinikID), "Client yang lambat/penuh harus di-unregister otomatis dari Hub")

	// Channel Send milik client harus sudah di-close oleh Hub
	_, open := <-client.Send
	// message_1 ter-read, lalu receive berikutnya false (closed)
	_, open = <-client.Send
	assert.False(t, open, "Channel client.Send harus di-close setelah unregister otomatis")
}

func TestHub_GracefulShutdown(t *testing.T) {
	hub := realtime.NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	runExited := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(runExited)
	}()

	client1 := realtime.NewClient(1)
	client2 := realtime.NewClient(2)

	hub.RegisterClient(client1)
	hub.RegisterClient(client2)
	time.Sleep(20 * time.Millisecond)

	// Trigger shutdown via context cancel
	cancel()

	select {
	case <-runExited:
		// PASS: Goroutine Run exit bersih
	case <-time.After(1 * time.Second):
		t.Fatal("FAIL: Hub.Run tidak exit setelah context cancel (goroutine leak)!")
	}

	// Assert channel client1 dan client2 sudah di-close
	_, open1 := <-client1.Send
	assert.False(t, open1, "client1.Send channel harus di-close saat Hub shutdown")

	_, open2 := <-client2.Send
	assert.False(t, open2, "client2.Send channel harus di-close saat Hub shutdown")
}

func TestHub_DoubleUnregisterSafety(t *testing.T) {
	hub := realtime.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	client := realtime.NewClient(1)
	hub.RegisterClient(client)
	time.Sleep(20 * time.Millisecond)

	// Unregister pertama kali
	hub.UnregisterClient(client)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 0, hub.ClientCount(1))

	// Unregister kedua kali pada client yang sama -> HARUS aman, TIDAK Boleh panic double-close
	assert.NotPanics(t, func() {
		hub.UnregisterClient(client)
		time.Sleep(20 * time.Millisecond)
	}, "Double unregister tidak boleh memicu panic")
}

func TestHub_MultipleClientsSameKlinik(t *testing.T) {
	hub := realtime.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	const klinikID = int32(5)
	c1 := realtime.NewClient(klinikID)
	c2 := realtime.NewClient(klinikID)

	hub.RegisterClient(c1)
	hub.RegisterClient(c2)
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 2, hub.ClientCount(klinikID))

	// Broadcast ke klinik 5
	hub.BroadcastToKlinik(klinikID)

	msg1 := <-c1.Send
	msg2 := <-c2.Send

	assert.Equal(t, string(realtime.QueueUpdatedMessage), string(msg1))
	assert.Equal(t, string(realtime.QueueUpdatedMessage), string(msg2))

	// Unregister c1, c2 tetap ada
	hub.UnregisterClient(c1)
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 1, hub.ClientCount(klinikID))
}

func TestHub_PostShutdownCalls(t *testing.T) {
	hub := realtime.NewHub()
	ctx, cancel := context.WithCancel(context.Background())

	runExited := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(runExited)
	}()

	// Graceful shutdown Hub
	cancel()
	<-runExited

	// Memanggil method-method publik SETELAH Hub.Run() berhenti total
	// Harus return CEPAT tanpa deadlock/hang dan tanpa panic
	client := realtime.NewClient(1)

	done := make(chan bool)
	go func() {
		hub.RegisterClient(client)
		hub.UnregisterClient(client)
		hub.BroadcastToKlinik(1)
		done <- true
	}()

	select {
	case <-done:
		// PASS: Memanggil method publik setelah shutdown return cepat tanpa deadlock
	case <-time.After(1 * time.Second):
		t.Fatal("FAIL: Memanggil method Hub publik pasca-shutdown mengalami deadlock!")
	}
}
