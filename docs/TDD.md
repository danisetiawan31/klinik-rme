# TDD — Modul RME & Antrian Klinik

## Tech Stack
- **Backend:** Golang
- **Database:** PostgreSQL saja — audit log dan data bisnis wajib atomic dalam transaksi yang sama; split DB (mis. MongoDB untuk audit) kehilangan garansi ini
- **Frontend:** Angular
- **Realtime:** WebSocket, in-memory hub (single-instance)
- **Migration:** golang-migrate sebagai library, auto-run saat binary start — aman untuk single-instance, tidak ada risiko dua proses migration bersamaan
- **Testing:** Go `testing` + `testify` + testcontainers (backend), Vitest (frontend, default Angular CLI, tidak perlu setup tambahan)
- **Deployment:** Docker multi-stage, Nginx reverse proxy, docker-compose, GitHub Actions CI
- **Email:** Resend — invite user baru & reset password

## ERD

```mermaid
erDiagram
    KLINIK ||--o{ KUNJUNGAN : ""
    KLINIK ||--o{ QUEUE_COUNTER : ""
    PASIEN ||--o{ KUNJUNGAN : ""
    USER ||--o{ USER_ROLE : ""
    USER ||--o{ KUNJUNGAN : "menangani"
    USER ||--o{ REKAM_MEDIS : "menulis"
    USER ||--o{ AUDIT_LOG : "melakukan"
    USER ||--o{ SESSIONS : ""
    USER ||--o{ PASSWORD_TOKENS : ""
    KUNJUNGAN ||--o{ REKAM_MEDIS : ""
    REKAM_MEDIS ||--o{ DIAGNOSIS : ""
    REKAM_MEDIS ||--o{ TINDAKAN : ""
    REKAM_MEDIS |o--o| REKAM_MEDIS : "addendum_of"

    KLINIK {
        int id PK
        string nama
        time jam_buka
        time jam_tutup
        string display_token_hash "SHA256(token), untuk papan antrian"
    }
    QUEUE_COUNTER {
        int klinik_id PK_FK
        date tanggal PK
        int last_number
    }
    SESSIONS {
        string id_hash PK "SHA256(token mentah), token asli cuma di cookie client"
        int user_id FK
        timestamp created_at
        timestamp expires_at "sliding"
        timestamp absolute_expires_at "hard cap 24 jam"
    }
    PASSWORD_TOKENS {
        string token_hash PK "SHA256(token mentah), pola sama seperti sessions"
        int user_id FK
        string type "invite atau reset"
        timestamp expires_at "invite 7 hari, reset 1 jam"
        timestamp consumed_at "nullable, diisi atomic saat dipakai"
        timestamp created_at
    }
    USER {
        int id PK
        string nama
        string email
        string password_hash "nullable sampai invite diselesaikan"
    }
    USER_ROLE {
        int user_id PK_FK
        string role PK
    }
    PASIEN {
        int id PK
        string nik "nullable, fallback id kalau kosong"
        string nama
        date tanggal_lahir
        string jenis_kelamin
        string alamat
        string no_telp
        timestamp consent_at
        int version "optimistic locking"
        timestamp deleted_at "soft delete"
    }
    KUNJUNGAN {
        int id PK
        int pasien_id FK
        int klinik_id FK
        int dokter_id FK "diisi atomic saat klaim, nullable sampai dipanggil"
        date tanggal_kunjungan
        int nomor_antrian
        bool is_priority
        string priority_reason "nullable"
        int skip_count
        string status "menunggu, dipanggil, selesai, tidak_hadir"
        timestamp dipanggil_at
        timestamp created_at
    }
    REKAM_MEDIS {
        int id PK
        int kunjungan_id FK
        int dokter_id FK
        text keluhan
        text hasil_pemeriksaan
        bool is_addendum
        int addendum_of FK "nullable, self-reference, unique partial index"
        text alasan_addendum "nullable"
        timestamp deleted_at "soft delete"
        timestamp created_at
    }
    DIAGNOSIS {
        int id PK
        int rekam_medis_id FK
        string kode_icd "nullable"
        text deskripsi
    }
    TINDAKAN {
        int id PK
        int rekam_medis_id FK
        string jenis "tindakan atau resep"
        text deskripsi
    }
    AUDIT_LOG {
        int id PK
        string tabel_target
        int record_id
        int actor_user_id FK
        string aksi "create, update"
        jsonb before_data "nullable"
        jsonb after_data
        string hash_entry
        string previous_hash "selalu terisi, dari audit_log_tail"
        timestamp created_at
    }
    AUDIT_LOG_TAIL {
        int id PK "selalu 1, CHECK (id = 1)"
        string last_hash "genesis = SHA256('klinik-rme-genesis')"
    }
```

## Concurrency & Data Integrity

### Nomor antrian
Atomic upsert, bukan `SELECT COUNT()` (rawan race condition):
```sql
INSERT INTO queue_counter (klinik_id, tanggal, last_number)
VALUES (?, ?, 1)
ON CONFLICT (klinik_id, tanggal)
DO UPDATE SET last_number = queue_counter.last_number + 1
RETURNING last_number
```

