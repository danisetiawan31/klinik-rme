# AGENTS.md — Modul RME & Antrian Klinik

Instruksi kerja untuk AI coding agent (Antigravity). Dibaca otomatis sebelum melakukan perubahan apapun di project ini.

## 1. Dokumen Acuan (source of truth — WAJIB dibaca sebelum implementasi)

- `docs/PRD.md` — latar belakang, scope, aktor, alur operasional, fitur MVP
- `docs/TDD.md` — arsitektur, concurrency & data integrity pattern, auth/session, realtime, testing, deployment (termasuk ERD sebagai section di dalamnya)
- `docs/api-contract.md` — kontrak endpoint lengkap (request/response, role per endpoint, error format)

**Dokumen di atas TIDAK BOLEH diubah oleh agent.** Itu hasil proses discovery yang sudah difinalisasi terpisah. Kalau implementasi butuh sesuatu yang tidak ada di dokumen ini — STOP, tanya ke user. Jangan berimprovisasi menambah atau mengubah requirement sendiri.

## 2. Tech Stack

| Layer      | Teknologi                                                                  |
| ---------- | -------------------------------------------------------------------------- |
| Backend    | Go + Gin                                                                   |
| DB access  | sqlc (generate dari raw SQL) + pgx/v5 sebagai driver                       |
| Database   | PostgreSQL, migration via golang-migrate (library, auto-run saat start)    |
| Frontend   | Angular (standalone components, Vitest), Tailwind v4                       |
| Realtime   | WebSocket, in-memory hub per proses (single-instance)                      |
| Email      | Resend (invite user, forgot-password)                                      |
| Deployment | Docker multi-stage, Nginx reverse proxy, docker-compose, GitHub Actions CI |

## 3. Prinsip Kerja — ATURAN PALING PENTING DI FILE INI

1. **Selalu rencana dulu, baru eksekusi.** Sebelum menulis kode untuk task apapun, tulis dulu rencana implementasi berupa task list langkah-langkah kecil. Tunggu persetujuan eksplisit dari user sebelum mulai coding.
2. **Satu langkah kecil per iterasi — bukan satu fitur, apalagi semua fitur.** Definisi "langkah kecil":
   - Backend: 1 endpoint + fungsi service pendukungnya (bukan 1 domain penuh, bukan seluruh CRUD sekaligus)
   - Frontend: 1 komponen atau 1 halaman (bukan 1 alur penuh dari awal sampai akhir)
   - Schema: 1 migration per perubahan
3. **Berhenti setelah 1 langkah selesai.** Laporkan: apa yang dikerjakan, file apa yang berubah, cara mengetesnya. Tunggu review/persetujuan user sebelum lanjut ke langkah berikutnya. JANGAN otomatis lanjut tanpa diminta, walaupun "kelihatan jelas" langkah berikutnya apa.
4. **Jangan mengasumsikan requirement yang tidak eksplisit ada di `docs/`.** Ambigu → tanya. Jangan menebak lalu diam-diam mengimplementasikan tebakan itu.
5. **Ikuti urutan dependency logis** — backend + data model + concurrency correctness dulu (bagian paling berisiko teknis di project ini), baru UI Angular yang menyertainya. Jangan bangun layar yang bergantung ke endpoint yang belum ada & belum teruji.

## 4. Kebebasan Implementasi

- Poin #4 di atas berlaku untuk **requirement/scope** — itu tidak bisa ditebak, harus eksplisit ada di `docs/` atau dikonfirmasi user.
- Untuk **detail implementasi teknis** (struktur kode, validasi tambahan, pendekatan yang lebih efisien) — boleh berimprovisasi **asal ada benefit konkret** (lebih aman, lebih maintainable, lebih sesuai konvensi Go/Angular).
- Setiap improvisasi/penyimpangan dari rencana awal **wajib dicatat** di entry `done.md` saat langkah itu selesai — bagian "Catatan".
- Kalau penyimpangan itu prinsipnya relevan untuk fitur lain juga (bukan cuma spesifik langkah ini), tambahkan sebagai baris baru di §7/§9 file ini.
- Kode hasil generate sqlc wajib terpisah jelas dari kode manual (service/handler) — folder khusus (mis. internal/db/generated/ atau default sqlc sesuai config), jangan pernah diedit manual.

