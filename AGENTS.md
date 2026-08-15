# AGENTS.md — Modul RME & Antrian Klinik

Instruksi kerja untuk AI coding agent (Antigravity). Dibaca otomatis sebelum melakukan perubahan apapun di project ini.

## 1. Dokumen Acuan (source of truth — WAJIB dibaca sebelum implementasi)

- `docs/PRD.md` — latar belakang, scope, aktor, alur operasional, fitur MVP
- `docs/TDD.md` — arsitektur, concurrency & data integrity pattern, auth/session, realtime, testing, deployment (termasuk ERD)
- `docs/api-contract.md` — kontrak endpoint lengkap (request/response, role per endpoint, error format)

Dokumen acuan di atas diubah oleh user (bukan agent sepihak). Agent dilarang berimprovisasi menambah/mengubah requirement sendiri tanpa konfirmasi user. Namun, jika dalam implementasi/planning ditemukan requirement/kontrak lama yang perlu direvisi demi _best practice_, agent sah mengusulkan revisi tersebut ke user (termasuk untuk fitur berstatus "Selesai", lihat §4). Exception: `docs/DESIGN.md` (living document untuk Component Registry & design tokens) wajib diperbarui agent sendiri saat ada komponen baru.

## 2. Tech Stack

| Layer             | Teknologi                                                                  |
| ----------------- | -------------------------------------------------------------------------- |
| Backend           | Go + Gin                                                                   |
| DB access         | sqlc (generate dari raw SQL) + pgx/v5 sebagai driver                       |
| Database          | PostgreSQL, migration via golang-migrate (library, auto-run saat start)    |
| Frontend          | Angular (standalone components, Vitest), Tailwind v4                       |
| Realtime          | WebSocket, in-memory hub per proses (single-instance)                      |
| Email             | Resend (invite user, forgot-password)                                      |
| Observability     | Sentry (Error tracking, panic recovery & telemetry BE + FE)                |
| Code Intelligence | GitNexus (Symbol graph & impact analysis MCP)                              |
| Deployment        | Docker multi-stage, Nginx reverse proxy, docker-compose, GitHub Actions CI |

## 3. Prinsip Kerja — ATURAN PALING PENTING DI FILE INI

1. **Selalu rencana dulu, baru eksekusi.** Sebelum menulis kode untuk task apapun, tulis dulu rencana implementasi berupa task list langkah-langkah kecil. Tunggu persetujuan eksplisit dari user sebelum mulai coding.
2. **Satu langkah kecil per iterasi — bukan satu fitur, apalagi semua fitur.** Definisi "langkah kecil":
   - Backend: 1 endpoint + fungsi service pendukungnya (bukan 1 domain penuh, bukan seluruh CRUD sekaligus)
   - Frontend: 1 komponen atau 1 halaman (bukan 1 alur penuh dari awal sampai akhir)
   - Schema: 1 migration per perubahan
3. **Berhenti setelah 1 langkah selesai.** Laporkan pakai format berikut (wajib, bukan opsional — supaya laporan bisa diverifikasi langsung tanpa perlu dicek manual satu-satu):

   ## Laporan: <nama task/tahap>

   ### 1. Checklist scope (mirror nomor requirement di prompt/spec)
   - [x] <requirement> — <1 baris implementasi>
   - [ ] <requirement> — SKIP, alasan: <kenapa>

   ### 2. File berubah
   - `path/file` (baru/modify) — <1 baris isi, JANGAN paste kode mentah>

   ### 3. Verifikasi — command + output APA ADANYA (bukan parafrase/ringkasan)

   ```bash
   $ <command persis yang dijalankan>
   <berikan output penting, bukan output mentah>
   ```

   ### 4. Deviasi & konsistensi

   <Wajib diisi walau "tidak ada deviasi". Kalau task menyentuh konvensi yang rawan dilanggar diam-diam (styling hardcode di FE — lihat §8, format response/envelope di BE — lihat §7), sebutkan CARA mengeceknya (command yang dijalankan), bukan cuma klaim "sudah sesuai".>

   Task kecil (1 file, 1 endpoint) — Section 1 & 4 boleh dipadetin jadi 1-2 baris. Section 3 TIDAK BOLEH diskip dalam kondisi apa pun — itu satu-satunya bukti keras dibanding klaim naratif.

   Tunggu review/persetujuan user sebelum lanjut ke langkah berikutnya. JANGAN otomatis lanjut tanpa diminta, walaupun "kelihatan jelas" langkah berikutnya apa.