### Klaim pasien ("panggil berikutnya")
Atomic claim, prioritas-aware, `dokter_id` wajib diisi dari session (bukan input form — cegah spoofing):
```sql
UPDATE queue_entries
SET status = 'dipanggil', dipanggil_at = now(), dokter_id = $dokter_id
WHERE id = (
  SELECT id FROM queue_entries
  WHERE klinik_id = ? AND tanggal = ? AND status = 'menunggu'
  ORDER BY is_priority DESC, skip_count ASC, nomor_antrian ASC
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
RETURNING *;
```
`SKIP LOCKED` (bukan blocking) — dua dokter bisa klaim pasien berbeda secara paralel tanpa saling tunggu.

Status `queue_entries`: `menunggu → dipanggil → selesai`. No-show pakai `skip_count` (bukan status terpisah) — pasien di-skip balik ke `menunggu`, `skip_count` naik, ikut tie-breaker di query claim. `tidak_hadir` final, hanya diset manual di penutupan hari.

### Audit trail hash chaining
Singleton `audit_log_tail` (1 baris, `id` selalu 1), lock `FOR UPDATE` — **bukan** `SKIP LOCKED` seperti klaim antrian, karena chain butuh urutan sekuensial ketat, bukan bisa lompat:
```sql
SELECT last_hash FROM audit_log_tail WHERE id = 1 FOR UPDATE;
-- hitung hash_entry baru pakai last_hash ini
INSERT INTO audit_log (..., previous_hash, hash_entry) VALUES (..., last_hash, ?);
UPDATE audit_log_tail SET last_hash = ? WHERE id = 1;
```
Genesis: `audit_log_tail` diseed 1 baris awal dengan `last_hash = SHA256('klinik-rme-genesis')` — nilai tetap dan terdokumentasi, bukan sembarang, supaya verifikasi ulang chain dari awal tidak ambigu.

**Wajib satu transaksi eksplisit**: [perubahan data bisnis] + [lock tail] + [insert audit_log] + [update tail] wajib dalam satu `BEGIN...COMMIT`. Tidak boleh ada write bisnis yang commit sebelum audit entry-nya ikut commit — ini alasan awal PostgreSQL-only dipilih (bukan split DB), jadi tidak boleh dilonggarkan di level kode.

**Scope audit log**: berlaku untuk `rekam_medis` (create & addendum) dan edit biodata `pasien`. **Tidak berlaku** untuk `queue_counter` maupun perubahan status `queue_entries` (klaim, skip, no-show) — operasi antrian sengaja dikecualikan dari audit trail supaya paralelisme `SKIP LOCKED` di klaim antrian tidak ikut ter-serialize oleh lock `audit_log_tail` (`FOR UPDATE`). Atribusi klaim (`dokter_id`, `dipanggil_at`) sudah cukup terekam langsung di kolom `queue_entries`, tidak perlu hash-chain.

DB trigger tolak `UPDATE`/`DELETE` langsung di tabel `audit_log` (tamper-evident, bukan tamper-proof).

### Versi terkini rekam medis (addendum chain)
```sql
SELECT r.* FROM rekam_medis r
WHERE r.kunjungan_id = ?
AND r.deleted_at IS NULL
AND NOT EXISTS (
  SELECT 1 FROM rekam_medis r2
  WHERE r2.addendum_of = r.id AND r2.deleted_at IS NULL
)
ORDER BY r.created_at DESC LIMIT 1;
```
`addendum_of` unique-constrained via partial index:
```sql
CREATE UNIQUE INDEX uq_addendum_of_active
ON rekam_medis (addendum_of)
WHERE deleted_at IS NULL;
```
Percabangan chain dicegah DB di antara baris aktif, bukan app logic. Soft-delete pada addendum otomatis membuka kembali parent-nya sebagai target valid. Constraint yang sama sekaligus jadi safety net kalau dua request coba bikin addendum ke parent yang sama secara bersamaan.

Lain-lain: soft-delete saja untuk `pasien`/`rekam_medis` (retensi hukum), optimistic locking (`version`) di `PASIEN` untuk cegah race condition edit biodata bersamaan.

### Konsumsi token invite/reset
Atomic, cegah token dipakai dua kali secara bersamaan (severity rendah dibanding bug lain di atas, tapi pola sama — higiene desain, bukan blocking):
```sql
UPDATE password_tokens
SET consumed_at = now()
WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > now()
RETURNING user_id;
```
Cek `rows affected`/hasil `RETURNING` — kalau kosong, token sudah dipakai/expired/invalid, tolak request.

