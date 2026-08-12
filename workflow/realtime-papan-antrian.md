# Realtime & Papan Antrian

## Konteks & tujuan

WebSocket hub in-memory untuk notifikasi realtime perubahan antrian, dan autentikasi terpisah (display token) untuk papan antrian publik yang tidak pakai sesi staff. Fitur ini SEBAGIAN BESAR retrofit — menyuntik broadcast call ke handler yang sudah ada dari item #5 dan #7 (sudah closed), bukan cuma menulis kode baru berdiri sendiri.

## Requirement

### Migration

- ALTER TABLE klinik ADD COLUMN display_token_hash (nullable, mulai NULL sampai admin panggil regenerate minimal 1x — ini disengaja, bukan bug: sebelum itu papan antrian tidak mungkin autentikasi karena tidak ada apapun yang bisa match ke NULL).

### Endpoint baru

- `POST /admin/klinik/:id/display-token/regenerate` [admin]
- `WS /ws?klinikId=X[&displayToken=...]` [petugas, dokter, admin via cookie | display-token] — TANPA prefix /api/v1 (ikuti literal docs/api-contract.md persis, ini konsisten dengan pola /health yang juga di luar /api/v1).

### Endpoint yang DIMODIFIKASI (bukan baru)

- `GET /klinik/:id/antrian` (item #5) — middleware ganti jadi dual-auth, response shape beda per channel.

### Hub WebSocket (gorilla/websocket, in-memory, single proses)

- 1 goroutine tunggal kelola `map[klinikId][]*Client` via channel (register/unregister/broadcast) — TIDAK ADA mutex di map, hindari data race sesuai docs/TDD.md.
- Tiap `Client`: `send chan []byte` (buffered) + goroutine `writePump` terpisah yang benar-benar nulis ke `conn.WriteMessage` — TIDAK PERNAH ada 2 goroutine nulis ke 1 `*websocket.Conn` yang sama secara bersamaan (larangan eksplisit gorilla/websocket).
- `readPump` per client: cuma untuk deteksi close/pong (ping-pong keepalive standar), server TIDAK PERNAH proses pesan isi dari client (WS ini satu arah: server→client notifikasi saja, sesuai docs/TDD.md "invalidation ping saja").
- Broadcast: kirim `{"type":"queue_updated"}` ke SEMUA client terdaftar untuk `klinikId` tertentu. Non-blocking send ke `send channel` tiap client — kalau channel penuh (client lambat/mati), unregister client itu, JANGAN blocking Hub goroutine menunggu 1 client lambat.

### Dual-auth (REUSE, bukan endpoint baru — logic auth ini dipakai GET /klinik/:id/antrian DAN WS /ws)

Beda dari RBAC di endpoint lain: token cocok = otomatis authorized, TIDAK ADA role check untuk jalur display-token (konsep "role" tidak berlaku, token bukan milik user manapun).

- Cek cookie staff session dulu: kalau valid → authorized sebagai staff, WAJIB role IN (petugas, dokter, admin) — pakai middleware Authenticate+RequireRole yang sudah ada.
- Kalau cookie tidak ada/invalid → cek `X-Display-Token` (REST) atau query param `?displayToken=` (WS, WAJIB beda dari REST — browser WebSocket API tidak bisa set custom header, ini keterbatasan browser, TULIS KOMENTAR EKSPLISIT di kode biar tidak ada yang "perbaiki" jadi header dan diam-diam gagal, sesuai docs/AGENTS.md §7): hash token yang dikirim, bandingkan ke `klinik.display_token_hash` untuk `:id`/`klinikId` di request. `display_token_hash` NULL (belum pernah di-regenerate) → SELALU gagal match, apapun yang dikirim client. Match → authorized, TANPA role check.
- Dua-duanya gagal → 401.
- Attach ke context: channel yang dipakai (`"cookie"` atau `"display-token"`) — handler butuh info ini untuk tentukan response shape (GET /klinik/:id/antrian).

### `POST /admin/klinik/:id/display-token/regenerate` — detail

- Generate token baru (auth.GenerateToken dari item #2, REUSE), hash (auth.HashToken, REUSE).
- `UPDATE klinik SET display_token_hash=? WHERE id=?` — overwrite langsung, TIDAK ADA special-case "pertama kali vs regenerate ulang", keduanya sama-sama overwrite.
- Token lama (kalau ada) otomatis revoked (hash lama sudah tidak match apapun). Koneksi WS yang SUDAH TERSAMBUNG pakai token lama TETAP terhubung (auth cuma dicek saat handshake, bukan per-pesan) — keputusan diambil langsung karena tidak eksplisit di dokumen manapun, kalau salah baca intent tolong koreksi.
- Response 200 { displayToken } — token MENTAH, cuma sekali ini terlihat sebagai plaintext (pola sama seperti seed admin/invite user).

### `GET /klinik/:id/antrian` — retrofit detail

- Channel `"cookie"` → response `[{ id, nomorAntrian, status, isPriority, pasienNama }]` (shape yang sudah ada dari item #5, TIDAK BERUBAH).
- Channel `"display-token"` → response BARU `[{ nomorAntrian, status, isPriority }]` (TANPA `id`, TANPA `pasienNama` — publik, ruang tunggu, tidak boleh bocorkan identitas pasien).

### `WS /ws?klinikId=X[&displayToken=...]` — detail

- Upgrade koneksi HTTP ke WebSocket via gorilla/websocket Upgrader.
- Auth pakai logic dual-auth yang SAMA (REUSE) dengan GET /klinik/:id/antrian — gagal auth → tolak upgrade (401 sebelum upgrade, JANGAN upgrade dulu baru close).
- Sukses → register `Client` baru ke Hub untuk `klinikId` itu.
- Client disconnect (browser close, network drop, dst) → unregister dari Hub (readPump return → trigger unregister).

### Broadcast retrofit — 5 titik panggil di kode yang SUDAH ADA (item #5 dan #7)

Setelah `tx.Commit()` sukses, panggil `hub.Broadcast(klinikID)`:

1. `POST /kunjungan` (item #5)
2. `POST /klinik/:id/panggil-berikutnya` (item #5)
3. `POST /kunjungan/:id/lewati` (item #5)
4. `POST /kunjungan/:id/tidak-hadir` (item #5)
5. `POST /kunjungan/:id/rekam-medis` (item #7) — SATU-SATUNYA endpoint rekam medis yang broadcast, karena ini yang ubah `kunjungan.status`.

`POST /rekam-medis/:id/addendum` (item #7) TIDAK broadcast — tidak menyentuh status kunjungan (sudah diverifikasi di item #7).

Ini WAJIB perubahan signature handler yang sudah ada (constructor handler perlu terima `*realtime.Hub` sebagai parameter tambahan, mirip pola `pool`/`emailSender` yang sudah ada) — ini retrofit SAH, bukan scope creep, karena eksplisit bagian requirement fitur ini.

## Tahapan implementasi

- **Tahap 1**: Migration `display_token_hash` + Hub core (register/unregister/broadcast logic, murni Go, belum nyambung HTTP/WS) + test concurrency Hub.
- **Tahap 2**: `POST /admin/klinik/:id/display-token/regenerate` + middleware dual-auth + retrofit `GET /klinik/:id/antrian`.
- **Tahap 3**: Endpoint `WS /ws` (upgrade, auth, register/unregister ke Hub, ping-pong keepalive).
- **Tahap 4**: Retrofit broadcast ke 5 titik panggil (perubahan signature handler item #5 & #7).
- **Tahap 5**: Regresi + E2E (WS client simulasi connect, staff lakukan write, assert notifikasi diterima; regenerate revoke token lama; dual-auth kedua channel).

## Skema/struktur data

KLINIK (tambahan kolom dari item #5 yang sudah ada):
string display_token_hash -- nullable, SHA256(token), NULL sampai regenerate pertama kali

## Edge case yang perlu dihandle

- **`display_token_hash` NULL** — SELALU gagal match di dual-auth, tidak ada bypass.
- **Hub goroutine tidak boleh blocking** — 1 client lambat/mati TIDAK BOLEH menghambat broadcast ke client lain.
- **WS upgrade gagal auth** — tolak SEBELUM upgrade HTTP selesai (401 biasa), bukan upgrade dulu baru kirim close frame.
- **Query param `displayToken` untuk WS, header `X-Display-Token` untuk REST** — WAJIB beda, WAJIB dikomentari eksplisit alasannya di kode (keterbatasan browser WebSocket API).
- **Koneksi WS existing tidak di-force-disconnect saat regenerate** — keputusan diambil langsung (lihat detail regenerate di atas).

## Testing

- Migration: kolom `display_token_hash` nullable, default NULL — lawan Postgres asli.
- Hub concurrency (WAJIB): banyak goroutine register/unregister/broadcast BERSAMAAN ke Hub yang sama → assert tidak ada data race (jalankan dengan `go test -race`), tidak ada panic, state akhir map konsisten (jumlah client terdaftar sesuai yang tersisa setelah unregister).
- `POST /regenerate`: sukses (200, displayToken mentah di response); hash di DB berubah; token lama (kalau ada dari panggilan sebelumnya) tidak lagi match; role non-admin → 403.
- Dual-auth: cookie staff valid → authorized + role check tetap berlaku (petugas/dokter/admin lolos, role lain kalau ada → 403); display-token valid → authorized tanpa role check; display-token salah/hash NULL → 401; keduanya tidak ada → 401.
- `GET /klinik/:id/antrian` retrofit: via cookie → shape lama (dengan id+pasienNama); via display-token → shape baru (tanpa id+pasienNama) — assert field-field itu benar-benar TIDAK ADA di response, bukan cuma null.
- WS: connect dengan cookie valid → sukses upgrade, terdaftar di Hub; connect dengan displayToken valid → sukses; connect tanpa auth valid → upgrade ditolak; disconnect → ter-unregister dari Hub (assert via broadcast berikutnya tidak coba kirim ke client yang sudah disconnect).
- Broadcast retrofit: untuk tiap 5 titik panggil, simulasikan 1 client WS terdaftar untuk klinik terkait → lakukan write via endpoint tsb → assert client menerima pesan `{"type":"queue_updated"}`. Addendum (rekam medis) → assert client TIDAK menerima pesan apapun.
- Regresi seluruh test suite yang sudah ada.

## Kriteria selesai

Seluruh 5 tahap selesai, seluruh skenario test PASS (termasuk `-race` untuk Hub), `go vet` lolos, regresi tidak ada yang kebreak, user verifikasi manual (buka 2 tab browser/WS client, satu display board satu staff, lakukan aksi di staff, lihat notifikasi masuk di display board).