## 5. Kebijakan Test & Retry

- Backend: `testing` + `testify`. Setiap endpoint/service yang selesai di 1 langkah wajib disertai test yang benar-benar assert behavior, bukan boilerplate kosong.
- **Test yang menyentuh concurrency** (atomic upsert counter, klaim `FOR UPDATE SKIP LOCKED`, lock `audit_log_tail`, partial unique index `addendum_of`) **wajib jalan lawan Postgres asli** via testcontainers-go — bukan mock, bukan SQLite. `go test -race` hanya mendeteksi race di memori proses Go, **tidak** mendeteksi race condition lintas-koneksi database — jangan andalkan itu sebagai bukti concurrency aman. Test concurrency wajib spawn goroutine konkuren yang benar-benar memanggil fungsi terkait secara bersamaan, lalu assert hasilnya (tidak ada nomor dobel, tidak ada klaim ganda).
- Frontend: Vitest (default Angular CLI). Unit test wajib untuk logic kritis (guard role, form validation, state derivation dari WS event) — tidak wajib coverage penuh komponen kosmetik.
- Lint + test wajib lolos tiap perubahan kode. Kalau ada test gagal, boleh coba perbaiki maksimal **2x percobaan**. Masih gagal setelah itu — STOP, laporkan ke user (test mana, pesan error, dugaan penyebab), jangan lanjut ke langkah berikutnya, jangan update `done.md`.
- Jalankan test scope penuh (bukan cuma domain yang disentuh) kalau perubahan menyentuh file yang dipakai lintas-domain (middleware auth/RBAC, error handler, util shared) — atau sebelum 1 fitur besar resmi ditutup di `done.md`.

## 6. Update done.md

Setelah 1 langkah kecil selesai, test lolos, DAN user sudah approve — tambahkan entry log:
- Jika perubahan berada di folder `backend/`, catat entry ke `workflow/done_be.md`.
- Jika perubahan berada di folder `frontend/`, catat entry ke `workflow/done_fe.md`.

Informasi yang dicatat meliputi: apa yang dikerjakan, file yang berubah, cara verifikasi, dan "Catatan" untuk penyimpangan/improvisasi (§4). Format entry tetap konsisten seperti sebelumnya.

## 7. Konvensi Kode — Backend

**Response & error**

- Sukses: return resource langsung, **tanpa envelope**. Jangan bikin format bungkus baru.
- Error: `{ "error": { "code", "message", "requestId" } }` — `message` dikurasi per `code` (aman ditampilkan, bukan raw error DB/Go — mencegah kebocoran data seperti NIK ikut ke-embed di pesan constraint violation). `requestId` (UUID per request) dicatat di server log bareng detail lengkap (stack trace, raw error) untuk debugging — jangan expose detail itu ke client.
- Path param `:id` (integer, bukan UUID di project ini) wajib divalidasi sebagai angka valid sebelum dipakai — jangan biarkan nembus ke query dan jadi 500 tak terkontrol.

**Penamaan**

- Path API & field JSON domain bisnis: Bahasa Indonesia, konsisten dengan `api-contract.md` persis (`/pasien`, `/kunjungan`, `nomorAntrian`, `dokterId`, `hasilPemeriksaan`).
- Istilah teknis generik: Bahasa Inggris (`requestId`, `token`).
- JSON casing: camelCase.

**Data integrity & concurrency** — ringkasan dari `TDD.md`, detail lengkap tetap rujuk ke sana:

- Nomor antrian: atomic upsert `INSERT ... ON CONFLICT DO UPDATE ... RETURNING`, **jangan** `SELECT COUNT()`.
- Klaim pasien: atomic `UPDATE ... WHERE id = (SELECT ... FOR UPDATE SKIP LOCKED) RETURNING`, `dokterId` wajib dari session (bukan body/params).
- No-show: `skip_count` (bukan status terpisah); `tidak_hadir` final, hanya diset manual di penutupan hari.
- Audit trail: singleton `audit_log_tail` (`id`=1), lock `FOR UPDATE` (**bukan** `SKIP LOCKED` — chain butuh urutan sekuensial ketat). Wajib satu transaksi eksplisit: [perubahan data bisnis] + [lock tail] + [insert audit_log] + [update tail]. **Audit log berlaku untuk `rekam_medis` (create & addendum) dan edit biodata `pasien` — TIDAK berlaku untuk `queue_counter`/`queue_entries`** (operasi antrian sengaja dikecualikan supaya `SKIP LOCKED` tidak ikut ter-serialize).
- Versi terkini rekam medis: traverse-based leaf query (`NOT EXISTS` + `deleted_at IS NULL`), dijaga DB via `uq_addendum_of_active` (partial unique index `WHERE deleted_at IS NULL`) — jangan andalkan flag `is_latest` yang bisa desync.
- Optimistic locking: kolom `version` di `PASIEN` saja (rekam medis immutable, koreksi via addendum, bukan lewat locking).
- Soft-delete saja untuk `pasien`/`rekam_medis` (retensi hukum) — jangan hard-delete.
- **Reactive exception handling:** tangkap `*pgconn.PgError` setelah write gagal, cek `.Code` (SQLSTATE — `23505` unique_violation, `23503` fk_violation, `23514` check_violation). **Jangan** `SELECT` preemptive sebelum write untuk validasi yang sudah dijamin constraint DB — itu celah race condition, exact filosofi yang sama dengan semua pattern locking di atas.

**Auth & RBAC**

- Semua token (session, display token papan antrian, invite/reset password) disimpan **hash** (SHA256) di DB — token mentah cuma pernah ada di cookie/link client, tidak pernah disimpan plaintext.
- Token generation wajib `crypto/rand` (bukan `math/rand`), minimal 128-bit entropy, base64url.
- Cookie session: `httpOnly`, `Secure`, `SameSite=Strict`. FE dan BE **wajib** serve dari origin yang sama (Nginx `/api/*` → Go) — berlaku juga di local dev (Angular dev-server proxy config), bukan cuma production.
- Sliding expiration + `absolute_expires_at` (hard cap 24 jam).
- Papan antrian: token terpisah (`display_token_hash` di `KLINIK`), bukan session staff. Endpoint yang dipakai papan antrian wajib terima **dua jalur eksplisit**: cookie staff **atau** header `X-Display-Token` — bukan staff-only, bukan open.
- WebSocket: browser API tidak bisa custom header — papan antrian kirim token via **query param** (`?displayToken=`), beda dari REST yang pakai header. Ini keterbatasan browser, tulis eksplisit di kode biar tidak ada yang "perbaiki" jadi header dan diam-diam gagal.
- RBAC: middleware cek role terhadap tabel role-per-endpoint di `api-contract.md` — **wajib dipasang eksplisit di endpoint**, jangan diasumsikan aman karena UI menyembunyikan tombolnya.
- Endpoint Rekam Medis: `[dokter]` saja — admin **sengaja dikecualikan** dari akses langsung (investigasi lewat `GET /admin/audit-log/:id`, bukan endpoint klinis). Ini mengurangi exposure insidental, bukan mencegah akses yang benar-benar disengaja — jangan "simplify" jadi admin akses semua.
- Password: bcrypt, **cost factor 12** (bukan argon2id — ekosistem Go untuk argon2id butuh handle manual salt/parameter, bcrypt sudah matang lewat `x/crypto/bcrypt`). Known limitation: bcrypt memotong input di 72 byte — bukan masalah untuk password normal, tapi jangan asumsikan input lebih panjang tervalidasi penuh.
- **Admin tidak pernah menentukan password user.** `POST /admin/users` membuat user dengan `password_hash` **nullable**, trigger email invite via Resend (token hash di DB, TTL **7 hari**). `POST /auth/forgot-password` (publik, tanpa auth) — **selalu** return 200 generik terlepas email terdaftar atau tidak (cegah user enumeration), TTL token **1 jam**. **`forgot-password` TIDAK PERNAH mengembalikan token/link mentah di response** — beda dari endpoint admin-authenticated (`POST /admin/users`, resend-invite) yang boleh, karena caller-nya sudah admin ter-otentikasi. Konsumsi token (invite maupun reset) wajib atomic: `UPDATE ... WHERE token_hash = ? AND consumed_at IS NULL`, cek rows affected. Login wajib reject bersih untuk `password_hash` masih null — jangan coba hash-compare ke nilai kosong.