## Auth & Session
- Token sesi dari `crypto/rand` (bukan `math/rand`), minimal 128-bit entropy, encode base64url
- DB simpan `SHA256(token)` sebagai `id_hash`, bukan token mentah — konsisten dengan `password_hash`; SHA-256 cukup (bukan bcrypt) karena butuh lookup langsung by hash, bukan compare lambat
- Cookie: `httpOnly`, `Secure`, `SameSite=Strict`
- FE dan BE wajib serve dari origin yang sama (Nginx reverse proxy, `/api/*` → Go) — berlaku juga di local dev (Angular dev-server proxy config), bukan cuma production, karena `SameSite=Strict` diam-diam tidak mengirim cookie di request cross-origin
- Sliding expiration (`expires_at` diperpanjang tiap request aktif) + `absolute_expires_at` (hard cap 24 jam) sebagai lapis kedua
- RBAC: middleware cek `user_roles` terhadap role yang dibutuhkan endpoint

### Invite & reset password (Resend)
- Admin bikin user tanpa set password — sistem generate token invite, kirim email via Resend berisi link set-password pertama kali. `password_hash` nullable sampai diselesaikan; login wajib tolak bersih untuk akun dengan `password_hash` null, bukan mencoba hash-compare ke nilai kosong.
- Forgot-password: token jenis sama (`password_tokens`, `type='reset'`), pola hash sama seperti `sessions`/`display_token` — token mentah cuma pernah ada di link email, DB cuma simpan hash
- Durasi: token invite 7 hari, token reset 1 jam (window take-over akun aktif harus pendek, beda kebutuhan dari invite)
- `POST /auth/forgot-password` **wajib** selalu return response generik (200) terlepas email terdaftar atau tidak — mencegah user enumeration
- **Token tidak pernah dikembalikan lewat response API**, termasuk untuk `forgot-password` — kalau iya, siapa pun bisa dapat token reset akun orang lain cuma dengan tahu emailnya, tanpa perlu akses inbox sama sekali (lebih parah dari masalah yang mau diselesaikan). Untuk kebutuhan demo, token dicatat di server log (pola sama seperti `requestId` di error handling) — presenter `tail` log, bukan baca dari response API
- Pengecualian: `POST /admin/users` (invite user baru) boleh return link invite mentah ke admin selain dikirim email — caller-nya admin ter-otentikasi yang memang legitimate menginisiasi akun itu, beda konteks dari `forgot-password` yang publik/tanpa auth
- Email gagal terkirim (Resend down/timeout) tidak boleh gagalkan pembuatan user — log kegagalan, sediakan aksi admin "resend invite"

### Display token (papan antrian)
Papan antrian tidak pakai session staff — akan putus di `absolute_expires_at` tanpa ada yang re-login layar publik. Token terpisah, long-lived, per-klinik (`display_token_hash` di `KLINIK`, pola hash sama seperti session). Admin punya aksi "regenerate display token" (overwrite hash = revoke otomatis token lama).

`GET /klinik/{id}/antrian` (dan endpoint sejenis untuk papan antrian): terima cookie session staff **atau** header `X-Display-Token` yang cocok — bukan staff-only, bukan open.

## Realtime (WebSocket)
- Hub in-memory per proses Go, `map[klinik_id][]*Client`, register/unregister lewat channel (hindari data race di map)
- Tidak perlu Redis/pub-sub — single-instance, cukup
- Pesan: invalidation ping saja (`{"type":"queue_updated"}`), bukan full state — client wajib refetch via REST. Menghindari kelas bug ordering yang muncul kalau push full state langsung
- Auth koneksi WS staff: ikut cookie session otomatis (origin sama). Papan antrian: `display_token`
- Reconnect: client auto-reconnect + backoff, wajib refetch REST sekali di awal koneksi/reconnect — WS bukan pembawa data, cuma notifikasi; update yang lewat saat disconnect otomatis ter-cover lewat refetch

## Testing
- Backend: `testing` + `testify`; test concurrency (counter, klaim, audit chain, partial index) **wajib lawan Postgres asli** via testcontainers — bukan mock, bukan SQLite, karena correctness project ini bergantung ke behavior lock/index spesifik Postgres
- `go test -race` hanya mendeteksi race di memori proses Go, **tidak** mendeteksi race condition lintas-koneksi database. Test concurrency wajib spawn goroutine konkuren yang benar-benar memanggil fungsi klaim/counter bersamaan, lalu assert hasilnya (tidak ada nomor dobel, tidak ada klaim ganda)
- Frontend: Vitest
- Lint + test wajib lolos tiap perubahan kode (lihat AGENTS.md)

## Deployment
- Docker multi-stage: Go jadi binary kecil, Angular di-build jadi static files
- Nginx reverse proxy: `/api/*` → Go, sisanya → Angular static (konsekuensi langsung dari `SameSite=Strict`)
- Migration jalan otomatis saat binary start (golang-migrate sebagai library, bukan langkah CLI terpisah) — aman untuk single-instance
- `docker-compose`: postgres + backend + frontend, satu command untuk local dev/demo
- CI (GitHub Actions): lint + test tiap push — lapis kedua di atas kebijakan `AGENTS.md`
- Hosting: fleksibel (VPS kecil / Railway / Fly.io), belum blocking
- `GET /health` (tanpa auth, `SELECT 1` ke Postgres) — dipakai container/reverse-proxy untuk cek instance masih hidup
