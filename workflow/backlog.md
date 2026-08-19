# Backlog — Modul RME & Antrian Klinik

Urutan wajib diikuti sesuai nomor (dependency teknis, bukan preferensi), kecuali ditandai "swappable". Metodologi: backend-first penuh, baru frontend — isolasi risiko koreksi + minimalkan context-switch antara 2 stack baru sekaligus.

Tiap item di sini akan jadi 1 `workflow/<nama_fitur>.md` spec begitu mulai dikerjakan (lihat `AGENTS.md` §1, §3) — checklist ini cuma peta urutan & cakupan, bukan spec detail.

## Backend

### Fase 1 — Infrastruktur

- [x] **1. Project scaffolding** — Go module, Gin skeleton, pgx pool, `sqlc` config, `golang-migrate` wiring (auto-run saat start), middleware error/requestId (`AGENTS.md` §7), `GET /health`.
- [x] **2. Auth & RBAC foundation** (Selesai)
  - [x] Migration: `users`, `sessions`, `user_roles`, `password_tokens`
  - [x] Session: login/logout, `GET /auth/me`, cookie `httpOnly`+`Secure`+`SameSite=Strict`
  - [x] Middleware RBAC (role-per-endpoint sesuai `api-contract.md`)
  - [x] Migration token invite/reset, integrasi Resend: `POST /admin/users` (tanpa password), `POST /auth/forgot-password`, `POST /auth/reset-password`
  - [x] **Seed admin pertama** — startup check idempotent via kode Go (`internal/bootstrap/admin.go`), generate & print invite token ke server log jika `password_hash` admin masih null; skip total jika admin sudah selesai setup.
- [x] **3. Audit trail infrastructure** (Selesai) — migration `audit_log` + `audit_log_tail` (genesis seed `SHA256('klinik-rme-genesis')`), helper hash-chain `internal/audit/` (`LockAuditLogTail FOR UPDATE` + insert + update 1 transaksi caller), DB trigger tolak `UPDATE`/`DELETE` di `audit_log`.

### Fase 2 — Domain inti

- [x] **4. Pasien** (Selesai) — migration `pasien` (+`version`, `deleted_at`, `consent_at`); `POST /pasien`, `GET /pasien/search` (nik+nama), `GET /pasien/:id`, `PATCH /pasien/:id` (optimistic lock); tersambung audit trail.
- [x] **5. Klinik & Antrian** (Selesai) — migration `klinik`, `queue_counter`, `kunjungan`; `GET /klinik/:id`; `POST /kunjungan` (atomic upsert counter); `GET /kunjungan/:id`, `GET /klinik/:id/antrian`; `POST /klinik/:id/panggil-berikutnya` (`FOR UPDATE SKIP LOCKED`, prioritas + `skip_count`); `POST /kunjungan/:id/lewati`; `POST /kunjungan/:id/tidak-hadir`.
- [x] **6. Realtime & Papan Antrian** (Selesai) — migration `klinik.display_token_hash`; Hub WS in-memory (`gorilla/websocket`); `POST /admin/klinik/:id/display-token/regenerate`; middleware dual-auth (cookie | `X-Display-Token`); broadcast notify-then-refetch di 5 handler mutasi antrian (`CreateKunjungan`, `PanggilBerikutnya`, `Lewati`, `TidakHadir`, `CreateRekamMedisAwal`); `GET /ws` endpoint WebSocket handler; E2E lifecycle test suite teruji 100%.
- [x] **7. Rekam Medis** (Selesai) — migration `rekam_medis`, `diagnosis`, `tindakan` (+ `uq_addendum_of_active` partial unique index + `uq_rekam_medis_root_per_kunjungan`); `POST /kunjungan/:id/rekam-medis`, `POST /rekam-medis/:id/addendum`; `GET /kunjungan/:id/rekam-medis` (leaf query), `GET /pasien/:id/riwayat`; tersambung audit trail; endpoint `[dokter]` only.
- [x] **8. Admin** (Selesai) — `GET`/`POST /admin/users`; **`POST /admin/users/:id/resend-invite`**; **`PATCH /admin/users/:id`** (koreksi nama/email — jalur recovery kalau email user salah/tidak bisa diakses, dipakai bareng resend-invite, gantiin skema admin-set-password yang sudah dihapus); `PATCH /admin/users/:id/roles`; `GET /admin/audit-log` (+pagination), `GET /admin/audit-log/:id`.
- [x] **9. Laporan Harian** (Selesai) — `GET /laporan/harian?tanggal=` (default hari ini, Asia/Jakarta); agregat `totalKunjungan`/`totalSelesai`/`totalTidakHadir` per klinik tunggal.