**Lain-lain**

- Timezone: **Asia/Jakarta** di-set global (`TZ` env di startup). Semua parsing/format tanggal-jam dari input client pakai timezone ini konsisten — jangan campur UTC di satu tempat dan lokal di tempat lain.
- Semua list endpoint (`GET /admin/audit-log`, `GET /admin/users`, `GET /pasien/search`, dst) wajib pagination (`page`/`limit`) — jangan return array tak terbatas.
- `GET /pasien/search` menerima `nik` dan/atau `nama` (partial match) — jangan cuma `nik`, karena pasien fallback-ID (tanpa NIK) butuh jalur pencarian sendiri.

## 8. Konvensi Kode — Frontend (Angular)

**State & reactivity**

- Standalone components, bukan `NgModule` — standar Angular saat ini, dan tidak ada kebutuhan lazy-loaded module besar di scope project ini.
- Signals + service layer tipis per domain — bukan NgRx. Arah resmi Angular sendiri terus memperkuat Signals sebagai primitive utama (signal-based input, `model()`, dst); NgRx proporsional untuk state kompleks lintas-fitur skala besar, bukan untuk project ini.
- `@if`/`@for`/`@switch` (native control flow) — bukan `*ngIf`/`*ngFor`. Structural directive lama sudah posisi legacy di Angular saat ini, CLI generate `@if` by default.
- `ChangeDetectionStrategy.OnPush` default di semua komponen — pasangan alami Signals, memaksimalkan fine-grained reactivity yang jadi alasan Signals dipilih. `Default` (check tiap siklus) tidak nyambung sama pilihan state management di atas.
- **Reactive Forms wajib** (bukan template-driven) untuk form dengan array dinamis — `diagnosis[]`/`tindakan[]` di rekam medis butuh `FormArray`. Ini konsekuensi bentuk data di `api-contract.md`, bukan pilihan gaya.

**Struktur folder** — konvensi berbasis fitur, direkomendasikan kuat untuk project single-app seperti ini (bukan satu-satunya struktur valid, tapi paling proporsional & paling mudah di-onboarding Antigravity):

```
src/app/
  core/
    auth/          — auth service, HTTP interceptor, route resolver (auth-resolve)
    realtime/      — RealtimeService (wrap WS, reconnect+backoff)
    guards/        — role guard
    types/         — tipe generik lintas-fitur (ErrorEnvelope, PaginatedResponse)
  features/
    pasien/
      pasien.routes.ts
      pasien.service.ts
      pasien.types.ts       — tipe spesifik domain ini, persis shape api-contract
      components/
    antrian/
    rekam-medis/
    admin/
    papan-antrian/   — publik, display token, tanpa guard staff
  shared/            — komponen dipakai >1 fitur
```

**Auth & routing**

