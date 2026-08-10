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