## Frontend

### Fase 1 — Fondasi

- [x] **10. Project scaffolding** — Angular CLI (standalone, Vitest, Tailwind v4), struktur folder `core/features/shared` (`AGENTS.md` §8), `environment.ts`, HTTP interceptor (`withCredentials`, handle `401`, parse `error.code`/`message`).
- [x] **11. Auth Infra & Core Shell** (Selesai Penuh) — Halaman login (`/login`); auth resolver `staffAuthResolver` di-scope ke staff route; `AuthService` berbasis Signal; `roleGuard` per role; `ForbiddenComponent` (`/forbidden`); UI Shell layout (`ShellComponent` dengan sidebar desktop collapsible, mobile drawer `hlm-sheet`, header `ClinicStatusIndicator`, user menu); index route `/` (`LandingComponent` dengan shortcut per role `petugas`, `dokter`, `admin`); global timezone `Asia/Jakarta` & locale `id-ID`; 44 unit tests (100% PASS).
- [x] **11b. Auth Recovery pages** (Selesai Penuh) — Halaman forgot-password (Tahap 1) dan set-password (Tahap 2), Reactive Forms validasi password (min 8 char), handling error token kontekstual (`INVALID_TOKEN` / missing token), unit test 100% PASS (64 unit tests).
- [x] **12. Profil / Account Settings** (Selesai Penuh) — halaman ganti password sendiri (`PATCH /auth/me/password`, `ProfilComponent` di `/profil`), info akun read-only, form reactive, error inline `INVALID_PASSWORD`, success/error toast, link user menu di Shell.
- [x] **13. RealtimeService** (Selesai Penuh) — wrap koneksi native WS (`GET /ws?klinikId=X`), Signal reactivity (`connectionStatus`, `lastUpdateAt`), exponential backoff (1s - 30s) + jitter ±20%, reconnect otomatis, `proxy.conf.json` dengan `"ws": true`, unit test (9 unit tests, 100% PASS).

### Fase 2 — Domain inti _(urutan ngikutin backend, boleh diprioritaskan ulang sesuai kebutuhan demo)_

- [x] **14. Pasien** (Selesai Penuh) — form registrasi (+consent), pencarian (nik/nama), halaman detail+riwayat ringkas, edit biodata.
- [x] **15. Antrian (staff-facing)** (Selesai Penuh) — tampilan antrian klinik (real-time + dual-trigger), tombol aksi dokter (panggil berikutnya, lewati) & petugas/admin (tandai tidak hadir dengan modal konfirmasi), indikator prioritas + status badge Spartan, form modal registrasi antrian via PasienDetail (validasi wajib alasan prioritas + proaktif disabled saat klinik tutup).
- [x] **16. Papan Antrian (publik)** (Selesai Penuh) — route terpisah tanpa guard staff (/papan-antrian), autentikasi dual-auth (display token via header `X-Display-Token` / WS query param), reaktivitas WebSocket (subscribe `RealtimeService`, refetch on notify), layout TV jarak jauh (nomor 3-digit zero-padded, live Jakarta clock).
- [x] **17. Rekam Medis** (Selesai Penuh) — form isi rekam medis SOAP (Reactive Forms + `FormArray` untuk `diagnosis[]`/`tindakan[]`), tampilan versi terkini (leaf query), modal addendum koreksi berantai, integrasi aksi pemeriksaan pada dashboard antrian & link riwayat rekam medis pasien.
- [ ] **18. Admin dashboard** — manajemen user (invite, **resend invite**, **edit email/nama**, roles), filter+detail audit log, regenerate display token.
- [x] **19. Laporan Harian** (Selesai Penuh) — halaman rekapitulasi harian staff (/laporan-harian), filter native date input (Asia/Jakarta), 3 inline stat cards (Total Kunjungan, Selesai, Tidak Hadir), auto-fetch on init & on change, error handling toast.