- HTTP interceptor terpusat: attach `withCredentials: true` (perlu untuk cookie cross-origin di local dev), handle `401` (redirect ke login), parse `error.code`/`error.message` sesuai format `api-contract.md`.
- **Resolusi auth state sebelum guard dievaluasi — WAJIB.** Cookie session `httpOnly` (sengaja, cegah XSS baca token) berarti Angular tidak tahu status login sampai `GET /auth/me` selesai — kalau guard dievaluasi sebelum itu, user yang sebenarnya sudah login bisa ke-redirect keliru ke `/login` (race condition, gampang lolos di testing lokal karena koneksinya cepat). Fix: pakai `provideAppInitializer`/resolver yang di-await, **di-scope ke root route staff saja** (bukan `APP_INITIALIZER` global) — supaya route `papan-antrian` (publik, tidak butuh sesi staff sama sekali) tidak ikut tertahan.
- **Route guard per role** — `CanActivateFn` baca role dari Signals (sudah resolve lewat poin di atas), redirect kalau tidak sesuai. Route group `papan-antrian` **tidak** pakai guard staff sama sekali.
- Selector prefix komponen: `app-` (default Angular CLI).
- Environment config: `environment.ts`/`environment.prod.ts` standar CLI untuk base URL API.

**Realtime**

- 1 service (`RealtimeService`) wrap koneksi WS, handle reconnect + backoff, expose event lewat Signal — komponen jangan pegang raw `WebSocket` object langsung.
- **Dev-server proxy WebSocket** — `proxy.conf.json` (jaga origin sama, `/api/*` → backend, sesuai TDD) itu default cuma proxy HTTP biasa. Buat koneksi `/ws`, tambahkan `"ws": true` eksplisit di entry proxy-nya — tanpa ini, REST tetap kelihatan jalan normal tapi WebSocket diam-diam gagal atau salah target.

**Styling & testing**

- Tailwind v4.
- UI primitives: spartan/ui (spartan.ng) — headless (Angular CDK) + style Tailwind di-copy-paste lewat spartan CLI, bukan npm package tertutup. Tambah komponen baru via CLI spartan sesuai kebutuhan tiap fitur, jangan reinvent dari nol.
- Styling mengacu ke `docs/design.md`.
- Test file co-located (`*.spec.ts` di sebelah komponen), default Angular CLI + Vitest.

## 9. Larangan

- JANGAN generate kode untuk requirement yang tidak eksplisit ada di `docs/PRD.md`/`TDD.md`/`api-contract.md`, kecuali diminta eksplisit oleh user.
- JANGAN menambah dependency/library baru tanpa menyebutkan alasan & minta konfirmasi dulu.
- JANGAN ubah migration/schema tanpa konfirmasi eksplisit — perubahan schema butuh migration baru dan berdampak ke API contract juga.
- JANGAN mengubah isi folder `docs/`.
- JANGAN bikin format response baru — sukses return resource mentah (tanpa envelope), error ikuti persis `{ error: { code, message, requestId } }` di `api-contract.md`.
- JANGAN ubah isi `workflow/<nama_fitur>.md` (spec) setelah disepakati — penyimpangan dari spec dicatat di `workflow/done.md` bagian Catatan (§4), bukan dengan mengedit spec aslinya supaya kelihatan konsisten dengan yang benar-benar dibangun.

## 10. Kebijakan Verifikasi Visual

- Review dilakukan user bersama Antigravity secara langsung di lingkungan lokal — **bukan** dikirim ke sesi Claude chat terpisah, karena screenshot/file visual tidak otomatis tersinkron lintas tool tanpa upload manual.
- Review visual manual HANYA diminta 1x per fitur (di akhir, setelah semua tahap fitur itu selesai) — bukan di setiap tahap kecil. Testing otomatis tetap wajib tiap tahap sesuai §5.
- Pengecualian: review per-tahap boleh diminta kalau tahap itu menetapkan referensi visual baru yang akan dikunci (mis. halaman pertama sebuah fitur besar, atau papan antrian sebagai tampilan publik pertama), atau ada perubahan visual signifikan yang perlu dikonfirmasi sebelum tahap berikutnya melanjutkan pola yang sama.
- Di luar 2 kondisi itu, laporan tahap cukup: file yang diubah, hasil test otomatis, cara verifikasi singkat.