4. **Jangan mengasumsikan requirement yang tidak eksplisit ada di `docs/`.** Ambigu → tanya. Jangan menebak lalu diam-diam mengimplementasikan tebakan itu.
5. **Ikuti urutan dependency logis** — backend + data model + concurrency correctness dulu (bagian paling berisiko teknis di project ini), baru UI Angular yang menyertainya. Jangan bangun layar yang bergantung ke endpoint yang belum ada & belum teruji.

## 4. Kebebasan Implementasi

- **Scope vs Detail Teknis**: Scope/requirement wajib eksplisit dari `docs/` atau dikonfirmasi user. Untuk detail teknis (struktur kode, validasi tambahan, pendekatan efisien), agent bebas berimprovisasi asal ada benefit konkret (lebih aman, maintainable, sesuai konvensi Go/Angular). Catat improvisasi di bagian "Catatan" log `done.md` (§6).
- **Generated Code**: Kode hasil `sqlc` di `internal/db/generated/` wajib terpisah jelas dan DILARANG diedit manual.
- **Modifikasi & Rollback Fitur "Selesai"**: Status "Selesai" di `backlog.md` adalah checkpoint, bukan segel permanen. Jika menemukan kebutuhan modifikasi kode/skema/kontrak lama dengan benefit konkret yang jelas, usulkan ke user (TETAP wajib approval eksplisit sebelum dieksekusi). Rollback diperbolehkan jika modifikasi ternyata salah arah setelah dicoba. Modifikasi fitur selesai dicatat sebagai entry **Addendum** baru di `done_be.md`/`done_fe.md` tanpa menghapus entry asli (`backlog.md` tetap `[x]`).
- **Tooling Investigasi (Opsional)**: GitNexus tersedia via MCP (`.agents/mcp_config.json`) untuk investigasi struktural — trace execution flow, cek dependency/impact sebelum modifikasi kode, atau blast-radius check sebelum refactor. Dipakai kalau relevan — BUKAN wajib di setiap task kecil. Graph-nya snapshot statis dari `gitnexus analyze` terakhir. Setelah ada perubahan struktural signifikan (1 fitur besar selesai, refactor besar) — jalankan ulang `gitnexus analyze` dari root project sebelum mengandalkan hasilnya lagi.

## 5. Kebijakan Test & Retry

- Backend: `testing` + `testify`. Setiap endpoint/service yang selesai di 1 langkah wajib disertai test yang benar-benar assert behavior, bukan boilerplate kosong.
- **Test yang menyentuh concurrency** (atomic upsert counter, klaim `FOR UPDATE SKIP LOCKED`, lock `audit_log_tail`, partial unique index `addendum_of`) **wajib jalan lawan Postgres asli** via testcontainers-go — bukan mock, bukan SQLite. `go test -race` hanya mendeteksi race di memori proses Go, **tidak** mendeteksi race condition lintas-koneksi database — jangan andalkan itu sebagai bukti concurrency aman. Test concurrency wajib spawn goroutine konkuren yang benar-benar memanggil fungsi terkait secara bersamaan, lalu assert hasilnya (tidak ada nomor dobel, tidak ada klaim ganda).
- Frontend: Vitest (default Angular CLI). Unit test wajib untuk logic kritis (guard role, form validation, state derivation dari WS event) — tidak wajib coverage penuh komponen kosmetik.
- Lint + test wajib lolos tiap perubahan kode. Kalau ada test gagal, boleh coba perbaiki maksimal **2x percobaan**. Masih gagal setelah itu — STOP, laporkan ke user (test mana, pesan error, dugaan penyebab), jangan lanjut ke langkah berikutnya, jangan update `done.md`.
- Jalankan test scope penuh (`go test -v -p 1 ./...` tanpa `-short`) kalau perubahan menyentuh file yang dipakai lintas-domain (middleware auth/RBAC, error handler, util shared, sqlc generated) — atau sebelum 1 fitur besar resmi ditutup di `done.md`.

## 6. Update done.md

Setelah 1 langkah kecil selesai, test lolos, DAN user sudah approve — tambahkan entry log:

- Jika perubahan berada di folder `backend/`, catat entry ke `workflow/done_be.md`.
- Jika perubahan berada di folder `frontend/`, catat entry ke `workflow/done_fe.md`.

Informasi yang dicatat meliputi: apa yang dikerjakan, file yang berubah, cara verifikasi, dan "Catatan" untuk penyimpangan/improvisasi (§4). Format entry tetap konsisten seperti sebelumnya.

## 7. Konvensi Kode — Backend

**Response & error**

- Sukses: return resource langsung, **tanpa envelope**. Jangan bikin format bungkus baru.
- List endpoint (`page`/`limit`) wajib sertakan header response `X-Total-Count` (total row cocok filter) — jangan taruh total di body, itu melanggar aturan no-envelope di atas.
- Error: `{ "error": { "code", "message", "requestId" } }` — `message` dikurasi per `code` (aman ditampilkan, bukan raw error DB/Go — mencegah kebocoran data seperti NIK ikut ke-embed di pesan constraint violation). `requestId` (UUID per request) dicatat di server log bareng detail lengkap (stack trace, raw error) untuk debugging — jangan expose detail itu ke client.
- Path param `:id` (integer, bukan UUID di project ini) wajib divalidasi sebagai angka valid sebelum dipakai — jangan biarkan nembus ke query dan jadi 500 tak terkontrol.

**Penamaan**

- Path API & field JSON domain bisnis: Bahasa Indonesia, konsisten dengan `api-contract.md` persis (`/pasien`, `/kunjungan`, `nomorAntrian`, `dokterId`, `hasilPemeriksaan`).
- Istilah teknis generik: Bahasa Inggris (`requestId`, `token`).
- JSON casing: camelCase.

**Data integrity & concurrency**

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
- Role `admin` mutually exclusive dari role `dokter` — 1 user tidak boleh punya keduanya sekaligus (kontradiction langsung dengan alasan keamanan admin-dikecualikan-dari-rekam-medis di api-contract.md). Wajib divalidasi di `PATCH /admin/users/:id/roles`.
- Password: bcrypt, **cost factor 12** (bukan argon2id).
- **Admin tidak pernah menentukan password user.** `POST /admin/users` membuat user dengan `password_hash` **nullable**, trigger email invite via Resend (token hash di DB, TTL **7 hari**). `POST /auth/forgot-password` (publik, tanpa auth) — **selalu** return 200 generik terlepas email terdaftar atau tidak (cegah user enumeration), TTL token **1 jam**. **`forgot-password` TIDAK PERNAH mengembalikan token/link mentah di response** — beda dari endpoint admin-authenticated (`POST /admin/users`, resend-invite) yang boleh. Konsumsi token (invite maupun reset) wajib atomic: `UPDATE ... WHERE token_hash = ? AND consumed_at IS NULL`, cek rows affected. Login wajib reject bersih untuk `password_hash` masih null.

**Lain-lain**

- Timezone: **Asia/Jakarta** di-set global (`TZ` env di startup). Semua parsing/format tanggal-jam dari input client pakai timezone ini konsisten.
- Semua list endpoint (`GET /admin/audit-log`, `GET /admin/users`, `GET /pasien/search`, dst) wajib pagination (`page`/`limit`).
- `GET /pasien/search` menerima `nik` dan/atau `nama` (partial match).

## 8. Konvensi Kode — Frontend (Angular)

**State & reactivity**

- Standalone components, bukan `NgModule`.
- Signals + service layer tipis per domain — bukan NgRx.
- `@if`/`@for`/`@switch` (native control flow) & `ChangeDetectionStrategy.OnPush` default di semua komponen.
- **Pemisahan template HTML**: Gunakan file `.component.html` terpisah (`templateUrl: './<komponen>.component.html'`) — bukan inline string `template:` — demi pemisahan logic vs UI, keterbacaan, dan syntax highlighting IDE.
- **Reactive Forms wajib** (bukan template-driven) untuk form dengan array dinamis — `diagnosis[]`/`tindakan[]` di rekam medis butuh `FormArray`.

**Styling & UI Primitives**

