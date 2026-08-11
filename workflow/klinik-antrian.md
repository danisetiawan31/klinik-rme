# Klinik & Antrian

## Konteks & tujuan

Manajemen data klinik (single-instance), generate nomor antrian atomic, klaim pasien oleh dokter dengan prioritas, skip/no-show handling. Bagian paling kritis dari sisi concurrency di seluruh project — `FOR UPDATE SKIP LOCKED` untuk true parallelism antar dokter yang klaim bersamaan.

## Requirement

### Migration (3 tabel dari ERD docs/TDD.md, TANPA kolom display_token_hash di klinik — itu ditambahkan via migration terpisah di item #6)

- `klinik`, `queue_counter`, `kunjungan`

### Koreksi penamaan dari docs/TDD.md

Section "Concurrency & Data Integrity" di docs/TDD.md pakai nama tabel `queue_entries` dan kolom `tanggal` di contoh SQL — itu TIDAK SESUAI dengan ERD final (`kunjungan`, `tanggal_kunjungan`). WAJIB pakai nama dari ERD (`kunjungan`, `tanggal_kunjungan`), bukan nama di contoh SQL — contoh SQL itu pseudo-code lama yang tidak sempat disinkronkan, logic-nya tetap valid, cuma penamaannya yang perlu diterjemahkan.

### Seed klinik tunggal (WAJIB, keputusan final — beda dari kalimat backlog.md yang tidak menyebut ini eksplisit)

