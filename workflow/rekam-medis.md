# Rekam Medis

## Konteks & tujuan

Pencatatan rekam medis terstruktur per kunjungan, immutable dengan koreksi via addendum (bukan edit langsung). Konsumen kedua `audit.Record` (setelah Pasien). Menutup gap yang sengaja ditunda dari item #5: transisi `kunjungan.status` ke `'selesai'`.

## Requirement

### Migration

- `rekam_medis`, `diagnosis`, `tindakan` (3 migration terpisah, urutan dependency: rekam_medis dulu)
- Migration TERPISAH (bukan edit migration audit_log yang sudah closed di item #3): extend CHECK constraint `audit_log.aksi` dari `('create','update')` menjadi `('create','update','addendum')`. WAJIB investigasi nama constraint aktual di DB dulu sebelum ALTER (jangan asumsikan nama default `audit_log_aksi_check` tanpa verifikasi — cek via `\d audit_log` atau query `information_schema`/`pg_constraint`), supaya `DROP CONSTRAINT` tidak gagal karena nama salah.

### Endpoint (shape sesuai docs/api-contract.md, RBAC [dokter] SAJA untuk semua endpoint di fitur ini — TIDAK ADA akses admin/petugas, sesuai desain sengaja di docs/api-contract.md)

- `POST /kunjungan/:id/rekam-medis`
- `POST /rekam-medis/:id/addendum`
- `GET /kunjungan/:id/rekam-medis`
- `GET /pasien/:id/riwayat`

### Nilai `aksi` audit trail (KEPUTUSAN FINAL, sudah melalui debat teknis)

- `POST /kunjungan/:id/rekam-medis` (record awal) → `aksi='create'`
- `POST /rekam-medis/:id/addendum` → `aksi='addendum'` (BUKAN 'create', BUKAN 'update') — alasan: audit trail harus bisa menjawab pertanyaan compliance "berapa banyak koreksi rekam medis" lewat filter `WHERE aksi='addendum'` langsung, tanpa perlu gali `after_data` JSONB untuk cek `addendumOf`.

### `POST /kunjungan/:id/rekam-medis` — detail

- Body: `keluhan`, `hasilPemeriksaan` (wajib), `diagnosis[]` (minimal 1), `tindakan[]` (boleh kosong array, tapi field-nya wajib ada di request).
- WAJIB satu transaksi eksplisit: insert `rekam_medis` (dokter_id dari session, `addendum_of=NULL`) → insert semua `diagnosis[]` → insert semua `tindakan[]` → `UPDATE kunjungan SET status='selesai' WHERE id=?` (menutup gap item #5) → `audit.Record(aksi='create', beforeData=nil, afterData=snapshot lengkap termasuk diagnosis/tindakan)` → commit.
- Kunjungan sudah punya root record aktif (dijaga `uq_rekam_medis_root_per_kunjungan`) → 409 (constraint violation, reactive exception handling, BUKAN preemptive select).
- Kunjungan tidak ditemukan → 404.
- Response 201 sesuai docs/api-contract.md.

### `POST /rekam-medis/:id/addendum` — detail (PALING KRITIS)

- `:id` adalah id record LEAF SAAT INI yang mau di-addend (client kirim id yang mereka pikir masih terkini).
- WAJIB satu transaksi eksplisit:
  1. Fetch record `:id` (harus exist, `deleted_at IS NULL`) — kalau tidak ada → 404.
  2. **MERGE DI BACKEND, BUKAN DI CLIENT** (single source of truth): untuk field opsional (`keluhan`, `hasilPemeriksaan`, `diagnosis`, `tindakan`) yang TIDAK DIKIRIM di request (key absent di JSON, BUKAN array kosong `[]` — bedakan dua ini secara eksplisit, `[]` berarti "sengaja dikosongkan", absent berarti "carry-over dari parent"), ambil nilainya dari record `:id` yang di-fetch di langkah 1.
  3. Insert row `rekam_medis` BARU: `kunjungan_id` = sama dengan parent (bukan dari request), `dokter_id` dari session (TIDAK dibatasi harus dokter yang sama dengan penulis awal — dokter manapun boleh addend), `addendum_of=:id`, `alasan_addendum` dari request (wajib), field lain hasil merge dari langkah 2.
  4. Insert `diagnosis[]`/`tindakan[]` hasil merge (COPY rows dari parent kalau carry-over, atau dari request kalau dikirim eksplisit).
  5. `audit.Record(aksi='addendum', beforeData=snapshot record :id SEBELUM addendum, afterData=snapshot record BARU setelah merge)`.
  6. Commit.
- **Constraint violation `uq_addendum_of_active`** (staff lain sudah addend `:id` duluan, di antara fetch dan insert) → 409, map dari `23505` (reactive exception handling, pattern established). Response harus jelas ke client: "refetch versi terbaru & retry" sesuai docs/api-contract.md.
- Response 201 sesuai docs/api-contract.md.

### `GET /kunjungan/:id/rekam-medis` — detail

- Leaf query PERSIS pattern di docs/TDD.md (`NOT EXISTS` traversal, `deleted_at IS NULL`) — JANGAN pakai flag `is_latest` (tidak ada di skema, dan memang sengaja tidak ada, sesuai docs/AGENTS.md §7).
- Tidak ada rekam medis sama sekali untuk kunjungan itu → 404.
- Response { id, keluhan, hasilPemeriksaan, diagnosis[], tindakan[], isAddendum, createdAt } sesuai docs/api-contract.md.

### `GET /pasien/:id/riwayat` — detail

- List semua `kunjungan` milik pasien itu, JOIN leaf `rekam_medis` masing-masing.
- Keputusan (diambil langsung, tidak eksplisit di dokumen manapun): kunjungan yang BELUM punya rekam medis sama sekali (masih 'menunggu'/'dipanggil', atau 'tidak_hadir') TIDAK muncul di list ini — endpoint ini untuk review riwayat KLINIS, bukan riwayat kunjungan administratif (itu sudah ada di `pasien.riwayatKunjunganRingkas` dari item #4). Kalau ini salah baca intent, tolong koreksi.
- Response sesuai docs/api-contract.md.

## Tahapan implementasi

- **Tahap 1 (Migration & query dasar)**: 3 migration rekam_medis/diagnosis/tindakan + migration extend CHECK audit_log.aksi + query sqlc (insert lengkap dengan children, leaf query, addendum insert, list untuk riwayat).
- **Tahap 2 (POST rekam-medis awal)**: endpoint create + transisi status kunjungan + audit integration.
- **Tahap 3 (POST addendum)**: endpoint addendum + merge logic backend + audit integration + 409 handling.
- **Tahap 4 (GET endpoints)**: `GET /kunjungan/:id/rekam-medis`, `GET /pasien/:id/riwayat`.
- **Tahap 5 (Regresi & concurrency test)**: race condition 2 addendum bersamaan ke parent yang sama (cuma 1 boleh sukses via `uq_addendum_of_active`), regresi penuh.

## Skema/struktur data

REKAM_MEDIS {
int id PK
int kunjungan_id FK
int dokter_id FK
text keluhan
text hasil_pemeriksaan
bool is_addendum -- true kalau addendum_of NOT NULL
int addendum_of FK -- nullable, self-reference
text alasan_addendum -- nullable, WAJIB terisi kalau is_addendum=true
timestamp deleted_at -- soft delete, nullable
timestamp created_at
}
-- UNIQUE INDEX uq_addendum_of_active ON rekam_medis(addendum_of) WHERE deleted_at IS NULL
-- UNIQUE INDEX uq_rekam_medis_root_per_kunjungan ON rekam_medis(kunjungan_id) WHERE addendum_of IS NULL AND deleted_at IS NULL
DIAGNOSIS { int id PK; int rekam_medis_id FK; string kode_icd nullable; text deskripsi }
TINDAKAN { int id PK; int rekam_medis_id FK; string jenis; text deskripsi }

## Edge case yang perlu dihandle

- **Nil vs empty-array di request addendum** — WAJIB dibedakan secara eksplisit di level parsing request (pakai pointer/nullable-aware type, mis. `*[]DiagnosisInput` di Go, JANGAN pakai slice biasa yang tidak bisa bedakan "key absent" dari "key ada tapi isinya `[]`"). Key absent → carry-over. Key ada dengan `[]` → sengaja dikosongkan, JANGAN carry-over.
- **Merge WAJIB di backend** — jangan percaya client kirim state lengkap. Fetch leaf dulu, merge di server, baru insert.
- **`uq_rekam_medis_root_per_kunjungan`** — mencegah 2 root record independen untuk 1 kunjungan yang sama; tanpa ini, leaf query bisa diam-diam sembunyikan salah satu (data loss tanpa notifikasi).
- **`alasan_addendum` wajib untuk addendum**, TIDAK relevan untuk record awal (biarkan NULL).
- **Addendum tidak dibatasi ke dokter penulis awal** — dokter manapun (dengan role dokter) boleh addend record siapapun.

## Testing

- Migration: 3 tabel + constraint (`uq_addendum_of_active`, `uq_rekam_medis_root_per_kunjungan`, CHECK audit_log.aksi extended) — lawan Postgres asli.
- `POST /kunjungan/:id/rekam-medis`: sukses (201, kunjungan.status jadi 'selesai', audit_log aksi='create'); kunjungan sudah punya root record → 409; kunjungan tidak ada → 404; role petugas/admin → 403.
- `POST /rekam-medis/:id/addendum`: sukses dengan SEBAGIAN field dikirim (200/201, field yang tidak dikirim ke-carry-over BENAR dari parent — assert eksplisit nilainya, bukan cuma "ada"); field array dikirim `[]` eksplisit → hasil kosong (BUKAN carry-over, buktikan beda dari kasus absent); `:id` sudah bukan leaf (sudah di-addend duluan) → 409; `:id` tidak ada → 404; audit_log aksi='addendum' (BUKAN 'create').
- `GET /kunjungan/:id/rekam-medis`: return leaf yang benar setelah addendum berkali-kali (buat chain 3 level, assert yang return cuma yang terbaru); tidak ada rekam medis → 404.
- `GET /pasien/:id/riwayat`: cuma kunjungan yang punya rekam medis yang muncul; kunjungan tanpa rekam medis tidak muncul.
- Concurrency (WAJIB lawan Postgres asli): 2 goroutine addendum BERSAMAAN ke parent yang SAMA → assert PERSIS 1 sukses, 1 gagal 409, `uq_addendum_of_active` benar-benar mencegah race.
- Regresi seluruh test suite yang sudah ada.

## Kriteria selesai

Seluruh 5 tahap selesai, seluruh skenario test PASS (termasuk concurrency addendum lawan Postgres asli), `go vet` lolos, regresi tidak ada yang kebreak, user verifikasi manual.
