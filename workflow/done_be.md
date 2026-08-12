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
- _Catatan Retroaktif (Refactoring Router)_: Middleware `Authenticate` awalnya dibungkus `if q != nil` di `router.go`. Saat fitur Klinik & Antrian dikembangkan, hal ini direfaktur: pengaman `q == nil` dipindahkan ke dalam fungsi `middleware.Authenticate` itu sendiri (me-return 500 SERVER_ERROR), dan pembungkus `if q != nil` di `router.go` dihapus agar pendaftaran middleware auth terjamin 100% unconditional di seluruh protected route group (`auth`, `admin`, `pasien`, `klinik`, `kunjungan`).

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
- _Catatan Retroaktif (Refactoring Router)_: Refactoring registrasi middleware `Authenticate` di `router.go` secara otomatis menyempurnakan registrasi grup route `/pasien` agar terpasang secara 100% unconditional.

---

## Klinik & Antrian — Tahap 1-4 (Selesai Penuh)

- **Tahap 1 (Migration, Seed Klinik, Query Dasar)**: Migration `000008_create_klinik`, `000009_create_queue_counter`, `000010_create_kunjungan`. Config env `KLINIK_NAMA`, `KLINIK_JAM_BUKA`, `KLINIK_JAM_TUTUP` (validasi HH:MM strict). Helper `bootstrap.SeedKlinik` (idempotent). Query sqlc (`InsertKunjungan`, `GetKunjunganByID`, `ListKunjunganByKlinikAndTanggal`, `ClaimNextKunjungan` FOR UPDATE SKIP LOCKED, `UpdateKunjunganSkip`, `UpdateKunjunganTidakHadir`). Test domain database `TestKlinikAntrianDomain_RealPostgreSQL` PASS 100%.
- **Tahap 2 (Endpoint Non-Klaim)**: Query baru `ListKunjunganWithPasienNamaByKlinikAndTanggal` (JOIN kunjungan+pasien). Endpoint `GET /klinik/:id` [petugas, dokter, admin], `POST /kunjungan` [petugas, admin] (validasi jam tutup & pasien soft-delete, atomic upsert queue counter), `GET /kunjungan/:id` [petugas, dokter, admin], `GET /klinik/:id/antrian` [petugas, dokter, admin]. Integration test `TestKlinikAntrianEndpoints_Integration` PASS 100%.
- **Tahap 3 (Endpoint Klaim & Resolusi + Test Konkurensi)**: Endpoint `POST /klinik/:id/panggil-berikutnya` [dokter saja] (dokterId dari session, return 204 jika antrian kosong), `POST /kunjungan/:id/lewati` [dokter saja] (status 'dipanggil' -> 'menunggu' + skipCount+1), `POST /kunjungan/:id/tidak-hadir` [dokter, admin]. Test integrasi & konkurensi `TestKlinikAntrianClaimEndpoints_Integration` (5 dokter klaim bersamaan: 0 collision, speedup ratio layer DB 3.00x) PASS 100%.
- **Tahap 4 (Integrasi E2E Lifecycle & Regresi Menyeluruh)**: Test E2E `TestKlinikAntrianFullLifecycle_E2E` (10 skenario a-j berurutan dalam container Postgres terisolasi khusus: registrasi pasien, pendaftaran antrian normal vs prioritas, panggil prioritas duluan, lewati, panggil ulang prioritas via tie-breaker `is_priority DESC`, resolusi tidak-hadir admin, verifikasi kondisi akhir antrian, dan verifikasi DB langsung 0 row `audit_log` tercatat). Regresi penuh `go test -v -p 1 ./...` & `go vet ./...` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:

- Endpoint `POST /kunjungan/:id/lewati` [dokter] ditambahkan ke `docs/api-contract.md` oleh user untuk menangani skenario pasien dipanggil tidak muncul di ruang periksa (status kembali ke `'menunggu'`, `skip_count` bertambah 1).
- Seed klinik diimplementasikan sebagai fungsi Go startup (`bootstrap.SeedKlinik`) di `internal/bootstrap/klinik.go`, bukan migration file, karena nilainya dinamis bergantung pada environment variable.
- Refactoring `router.go`: menghapus pembungkus `if q != nil` pada pendaftaran middleware `Authenticate` dan memindahkan nil-check internal ke dalam `middleware.Authenticate` (me-return `500 SERVER_ERROR`). Perubahan ini berlaku retroaktif menyempurnakan grup route Auth & RBAC Foundation dan Pasien agar pendaftaran middleware auth terjamin 100% unconditional.
- Fix infrastruktur DB pool: menyetel `config.MaxConns = 20` secara eksplisit pada `internal/db/db.go` (`NewPool`) untuk menjamin throughput konkurensi paralel non-blocking pada pengujian SKIP LOCKED.
- Optimasi skema DB: menambahkan index komposit `idx_kunjungan_klinik_tanggal_status` pada tabel `kunjungan` dan constraint `ON DELETE CASCADE` pada FK `queue_counter → klinik` sebagai bentuk _defensive engineering_.
- Pengecualian audit trail: operasi `kunjungan`, `klinik`, dan `queue_counter` secara eksplisit dikecualikan dari `audit.Record` agar tidak memicu lock contention pada `audit_log_tail`. Terverifikasi murni 0 row audit log tercatat di test E2E (skenario j).

---

## Rekam Medis — Tahap 1-5 (Selesai Penuh)

- **Tahap 1 (Migration, Query dasar, Extend Audit Aksi)**: Migration `000011_create_rekam_medis`, `000012_create_diagnosis`, `000013_create_tindakan`, `000014_extend_audit_log_aksi` (`aksi IN ('create','update','addendum')`). Partial unique index `uq_addendum_of_active` & `uq_rekam_medis_root_per_kunjungan`. Query sqlc (`rekam_medis.sql`, `diagnosis.sql`, `tindakan.sql`).
- **Tahap 2 (POST /kunjungan/:id/rekam-medis & Transisi Status Kunjungan)**: Endpoint `POST /kunjungan/:id/rekam-medis` [dokter saja] (body `keluhan`, `hasilPemeriksaan`, `diagnosis[]`, `tindakan[]`). Transaksi atomic: insert `rekam_medis` -> `UPDATE kunjungan SET status='selesai'` -> `audit.Record(aksi='create')`. Reactive 409 pada duplikasi root record.
- **Tahap 3 (POST /rekam-medis/:id/addendum & Backend Merge Strategy)**: Endpoint `POST /rekam-medis/:id/addendum` [dokter saja] (`alasanAddendum` mandatory). Logika merge backend: absent keys carry-over dari parent, `[]` dikosongkan. Transaksi atomic insert leaf baru -> `audit.Record(aksi='addendum')`. Reactive 409 pada konflik `uq_addendum_of_active`.
- **Tahap 4 (GET /kunjungan/:id/rekam-medis & GET /pasien/:id/riwayat)**: Endpoint `GET /kunjungan/:id/rekam-medis` [dokter saja] (leaf query traversal `NOT EXISTS` + `deleted_at IS NULL`) & `GET /pasien/:id/riwayat` [dokter saja] (list kunjungan ber-rekam-medis).
- **Tahap 5 (Integrasi E2E Lifecycle, Test Konkurensi & Regresi Menyeluruh)**: Test konkurensi `TestRekamMedisAddendum_Concurrency` (2 goroutines addendum bersamaan: 1 sukses 201, 1 gagal 409 via constraint `uq_addendum_of_active`), Test E2E `TestRekamMedisFullLifecycle_E2E` (skenario a-j berurutan: auth, pasien, kunjungan, panggil antrian, create RM awal, addendum 1 & 2, fetch leaf/riwayat, RBAC non-dokter 403, audit log linkage). Regresi penuh `go test -v -p 1 ./...` & `go vet ./...` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:

- Extension constraint `audit_log.aksi` via migration `000014_extend_audit_log_aksi` (`CHECK (aksi IN ('create', 'update', 'addendum'))`).
- Dual partial unique index: `uq_addendum_of_active` (max 1 addendum aktif per parent) & `uq_rekam_medis_root_per_kunjungan` (max 1 root record per kunjungan).
- Backend merge strategy: pointer/nullable-aware types (`*[]DiagnosisInput`, `*[]TindakanInput`, `*string`) untuk membedakan "absent key" (carry-over) vs "array kosong `[]`" (dikosongkan).
- RBAC Enforcement: Seluruh endpoint Rekam Medis terisolasi [dokter] saja. Role non-dokter me-return 403 FORBIDDEN.
- Audit compliance: Addendum dicatat dengan `aksi='addendum'` (bukan 'create'/'update').

---

## Realtime & Papan Antrian — Tahap 1-5 (Selesai)

- **Tahap 1 (Migration & Hub Core)**: Migration `000015_add_display_token_hash_to_klinik`. Query sqlc `UpdateKlinikDisplayTokenHash` & `GetKlinikDisplayTokenHash`. Package `internal/realtime/` (`Client`, `Hub` single goroutine `Run(ctx)`, safe unregister, non-blocking broadcast, & guard `closed` channel).
- **Tahap 2 (Regenerate Endpoint, Dual-Auth Middleware, Retrofit GET Antrian)**: Endpoint `POST /admin/klinik/:id/display-token/regenerate` [admin] (overwrite hash & return raw token). Middleware `DualAuth` (cookie staff vs `X-Display-Token`/`?displayToken=` + `subtle.ConstantTimeCompare`). Bypass `RequireRole` khusus `auth_channel="display-token"` (disertai komentar eksplisit). Retrofit `GET /klinik/:id/antrian` (jalur cookie: lengkap `id` & `pasienNama`; jalur display-token: terfilter publik tanpa `id` & `pasienNama`).
- **Tahap 3 (Endpoint WebSocket Handler)**: Handler `GET /ws?klinikId=X` dengan Gorilla WebSocket upgrader (strict default same-origin check). Registered di root engine Gin dengan `DualAuth` + `RequireRole`. Supporting nilable `hub` pada `SetupRouter` (100% kompatibel dengan 19 call site di 15 file test).
- **Tahap 4 (Retrofit Broadcast Trigger di 5 Endpoint)**: Mengintegrasikan `if h.hub != nil { h.hub.BroadcastToKlinik(...) }` non-blocking di 5 handler mutasi antrian (`CreateKunjungan`, `PanggilBerikutnya`, `Lewati`, `TidakHadir`, ` CreateRekamMedisAwal`). `CreateAddendum` sengaja tidak disentuh (0 broadcast).
- **Tahap 5 (E2E Lifecycle & Regresi Penuh)**: Test suite E2E `realtime_e2e_lifecycle_test.go` memverifikasi alur penuh langkah a-i (multi-client broadcast lintas channel, token lama tetap aktif di socket WS existing, disconnect handling unregister, REST vs WS behavior beda terhadap revoked token). Regresi penuh `go test -v ./...` dan `go vet ./...` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:

- **Post-Shutdown Deadlock Guard**: Guard `h.closed` pada method publik `Hub` (`RegisterClient`, `UnregisterClient`, `BroadcastToKlinik`) mencegah blocking/deadlock saat graceful shutdown server.
- **Refactoring Shared Helper `validateStaffSession`**: Memindahkan validasi sesi cookie staff & sliding expiration ke helper terpusat `auth.go` yang direuse oleh `Authenticate` & `DualAuth`.
- **Timing Attack Hardening**: Penggunaan `crypto/subtle.ConstantTimeCompare` pada `DualAuth` saat membandingkan hash display token.
- **Signature `SetupRouter` Nilable Hub**: `SetupRouter` menerima `hub` nilable dengan default error fallback graceful pada `/ws`, disertai pembaruan 19 lokasi pemanggilan call site test suite.
- **Nil-Guarded Broadcast Triggers**: Seluruh pemanggilan broadcast di 5 handler bisnis dijaga `if hub != nil` untuk mencegah panic nil-pointer dereference pada test suite lama.