Sistem ini single-klinik (docs/PRD.md: "1 klinik kecil, bukan RS multi-poli"). TIDAK ADA endpoint `POST /klinik` di kontrak API manapun — baris `klinik` WAJIB di-seed via kode Go startup (pola sama seperti `bootstrap.SeedAdmin`, item #2), BUKAN migration statis (nama/jam operasional adalah data konfigurasi deployment-specific, bukan skema).

Config baru (required, ikuti pola validasi ketat yang sudah ada): `KLINIK_NAMA` (string), `KLINIK_JAM_BUKA` (format "HH:MM"), `KLINIK_JAM_TUTUP` (format "HH:MM").

Behavior startup, WAJIB idempotent tapi LEBIH SEDERHANA dari seed admin (tidak ada konsep token):

1. Cek: apakah tabel `klinik` benar-benar kosong (`SELECT COUNT(*) FROM klinik` atau setara)?
2. KOSONG → insert 1 baris pakai env var di atas.
3. SUDAH ADA ISI (≥1 baris) → skip total, JANGAN overwrite apapun — beda dari alasan idempotent seed admin (bukan cuma "cegah spam log", tapi karena BELUM ADA endpoint edit klinik sama sekali di backlog manapun; overwrite otomatis bisa merusak data yang mungkin sudah diedit manual lewat DB console).

Kegagalan di proses ini WAJIB fatal (hentikan startup), sama seperti seed admin — sistem tidak berguna tanpa klinik.

### Endpoint baru: `POST /kunjungan/:id/lewati` [dokter] — TIDAK ada di docs/api-contract.md, WAJIB ditambahkan user sendiri ke dokumen itu, tapi implementasi WAJIB jalan sesuai shape ini:

POST /kunjungan/:id/lewati [dokter]
← 200 { id, status: "menunggu", skipCount } | 409 kalau status kunjungan saat ini BUKAN "dipanggil"

Atomic: `UPDATE kunjungan SET status='menunggu', skip_count=skip_count+1 WHERE id=? AND status='dipanggil' RETURNING *`. Kalau 0 rows: disambiguasi SETELAH gagal (bukan preemptive select) — follow-up `GetKunjunganByID`: tidak ditemukan → 404; ditemukan tapi status bukan 'dipanggil' → 409 (code `INVALID_KUNJUNGAN_STATUS`).

### Endpoint (shape sesuai docs/api-contract.md persis, kecuali yang baru di atas)

- `GET /klinik/:id` [petugas, dokter, admin]
- `POST /kunjungan` [petugas, admin]
- `GET /kunjungan/:id` [petugas, dokter, admin]
- `GET /klinik/:id/antrian` [petugas, dokter, admin] — HANYA varian cookie-staff untuk fitur ini (`{ id, nomorAntrian, status, isPriority, pasienNama }`). Varian `X-Display-Token` BELUM diimplementasikan (butuh `display_token_hash`, itu item #6) — kalau request datang lewat header itu tanpa cookie valid, WAJIB 401 untuk sekarang, JANGAN coba parsial-implementasi.
- `POST /klinik/:id/panggil-berikutnya` [dokter]
- `POST /kunjungan/:id/lewati` [dokter] (baru, lihat di atas)
- `POST /kunjungan/:id/tidak-hadir` [dokter, admin]

### `POST /kunjungan` — detail

- `klinikId` TIDAK ada di request body (sesuai kontrak) — karena single-klinik, ambil baris `klinik` yang (seharusnya selalu) satu-satunya ada (`SELECT ... FROM klinik LIMIT 1` atau setara). Kalau tidak ada sama sekali (harusnya mustahil kalau seed startup berhasil) → 500 dengan log jelas, bukan crash tak terkontrol.
- Cek jam operasional: `now() (Asia/Jakarta) > klinik.jam_tutup` → 400 `KLINIK_TUTUP`. HANYA cek batas atas (jam tutup) — TIDAK ada validasi terhadap jam buka (PRD cuma bilang "terkunci setelah jam tutup", tidak ada requirement soal sebelum jam buka).
- `pasienId` tidak ditemukan (atau soft-deleted) → 404 `PASIEN_NOT_FOUND` (reuse error code dari item #4, jangan bikin baru).
- Sukses: atomic upsert `queue_counter` (klinik_id, tanggal=hari ini) → dapat `nomorAntrian` → insert `kunjungan` (status='menunggu', tanggal_kunjungan=hari ini, is_priority dari body default false, priority_reason dari body opsional, dokter_id NULL, skip_count=0).
- TIDAK ADA integrasi audit trail — sesuai docs/TDD.md, operasi antrian sengaja dikecualikan dari audit (supaya `SKIP LOCKED` tetap paralel, tidak ke-serialize lewat lock `audit_log_tail`).

### `GET /klinik/:id/antrian` — detail

Return SEMUA `kunjungan` hari ini di klinik itu, APAPUN statusnya (menunggu/dipanggil/tidak_hadir — 'selesai' belum bisa terjadi sampai item #7), `ORDER BY nomor_antrian ASC`. Keputusan ini diambil langsung (tidak eksplisit di dokumen manapun) — staff butuh gambaran penuh, bukan cuma yang masih menunggu.

### `POST /klinik/:id/panggil-berikutnya` — detail (PALING KRITIS, ikuti PERSIS pattern di docs/TDD.md, sudah diterjemahkan ke nama tabel benar)

```sql
UPDATE kunjungan
SET status = 'dipanggil', dipanggil_at = now(), dokter_id = $dokter_id
WHERE id = (
  SELECT id FROM kunjungan
  WHERE klinik_id = ? AND tanggal_kunjungan = ? AND status = 'menunggu'
  ORDER BY is_priority DESC, skip_count ASC, nomor_antrian ASC
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
RETURNING *;
```

`dokterId` WAJIB dari context sesi (session, BUKAN body/params — cegah spoofing, pola sudah ditetapkan project ini). Tidak ada row ter-update (antrian kosong untuk klinik+hari ini) → 204, BUKAN error.

### `POST /kunjungan/:id/tidak-hadir` — detail

Valid FROM status: `'menunggu'` ATAU `'dipanggil'` (dua-duanya state "belum resolved", operator bisa tutup hari dari kondisi manapun) → `'tidak_hadir'`. Status sudah `'selesai'` atau sudah `'tidak_hadir'` → 409 `INVALID_KUNJUNGAN_STATUS` (reuse code yang sama dengan `/lewati`). Atomic: `UPDATE kunjungan SET status='tidak_hadir' WHERE id=? AND status IN ('menunggu','dipanggil') RETURNING *`, disambiguasi 404 vs 409 dengan pola SAMA seperti `/lewati` (follow-up GetKunjunganByID setelah gagal, BUKAN preemptive select).

## Tahapan implementasi

- **Tahap 1 (Migration, seed klinik, query dasar)**: 3 migration + seed idempotent klinik + query sqlc dasar (atomic upsert counter, atomic claim SKIP LOCKED, CRUD dasar kunjungan/klinik).
- **Tahap 2 (Endpoint non-klaim)**: `GET /klinik/:id`, `POST /kunjungan`, `GET /kunjungan/:id`, `GET /klinik/:id/antrian`.
- **Tahap 3 (Endpoint klaim & resolusi)**: `POST /klinik/:id/panggil-berikutnya`, `POST /kunjungan/:id/lewati`, `POST /kunjungan/:id/tidak-hadir`.
- **Tahap 4 (Regresi & concurrency test menyeluruh)**: Multi-goroutine klaim bersamaan (verifikasi SKIP LOCKED benar-benar paralel, tidak ada double-claim), regresi penuh seluruh project.

## Skema/struktur data

KLINIK {
int id PK
string nama
time jam_buka
time jam_tutup
-- TANPA display_token_hash (item #6 nanti nambah via migration terpisah, ALTER TABLE)
}
QUEUE_COUNTER {
int klinik_id PK_FK
date tanggal PK
int last_number
}
KUNJUNGAN {
int id PK
int pasien_id FK -- REFERENCES pasien(id)
int klinik_id FK -- REFERENCES klinik(id)
int dokter_id FK -- nullable, REFERENCES users(id), diisi atomic saat klaim
date tanggal_kunjungan
int nomor_antrian
bool is_priority -- DEFAULT false
string priority_reason -- nullable
int skip_count -- DEFAULT 0
string status -- CHECK IN ('menunggu','dipanggil','selesai','tidak_hadir'), DEFAULT 'menunggu'
timestamp dipanggil_at -- nullable
timestamp created_at -- DEFAULT now() aman di sini (BUKAN dipakai untuk hash-chain apapun, beda dari audit_log)
}

## Edge case yang perlu dihandle

- **Status `'selesai'` TIDAK PERNAH di-set di fitur ini** — itu transisi milik item #7 (rekam medis). JANGAN buat endpoint/logic apapun yang men-set status ini sekarang.
- **`GET /klinik/:id/antrian` via `X-Display-Token` tanpa cookie** → 401 untuk sekarang (bukan silently ignore atau partial-support) — infrastrukturnya belum ada sampai item #6.
- **`POST /kunjungan` saat tabel `klinik` kosong** (seharusnya mustahil kalau startup seed berhasil, tapi defensif) → 500 dengan log jelas, JANGAN asumsikan selalu ada tanpa cek.
- **Race condition klaim** — `FOR UPDATE SKIP LOCKED` (BUKAN `FOR UPDATE` biasa) WAJIB dipakai persis seperti di docs/TDD.md — ini beda prinsip dari lock `audit_log_tail` (yang sengaja sekuensial). Salah pakai `SKIP LOCKED` di sini bukan bug ringan, itu menghilangkan seluruh tujuan desain paralelisme klaim antar dokter.
- **`priorityReason` tanpa `isPriority=true`** — TIDAK PERLU validasi silang (kalau dikirim reason tanpa isPriority true, terima saja apa adanya, tidak ada requirement eksplisit yang melarang ini).

## Testing

- Seed klinik: startup pertama (tabel kosong) → 1 baris ter-insert sesuai env var. Restart dengan baris sudah ada → TIDAK ada perubahan (manipulasi manual nama klinik di DB dulu, assert tetap sama setelah "restart" ulang fungsi seed).
- `GET /klinik/:id`: sukses; id tidak ada → 404.
- `POST /kunjungan`: sukses (201, nomorAntrian mulai dari 1 untuk hari itu, status='menunggu'); dipanggil setelah jam_tutup (manipulasi waktu test atau jam_tutup klinik) → 400 KLINIK_TUTUP; pasienId tidak ada → 404; role dokter → 403; 2x POST hari sama → nomorAntrian berturutan (1, 2, bukan collision).
- `GET /kunjungan/:id`: sukses; tidak ada → 404.
- `GET /klinik/:id/antrian`: return semua status hari ini urut nomor_antrian; via X-Display-Token tanpa cookie → 401.
- `POST /panggil-berikutnya`: sukses (200, status jadi 'dipanggil', dokter_id dari session BUKAN body meski dikirim beda di body — buktikan diabaikan); antrian kosong → 204; prioritas didahulukan (buat 1 non-prioritas nomor kecil + 1 prioritas nomor besar, assert yang prioritas kepanggil duluan); role petugas → 403.
- `POST /lewati`: sukses dari status 'dipanggil' (200, status balik 'menunggu', skip_count+1); dari status 'menunggu' → 409; tidak ada → 404.
- `POST /tidak-hadir`: sukses dari 'menunggu' DAN dari 'dipanggil' (dua skenario terpisah); dari 'tidak_hadir' (sudah final) → 409.
- Concurrency (WAJIB, lawan Postgres asli via testcontainers-go): minimal 5 goroutine "dokter" berbeda panggil `panggil-berikutnya` BERSAMAAN saat ada 5 kunjungan menunggu → assert SEMUA 5 dapat kunjungan BERBEDA (tidak ada double-claim), DAN waktu eksekusi total signifikan LEBIH CEPAT dibanding kalau dijalankan sekuensial (buktikan SKIP LOCKED benar-benar memberi paralelisme, bukan cuma "kebetulan benar" tapi diam-diam serial).
- Regresi seluruh test suite yang sudah ada.

## Kriteria selesai

Seluruh 4 tahap selesai, seluruh skenario test PASS (termasuk concurrency lawan Postgres asli), `go vet` lolos, regresi tidak ada yang kebreak, user sudah update `docs/api-contract.md` (endpoint `/lewati`) dan verifikasi manual.
