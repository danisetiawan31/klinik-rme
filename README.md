# Modul RME & Antrian Klinik

![CI](https://github.com/danisetiawan31/klinik-rme/actions/workflows/ci.yml/badge.svg)

Modul internal untuk klinik kecil yang mencatat rekam medis elektronik (RME) terstruktur, mengelola antrian real-time tanpa duplikasi nomor, dan menjaga audit trail tamper-evident atas setiap perubahan data klinis.

Bukan integrasi resmi ke SATUSEHAT (butuh akses API Kemenkes, di luar jangkauan project ini) - skema data selaras konsep SATUSEHAT, tapi berdiri sendiri sebagai sistem internal 1 klinik.

> Seluruh data demo/testing di repo ini adalah data fiktif/sintetis, bukan data pasien asli.

## Fitur MVP

- Autentikasi (invite user via email, reset password, ganti password sendiri) + role-based access (petugas, dokter, admin)
- Manajemen pasien: registrasi, pencarian (NIK/nama), warning duplikasi NIK
- Antrian real-time: nomor antrian atomic, prioritas, handling no-show
- Papan antrian publik (display board), autentikasi terpisah dari sesi staff
- Rekam medis terstruktur per kunjungan, koreksi lewat addendum (bukan edit langsung)
- Audit trail hash-chain, tamper-evident
- Dashboard admin: manajemen user, filter audit log

Detail lengkap ada di [`docs/PRD.md`](docs/PRD.md).

## Tech Stack

| Layer      | Teknologi                                                         |
| ---------- | ------------------------------------------------------------------ |
| Backend    | Go + Gin                                                            |
| DB access  | sqlc + pgx/v5                                                       |
| Database   | PostgreSQL, migration via golang-migrate (auto-run saat start)     |
| Frontend   | Angular (standalone components, Signals, Vitest), Tailwind v4      |
| Realtime   | WebSocket, in-memory hub per proses                                |
| Email      | Resend (invite user, forgot-password)                              |
| Deployment | Docker multi-stage, Nginx reverse proxy, docker-compose, GitHub Actions CI |

Detail arsitektur, ERD, dan keputusan desain teknis (concurrency, locking, audit trail) ada di [`docs/TDD.md`](docs/TDD.md).

## Struktur Project (Monorepo)

```
klinik-rme/
├── AGENTS.md              # Rules kerja untuk AI coding agent (Antigravity)
├── docs/
│   ├── PRD.md              # Problem statement & scope MVP
│   ├── TDD.md               # Arsitektur, ERD, keputusan desain teknis
│   └── api-contract.md      # Kontrak API frontend <-> backend
├── workflow/
│   ├── backlog.md          # Fitur yang sudah diputuskan, belum dikerjakan
│   └── done.md              # Fitur yang sudah selesai
├── backend/                # Layanan Backend Go
│   ├── cmd/server/          # Entrypoint aplikasi
│   ├── internal/
│   │   ├── config/          # Loader & validasi environment variables
│   │   ├── db/              # Pool koneksi, migration runner, generated/ (sqlc)
│   │   └── api/             # Router, middleware, handler HTTP
│   ├── migrations/          # File migrasi golang-migrate
│   ├── queries/             # File query SQL untuk sqlc
│   ├── go.mod / go.sum
│   └── .env.example
└── frontend/               # Aplikasi Frontend Angular (placeholder)
```

## Prasyarat

- Go 1.23+
- PostgreSQL (lokal atau container)
- Docker - dibutuhkan untuk menjalankan test integrasi (`testcontainers-go` men-spin-up PostgreSQL asli saat `go test`)

## Menjalankan Secara Lokal

```bash
# 1. Clone & masuk ke folder backend
git clone https://github.com/danisetiawan31/klinik-rme.git
cd klinik-rme/backend

# 2. Siapkan environment variables
cp .env.example .env
# lalu edit .env sesuai kredensial PostgreSQL lokal kamu

# 3. Jalankan server
# (migration otomatis jalan saat start, tidak perlu langkah CLI terpisah)
go run ./cmd/server
```

Verifikasi server hidup:

```bash
curl http://localhost:8080/health
# -> {"status":"ok","db":"ok"}
```

## Environment Variables

Lihat [`backend/.env.example`](backend/.env.example) untuk daftar lengkap beserta keterangan wajib/opsional. Semua variabel database dan `TZ` wajib diisi - server akan gagal start dengan pesan jelas kalau ada yang kosong (lihat `backend/internal/config/config.go`).

## Testing

```bash
cd backend
go vet ./...
go test -v ./...
```

Test yang menyentuh concurrency (atomic counter antrian, klaim `SKIP LOCKED`, audit hash-chain) dan migration runner dijalankan lawan PostgreSQL asli via `testcontainers-go` - bukan mock, bukan SQLite. Pastikan Docker aktif sebelum menjalankan test.

## Status & Roadmap

Project ini dikerjakan bertahap, backend-first. Status fitur yang sedang berjalan ada di [`workflow/backlog.md`](workflow/backlog.md), fitur yang sudah selesai dicatat di [`workflow/done.md`](workflow/done.md).

## Lisensi

[MIT](LICENSE)