---

## Laporan Harian — Selesai

- **Fitur**: Endpoint `GET /laporan/harian?tanggal=` [petugas, dokter, admin] — rekap harian kunjungan per klinik (`totalKunjungan`, `totalSelesai`, `totalTidakHadir`). Query baru `GetLaporanHarian` (COUNT ... FILTER) di `kunjungan.sql`, resolusi klinik via `GetSingleKlinik()` (reuse pola dari `CreateKunjungan`), default `?tanggal=` ke hari ini (Asia/Jakarta) kalau kosong, validasi format `YYYY-MM-DD` eksplisit -> 400 `TANGGAL_INVALID` kalau invalid.
- **Verifikasi**: `TestLaporanHarian_Integration` (5 skenario: mixed status, tanggal tanpa data, default hari ini, format invalid, tanpa auth) PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- Sesuai spec 100%, tidak ada deviasi.

---

## Fitur Admin — Tahap 1-4 (Selesai)

- **Tahap 1 (User Management Dasar)**: Endpoint `GET /admin/users` (list user + roles dengan pagination `page`/`limit`), `PATCH /admin/users/:id` (partial update nama/email dengan reactive 409 conflict check), dan `POST /admin/users/:id/resend-invite` (invalidasi token invite lama `consumed_at = now()` & generate token 128-bit baru dalam 1 transaksi DB, kirim email via Resend best-effort setelah commit).
- **Tahap 2 (Roles Management)**: Endpoint `PATCH /admin/users/:id/roles` (full replace roles dalam 1 transaksi DB). Input deduplication otomatis (`dedupeStringSlice`), validasi roles non-kosong (400 `ROLES_CANNOT_BE_EMPTY`), mutual exclusion admin + dokter (400 `MUTUAL_EXCLUSION_ROLES`), dan last-admin guard pre-check (400 `LAST_ADMIN_GUARD` cegah sistem kehilangan admin terakhir). Real-time RBAC enforcement (middleware `RequireRole` query fresh DB tiap request).
- **Tahap 3 (Audit Log Read)**: Endpoint `GET /admin/audit-log` (list summary audit log tanpa `beforeData`/`afterData`, filter optional `tabelTarget`, `recordId`, `actorId`, pagination) dan `GET /admin/audit-log/:id` (detail lengkap audit log termasuk JSONB `beforeData`/`afterData` dan `hashEntry`). Filter int invalid (`?recordId=abc`) diabaikan secara aman sebagai filter tanpa error 500.
- **Tahap 4 (Integrasi E2E Lifecycle & Regresi Penuh)**: Test suite E2E `TestAdminFullLifecycle_E2E` memverifikasi alur penuh langkah a-h (login admin, GET users, PATCH user, resend invite + invalidasi token lama, PATCH roles & pembuktian real-time RBAC tanpa re-login, LAST_ADMIN_GUARD, GET audit log list & detail). Regresi penuh `go test -v -p 1 ./...` dan `go vet ./...` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- **Input Role Deduplication**: Array role input di `PATCH /admin/users/:id/roles` dideduplikasi otomatis sebelum validasi & insert untuk mencegah pelanggaran constraint komposit PK `(user_id, role)` (`23505 unique_violation`).
- **Resend Invite Transactionality**: Invalidasi token lama & insert token invite baru dieksekusi dalam 1 transaksi DB eksplisit. Dispatch email dilakukan best-effort setelah commit untuk mencegah kegagalan SMTP menggagalkan transaksi DB.
- **Robust Pre-check Last-Admin Guard**: Penghitungan jumlah admin aktif dilakukan secara preemptive (`q.CountUsersWithRole(ctx, "admin")`) khusus di operasi PATCH roles karena operasi ini tergolong jarang dan tidak concurrency-sensitive.
- **Safety Handling Query Parameters**: Parameter integer optional pada list audit log (`recordId`, `actorId`) mengabaikan string non-numerik (seperti `abc`) sehingga fallback menjadi un-filtered secara aman tanpa menyebabkan error DB/500.





