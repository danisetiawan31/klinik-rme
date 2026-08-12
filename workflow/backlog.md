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
- [ ] **8. Admin** — `GET`/`POST /admin/users`; **`POST /admin/users/:id/resend-invite`**; **`PATCH /admin/users/:id`** (koreksi nama/email — jalur recovery kalau email user salah/tidak bisa diakses, dipakai bareng resend-invite, gantiin skema admin-set-password yang sudah dihapus); `PATCH /admin/users/:id/roles`; `GET /admin/audit-log` (+pagination), `GET /admin/audit-log/:id`.
- [ ] **9. Laporan Harian** — ⚠️ **ditunda, diputuskan pas mulai item ini.** Shape request/response belum eksplisit di `api-contract.md` (baru sebatas "harus ada" di PRD). Tidak blocking apa pun sebelumnya — `AGENTS.md` §1 ("ambigu → tanya") sudah menjamin ini nggak kelewat diam-diam saat waktunya tiba.

## Frontend

### Fase 1 — Fondasi

- [x] **10. Project scaffolding** — Angular CLI (standalone, Vitest, Tailwind v4), struktur folder `core/features/shared` (`AGENTS.md` §8), `environment.ts`, HTTP interceptor (`withCredentials`, handle `401`, parse `error.code`/`message`).
- [ ] **11. Auth pages & guard** — halaman login, set-password (dipakai invite & reset, entrypoint beda link), forgot-password; resolver `GET /auth/me` di-scope staff route (bukan global); route guard per role.
- [ ] **12. Profil / Account Settings** — halaman ganti password sendiri (`PATCH /auth/me/password`, butuh password lama — beda flow dari set-password di item 11 yang berbasis token, bukan password lama), diakses semua role setelah login.
- [ ] **13. RealtimeService** — wrap koneksi WS, reconnect+backoff; `proxy.conf.json` dengan `"ws": true`.

### Fase 2 — Domain inti _(urutan ngikutin backend, boleh diprioritaskan ulang sesuai kebutuhan demo)_

- [ ] **14. Pasien** — form registrasi (+consent), pencarian (nik/nama), halaman detail+riwayat ringkas, edit biodata.
- [ ] **15. Antrian (staff-facing)** — tampilan antrian klinik, tombol panggil berikutnya, tandai tidak hadir, indikator prioritas.
- [ ] **16. Papan Antrian (publik)** — route terpisah tanpa guard staff, subscribe `RealtimeService`, refetch on notify.
- [ ] **17. Rekam Medis** — form isi rekam medis (Reactive Forms + `FormArray` untuk `diagnosis[]`/`tindakan[]`), tampilan versi terkini, form addendum, riwayat pasien.
- [ ] **18. Admin dashboard** — manajemen user (invite, **resend invite**, **edit email/nama**, roles), filter+detail audit log, regenerate display token.
- [ ] **19. Laporan Harian** — gantung ke shape item 9.
