# Done Log — Modul RME & Antrian Klinik

## Scaffolding Backend — Tahap 1: Foundation

- **Fitur**: Inisialisasi Go module, struct `cmd/server/`, `internal/config/`, `internal/db/` (pgx pool), config loader env strict, graceful shutdown.
- **Verifikasi**: `go test -v ./internal/config/...` PASS 100%.

---

## Scaffolding Backend — Tahap 2: Data Layer Tooling

- **Fitur**: Config `sqlc.yaml` v2, `golang-migrate` wrapper dengan pre-check `.sql`, auto-run migrasi startup, integration test `testcontainers-go`.
- **Verifikasi**: `sqlc generate` PASS, `go test -v ./internal/db/...` PASS 100%, `go vet ./...` PASS.

---

## Scaffolding Backend — Tahap 3: HTTP Layer

- **Fitur**: Gin router, middleware `RequestID`, `ErrorHandler` & `GlobalRecovery` (amplas raw error), endpoint `GET /health` (dual route `/health` & `/api/v1/health`), graceful shutdown (5s).
- **Verifikasi**: `go test -v ./internal/api/...` PASS 100%, `go vet ./...` PASS.

---

## Auth & RBAC Foundation — Tahap 1-5 (Selesai Penuh)

- **Tahap 1 (Skema & Helper)**: Migrasi 4 tabel (`users`, `user_roles`, `sessions`, `password_tokens`), package `internal/auth/` (`bcrypt` cost 12, `token` crypto/rand 128-bit, SHA256 hex).
- **Tahap 2 (Session & RBAC)**: Middleware `Authenticate` (sliding 2h, hard cap 24h, cookie `httpOnly`+`SameSite=Strict`+`isSecure` kondisional), middleware `RequireRole`, endpoint `POST /auth/login` (identical 401), `POST /auth/logout`, `GET /auth/me`, `PATCH /auth/me/password`.
- **Tahap 3 (Invite, Forgot/Reset, Resend)**: Resend Go SDK (`v3.12.0`, timeout 10s), `POST /admin/users` (transaksi DB `pool.Begin` atomic, error mapping 23514->400, 23505->409), `POST /auth/forgot-password` (selalu 200 generik, log token mentah), `POST /auth/reset-password` (atomic token update).
- **Tahap 4 (Seed Admin Bootstrap)**: Startup check idempotent via Go (`internal/bootstrap/admin.go`), insert admin (`password_hash=NULL`) & role, generate & log invite token jika password NULL, skip total jika password terisi.
- **Tahap 5 (Integrasi & Regresi Menyeluruh)**: Test E2E `TestAuthAndRBACFoundation_FullLifecycleE2E` (skenario a-m berurutan dalam 1 container Postgres), regresi penuh `go test -v ./...` & `go vet ./...` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:

- `CHECK` constraint dipasang di DB (`user_roles.role`, `password_tokens.type`).
- `sqlc.yaml` schema disesuaikan ke `"migrations/*.up.sql"`.
- `InsertPasswordToken` disederhanakan 4 parameter (drop `created_at`, DB `DEFAULT now()`).
- Seed admin diimplementasikan sebagai kode Go startup (bukan migration file) karena bergantung env `SEED_ADMIN_EMAIL`.
- Cookie `isSecure` di-set kondisional (`TLS != nil || X-Forwarded-Proto == "https"`).

---

## Audit Trail Infrastructure — Tahap 1-3 (Selesai Penuh)

- **Tahap 1 (Migration & Trigger)**: Migrasi `audit_log_tail` (CHECK id=1, genesis seed `SHA256('klinik-rme-genesis')`) & `audit_log` (polymorphic `record_id`, FK `actor_user_id`), PL/pgSQL function & trigger `BEFORE UPDATE OR DELETE` raising exception (`audit_log is append-only`).
- **Tahap 2 (Helper Hash-Chain)**: Package `internal/audit/` dengan fungsi `Record(...)`, locking `LockAuditLogTail` (`FOR UPDATE`), kalkulasi SHA256 hex hash dari struct JSON `auditHashInput`, insert `audit_log`, update `audit_log_tail.last_hash` di transaksi milik caller.
- **Tahap 3 (Verifikasi Trigger & Regresi Menyeluruh)**: Test integrasi 6 skenario (single call, sequential linkage, 10-goroutine concurrency serialization, robustness formula, nil `beforeData`, dan trigger protection pada row buatan aplikasi asli). Regresi penuh `go test -v ./...` & `go vet ./...` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:

- Sesuai spec 100%, tidak ada deviasi.

---

## Pasien — Tahap 1-3 (Selesai Penuh)

- **Tahap 1 (Migration & Query Dasar)**: Migration `pasien` (`id`, `nik` nullable tanpa unique, `nama`, `tanggal_lahir`, `jenis_kelamin` CHECK IN ('L','P'), `alamat`, `no_telp`, `consent_at` NOT NULL, `version` DEFAULT 1, `deleted_at` nullable). Query sqlc (`InsertPasien`, `GetPasienByID`, `GetPasienByIDIncludingDeleted`, `SearchPasien` NIK exact + nama ILIKE + AND + pagination + deleted_at IS NULL, `UpdatePasienOptimistic`).
- **Tahap 2 (Endpoint & Audit Integration)**: Endpoint `POST /pasien` [petugas, admin] (consent=true mandatory -> 400 CONSENT_REQUIRED, audit `aksi='create'`), `GET /pasien/search` [petugas, dokter, admin], `GET /pasien/:id` [petugas, dokter, admin] (riwayatKunjunganRingkas: []), `PATCH /pasien/:id` [petugas, admin] (optimistic lock version matching, silent ignore consent/consentAt, audit `aksi='update'` snapshot before/after).
- **Tahap 3 (Integrasi & Regresi Menyeluruh)**: Test E2E `TestPasienFullLifecycle_E2E` (skenario a-h berurutan dalam container Postgres terisolasi khusus, assert strict hash-chain linkage `row2.previous_hash == row1.hash_entry`), Concurrency test optimistic lock (2 parallel PATCH 1 success 200, 1 fail 409), regresi penuh `go test -v ./...` & `go vet ./...` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:

- Pre-fetch `GetPasienByID` pada `PATCH /pasien/:id` dilakukan di LUAR transaksi sebelum `pool.Begin`. Tetap 100% aman karena atomic check `WHERE id = $1 AND version = $2` pada `UpdatePasienOptimistic` (di dalam transaksi) tetap menjadi penjaga akhir race condition, menggunakan `req.Version` dari client (bukan dari hasil pre-fetch).
- Penanganan 0 rows returned dari `UpdatePasienOptimistic`: alih-alih mengasumsikan 409 secara langsung, handler mengeksekusi `q.GetPasienByIDIncludingDeleted(ctx, int32(id))` untuk membedakan secara presisi antara HTTP 404 (`PASIEN_NOT_FOUND` — pasien tidak ada / ter-soft-delete) vs HTTP 409 (`OPTIMISTIC_LOCK_FAILED` — konflik versi). Ini mengantisipasi jika kelak ada fitur soft-delete pasien di masa mendatang.
- Warning duplikasi NIK adalah tanggung jawab FE pre-submission check (`GET /pasien/search?nik=`). Backend `POST /pasien` sengaja menerima NIK ganda tanpa memblokir atau mengembalikan sinyal duplikasi.

