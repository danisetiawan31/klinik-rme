# Auth & RBAC Foundation

## Konteks & tujuan

Fondasi otentikasi & otorisasi untuk seluruh modul — dipakai semua endpoint berikutnya (item #3-9 backend, #11-19 frontend). Mencakup session cookie, role-based access control, invite/reset password via email, dan bootstrap admin pertama.

## Requirement

### Migration (4 tabel, sesuai ERD di docs/TDD.md)

- `users`, `user_roles`, `sessions`, `password_tokens`

### Endpoint (shape request/response ikuti docs/api-contract.md persis, section Auth)

- `POST /auth/login` [public]
- `POST /auth/logout` [any authenticated]
- `GET /auth/me` [any authenticated]
- `PATCH /auth/me/password` [any authenticated]
- `POST /auth/forgot-password` [public]
- `POST /auth/reset-password` [public]
- `POST /admin/users` [admin] — invite user baru, tanpa password

### Middleware RBAC

- Terapkan role check sesuai anotasi role di setiap endpoint pada `docs/api-contract.md` (`[public]`, `[any authenticated]`, `[admin]`, dst) — bukan cuma endpoint di atas, siapkan middleware generik yang dipakai ulang endpoint-endpoint fitur berikutnya juga.

### Seed admin bootstrap

- Sesuai `workflow/backlog.md` item #2 — migration insert 1 baris admin (`password_hash` null) dari env `SEED_ADMIN_EMAIL` (**wajib** ditambahkan sebagai required config baru, ikuti pola validasi ketat yang sudah ada di `internal/config`). `nama` admin pakai default `"Administrator"` (boleh diedit manual di DB kalau perlu, tidak perlu env var terpisah).
- Startup check idempotent — detail lengkap di section "Edge case" di bawah, WAJIB dibaca sebelum implementasi Tahap 4, jangan diimplementasikan dari asumsi sendiri.

### Integrasi Resend

- Belum ada library resmi yang ditentukan di `docs/TDD.md`/`AGENTS.md`. Di awal Tahap 3, investigasi dulu opsi library Go resmi/populer untuk Resend, laporkan pilihan + alasan ke user, **tunggu konfirmasi eksplisit sebelum menambahkan dependency ini** (sesuai `AGENTS.md` §9 — dependency baru wajib dikonfirmasi).

### Konvensi teknis yang WAJIB diikuti (sudah ditetapkan di AGENTS.md §7, jangan diulang tafsir sendiri)

Password hashing (bcrypt cost 12), token generation (`crypto/rand`, 128-bit, base64url, disimpan SHA256), cookie session (`httpOnly`+`Secure`+`SameSite=Strict`, sliding expiration + hard cap 24 jam), TTL token (invite 7 hari, reset 1 jam), forgot-password selalu 200 generik + tidak pernah return token mentah, `POST /admin/users` boleh return `inviteLink` mentah (beda konteks, caller sudah admin ter-otentikasi).

### Eksplisit DI LUAR SCOPE spec ini (jangan dikerjakan, itu item #8 nanti)

`GET /admin/users` (list), `POST /admin/users/:id/resend-invite`, `PATCH /admin/users/:id`, `PATCH /admin/users/:id/roles`, `GET /admin/audit-log*`.

## Tahapan implementasi

- **Tahap 1 (Skema & helper keamanan inti)**: 4 migration di atas. Helper bcrypt (hash/verify). Helper token generation + hashing. Query sqlc dasar yang dibutuhkan tahap berikutnya.
- **Tahap 2 (Session, RBAC, ganti password sendiri)**: `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`, middleware RBAC generik, `PATCH /auth/me/password`.
- **Tahap 3 (Invite, forgot/reset password, Resend)**: investigasi & konfirmasi library Resend dulu (lihat requirement di atas) → `POST /admin/users`, `POST /auth/forgot-password`, `POST /auth/reset-password`.
- **Tahap 4 (Seed admin bootstrap)**: migration seed + startup check idempotent (lihat edge case). Sengaja setelah Tahap 3 — butuh mekanisme invite-token sudah jadi.
- **Tahap 5 (Integrasi & regresi menyeluruh)**: test end-to-end lintas tahap (lihat section Testing) + regresi penuh scaffolding yang sudah ada (`config`, `db/migration`, `api/handler`, `api/middleware`) — pastikan tidak ada yang kebreak.

## Skema/struktur data

USER {
int id PK
string nama
string email UNIQUE -- dipakai sebagai identifier login, wajib unique
string password_hash NULLABLE -- null sampai invite diselesaikan
}
USER_ROLE {
int user_id PK_FK
string role PK -- salah satu dari: 'petugas', 'dokter', 'admin' (persis anotasi role di api-contract.md)
}
SESSIONS {
string id_hash PK -- SHA256(token mentah)
int user_id FK
timestamp created_at
timestamp expires_at -- sliding
timestamp absolute_expires_at -- hard cap 24 jam
}
PASSWORD_TOKENS {
string token_hash PK -- SHA256(token mentah)
int user_id FK
string type -- 'invite' atau 'reset'
timestamp expires_at -- invite 7 hari, reset 1 jam
timestamp consumed_at NULLABLE -- diisi atomic saat dipakai
timestamp created_at
}

## Edge case yang perlu dihandle

- **Login dengan `password_hash` masih null** — tolak bersih (401/400 sesuai konvensi error), JANGAN coba hash-compare ke nilai kosong/null.
- **Konsumsi token invite/reset** — atomic: `UPDATE password_tokens SET consumed_at = now() WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > now() RETURNING user_id`, cek rows affected. Kalau kosong → token sudah dipakai/expired/invalid, tolak request.
- **`forgot-password` untuk email tidak terdaftar** — tetap 200 generik, jangan bocorkan lewat response maupun timing yang signifikan berbeda (cegah user enumeration).
- **Resend gagal terkirim saat `POST /admin/users`** — pembuatan user TIDAK BOLEH gagal karenanya. Log kegagalan, user tetap ter-create dengan `password_hash` null (bisa di-resend manual nanti lewat item #8, di luar scope sekarang).
- **Seed admin bootstrap — idempotent, KHUSUS (jangan implementasi dari asumsi sendiri, ini sudah diverifikasi lewat diskusi user, bukan requirement mentah)**:
  - Kenapa bukan "reprint token yang sama": token cuma pernah disimpan dalam bentuk hash (SHA256, one-way) — TIDAK ADA cara mengambil kembali nilai token mentah dari hash yang tersimpan untuk di-print ulang persis. Kalau ada yang mencoba "fix" ini dengan menyimpan token mentah demi bisa reprint, itu pelanggaran langsung terhadap prinsip hash-only storage yang berlaku ke SEMUA token di project ini (sessions, display_token, password_tokens) — JANGAN lakukan itu.
  - Behavior yang benar, tiap kali server start:
    1. Cek `password_hash` admin (dari `SEED_ADMIN_EMAIL`) — kalau **sudah terisi** (admin sudah selesai setup) → skip total, tidak ada log/token apapun.
    2. Kalau **masih null** → query `password_tokens`: ada baris `type='invite'`, `consumed_at IS NULL`, `expires_at > now()` untuk admin ini?
       - **Ada** → JANGAN generate token baru, JANGAN invalidate yang lama. Boleh log pesan informatif (mis. "token invite admin masih valid, expired at <waktu> — cek log run sebelumnya"), tapi TIDAK reprint nilai token (memang tidak bisa, lihat alasan di atas).
       - **Tidak ada** (belum pernah dibuat, atau yang lama sudah expired/consumed) → generate token baru, simpan hash-nya, print nilai mentahnya ke log (satu-satunya momen token ini pernah terlihat sebagai plaintext).
  - Recovery kalau developer kehilangan token (lupa copy sebelum log lewat): TIDAK perlu mekanisme baru — tunggu sampai expired (auto-generate baru di restart berikutnya), atau hapus manual baris token itu di DB untuk memaksa regenerasi lebih cepat.
- **RBAC "any authenticated"** — beda dari role spesifik (`[admin]`, dst), middleware harus bisa terima semua role asal sesi valid, bukan cuma cek role tertentu.

## Testing

- Login: sukses (cookie ter-set sesuai spec), gagal (password salah / email tidak terdaftar / `password_hash` null).
- `GET /auth/me`: cookie valid, cookie invalid/expired, tanpa cookie (401).
- Logout: cookie di-clear, request berikutnya ke endpoint terproteksi jadi 401.
- RBAC middleware: role sesuai (lolos), role tidak sesuai (403), tanpa auth ke endpoint yang butuh auth (401).
- `PATCH /auth/me/password`: sukses, password lama salah (400).
- `POST /admin/users`: sukses (201, `inviteLink` ada, `password_hash` null di DB), Resend gagal tapi user tetap ter-create, non-admin ditolak (403).
- `POST /auth/forgot-password`: selalu 200 (email terdaftar maupun tidak), token ter-generate & ter-hash di DB untuk email terdaftar, token TIDAK PERNAH muncul di response manapun.
- `POST /auth/reset-password`: token valid → sukses + `consumed_at` terisi; token dipakai dua kali (test concurrency, dua request bersamaan konsumsi token yang sama, lawan Postgres asli via testcontainers-go sesuai konvensi test project — cuma 1 yang boleh berhasil); token expired; token invalid.
- Seed admin bootstrap: startup pertama (`password_hash` null, belum ada token) → token digenerate & di-log; restart berikutnya dengan token masih valid → TIDAK ada token baru digenerate (assert hash di DB tidak berubah); token expired lalu restart → token baru digenerate; admin sudah selesai setup → skip total, tidak ada log token.
- Regresi: seluruh test suite scaffolding yang sudah ada (`config`, `db/migration`, `api/handler`, `api/middleware`) tetap PASS.

## Kriteria selesai

Seluruh 5 tahap selesai, seluruh skenario test di atas PASS (termasuk lawan Postgres asli untuk yang concurrency-sensitive), `go vet` lolos, regresi scaffolding tidak ada yang kebreak, dan user sudah verifikasi manual (login, invite user baru, reset password, restart server untuk cek idempotent seed admin).
