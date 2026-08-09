# Done Log — Modul RME & Antrian Klinik

## Scaffolding Backend — Tahap 1: Foundation

- **Apa yang Dikerjakan**: Inisialisasi Go module (`github.com/danisetiawan31/klinik-rme`), struktur folder `cmd/server/`, `internal/config/`, `internal/db/`, config loader dengan validasi env vars ketat tanpa silent default, pgx pool (`pgx/v5`), graceful shutdown.
- **Verifikasi**: `go test -v ./internal/config/...`, run `./cmd/server` tanpa env -> Fatal error `missing required environment variable`.
- **Catatan**: Belum ada Gin & HTTP layer.

---

## Scaffolding Backend — Tahap 2: Data Layer Tooling

- **Apa yang Dikerjakan**: Setup `sqlc.yaml` v2 (`internal/db/generated/`), folder `migrations/` & `queries/`, wrapper library `golang-migrate` (`internal/db/migration.go`) dengan URL `file://` cross-platform, pre-check file `.sql`, auto-run migrasi saat startup server, integration test `migration_test.go` via `testcontainers-go` (`postgres:16-alpine`).
- **Verifikasi**: `sqlc generate` -> Exit code 0, `go test -v ./internal/db/...` -> PASS 100%, `go vet ./...` -> PASS.
- **Catatan**: Pre-check direktori & file `.sql` mencegah error swallow pada path yang salah.

---

## Scaffolding Backend — Tahap 3: HTTP Layer

- **Apa yang Dikerjakan**: Setup Gin skeleton di `internal/api/router.go`, middleware `RequestID` (UUID v4 per request), middleware `ErrorHandler` & `GlobalRecovery` (format `{ "error": { "code", "message", "requestId" } }` tanpa bocor raw error/credentials), endpoint `GET /health` (200 OK / 503 Service Unavailable), graceful shutdown HTTP server (timeout 5s).
- **Verifikasi**: `go test -v ./internal/api/...` -> PASS 100%, `go vet ./...` -> PASS.
- **Catatan**: `context.WithTimeout(c.Request.Context(), 2*time.Second)` di `/health` untuk fail-fast saat DB menggantung/retry koneksi.

---

## Auth & RBAC Foundation — Tahap 1: Skema & Helper Keamanan Inti

- **Apa yang Dikerjakan**: Migrasi 4 tabel (`users`, `user_roles`, `sessions`, `password_tokens`), package `internal/auth/` (`bcrypt` cost 12 & mapping `ErrMismatchedHashAndPassword`, `token` crypto/rand 128-bit & SHA256 hex), query sqlc dasar (`users`, `user_roles`, `sessions`), integration test `migration_test.go` via `testcontainers-go` menguji migrasi domain & constraint DB (`UNIQUE`, `FK`, `CHECK`, `NULLABLE`).
- **Verifikasi**: `go test -v ./internal/auth/...` -> PASS 100%, `go test -v ./internal/db/...` -> PASS 100%, `go vet ./...` -> PASS.
- **Catatan**: `CHECK` constraint `role` & `type` dipasang di DB; `sqlc.yaml` disesuaikan ke `schema: "migrations/*.up.sql"` agar menyaring `.down.sql` (`DROP TABLE`).