- Tailwind v4.
- UI primitives: spartan/ui (spartan.ng) — headless (Angular CDK) + style Tailwind di-copy-paste lewat spartan CLI.
- **Larangan hardcode styling — DILARANG KERAS.** Semua warna, radius, spacing, font wajib pakai token semantik dari `docs/design.md`/`global.css` (CSS variable) — bukan nilai literal (`#fff`, `rgb(...)`, angka px bebas) ditulis langsung di komponen. Sebelum lapor task selesai, WAJIB jalankan self-check (mis. `grep -rn "#[0-9a-fA-F]\{3,6\}\|rgb"`) dan lampirkan hasilnya (harus nihil match) di Section 4 laporan (§3 poin 3).
- **Cek Component Registry dulu** sebelum bikin komponen baru ATAU fetch primitive baru lewat Spartan CLI/MCP. Cek `docs/design.md` (Component Registry) dan folder `shared/`. Kalau sudah ada, reuse/extend — jangan fetch ulang copy mentah. Saat fetch primitive Spartan baru (pertama kali): setelah ter-copy, verifikasi dia memakai token semantik dari `docs/design.md` (CSS variable). Komponen baru (composed maupun primitive yang sudah diverifikasi) — tambahkan ke Component Registry di `docs/design.md` saat itu juga.

**Struktur folder**

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
- **Resolusi auth state sebelum guard dievaluasi — WAJIB.** Cookie session `httpOnly` berarti Angular tidak tahu status login sampai `GET /auth/me` selesai. Fix: pakai `provideAppInitializer`/resolver yang di-await, **di-scope ke root route staff saja** (bukan `APP_INITIALIZER` global) — supaya route `papan-antrian` (publik) tidak tertahan.
- **Route guard per role** — `CanActivateFn` baca role dari Signals, redirect kalau tidak sesuai. Route group `papan-antrian` **tidak** pakai guard staff sama sekali.
- Selector prefix komponen: `app-`. Environment config: `environment.ts`/`environment.prod.ts`.

**Realtime**

- 1 service (`RealtimeService`) wrap koneksi WS, handle reconnect + backoff, expose event lewat Signal.
- **Dev-server proxy WebSocket** — `proxy.conf.json`, tambahkan `"ws": true` eksplisit di entry proxy-nya.

**Testing**

- Test file co-located (`*.spec.ts` di sebelah komponen), default Angular CLI + Vitest.

## 9. Larangan

- JANGAN buat requirement/kode yang tidak eksplisit ada di `docs/` tanpa konfirmasi user (§1 & §3.4).
- JANGAN ubah dokumen acuan (§1), migration/schema DB, atau spec `workflow/<nama_fitur>.md` yang sudah disepakati tanpa approval eksplisit user (kecuali `docs/design.md` per §8).
- JANGAN menambah dependency/library baru tanpa alasan & konfirmasi eksplisit.
- JANGAN buat format response/envelope baru di luar spesifikasi §7.

## 10. Kebijakan Verifikasi Visual

- Review dilakukan user bersama Antigravity secara langsung di lingkungan lokal — **bukan** dikirim ke sesi Claude chat terpisah.
- Review visual manual HANYA diminta 1x per fitur (di akhir, setelah semua tahap fitur itu selesai) — bukan di setiap tahap kecil. Testing otomatis tetap wajib tiap tahap sesuai §5.
- Pengecualian: review per-tahap boleh diminta kalau tahap itu menetapkan referensi visual baru yang akan dikunci, atau ada perubahan visual signifikan yang perlu dikonfirmasi sebelum tahap berikutnya melanjutkan pola yang sama.
- Di luar 2 kondisi itu, laporan tahap cukup mengikuti format wajib §3 poin 3.

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **klinik-rme** (2135 symbols, 5904 relationships, 121 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> Index stale? Run `node .gitnexus/run.cjs analyze` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? `npx gitnexus analyze` (npm 11 crash → `npm i -g gitnexus`; #1939).

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows. For regression review, compare against the default branch: `detect_changes({scope: "compare", base_ref: "main"})`.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `query({search_query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `context({name: "symbolName"})`.
- For security review, `explain({target: "fileOrSymbol"})` lists taint findings (source→sink flows; needs `analyze --pdg`).

## Never Do

- NEVER edit a function, class, or method without first running `impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit changes without running `detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/klinik-rme/context` | Codebase overview, check index freshness |
| `gitnexus://repo/klinik-rme/clusters` | All functional areas |
| `gitnexus://repo/klinik-rme/processes` | All execution flows |
| `gitnexus://repo/klinik-rme/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
