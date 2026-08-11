# Pasien

## Konteks & tujuan

Manajemen data pasien: registrasi, pencarian, detail, edit biodata dengan optimistic locking. Konsumen pertama helper `audit.Record` (item #3) untuk operasi create maupun edit. Endpoint ini dipakai frontend item #14 nanti (belum ada sekarang).

## Requirement

### Migration (1 tabel, sesuai ERD di docs/TDD.md, dengan penyesuaian dijelaskan di section Edge case)

- `pasien`

### Endpoint (shape sesuai docs/api-contract.md persis)

- `POST /pasien` [petugas, admin]
- `GET /pasien/search?nik=&nama=&page=&limit=` [petugas, dokter, admin]
- `GET /pasien/:id` [petugas, dokter, admin]
- `PATCH /pasien/:id` [petugas, admin]

### Integrasi audit trail (WAJIB, keputusan final — beda dari kalimat literal docs/TDD.md lama)

Baik `POST /pasien` (create) MAUPUN `PATCH /pasien/:id` (edit) WAJIB memanggil `audit.Record` (package `internal/audit`, item #3) di TRANSAKSI YANG SAMA dengan write bisnisnya — pola identik dengan yang sudah dipakai project ini di tempat lain: `pool.Begin` → write bisnis (pakai `q.WithTx(tx)`) → `audit.Record(ctx, tx, q, actorUserID, "pasien", pasienID, aksi, beforeData, afterData)` → `tx.Commit`.

- `POST /pasien` sukses → `aksi="create"`, `beforeData=nil`, `afterData` = snapshot biodata lengkap yang baru dibuat.
- `PATCH /pasien/:id` sukses (version match) → `aksi="update"`, `beforeData` = snapshot biodata SEBELUM diubah, `afterData` = snapshot biodata SETELAH diubah.
- `PATCH /pasien/:id` GAGAL (409 version mismatch, atau 404 not found) → `audit.Record` TIDAK dipanggil sama sekali (tidak ada perubahan data bisnis yang terjadi, tidak ada yang perlu diaudit).
- Snapshot biodata untuk `beforeData`/`afterData` mencakup KECUALI `id`, `version`, `deletedAt` (field kontrol internal, bukan data pasien): `nik`, `nama`, `tanggalLahir`, `jenisKelamin`, `alamat`, `noTelp`. Untuk `aksi="create"`, `afterData` JUGA sertakan `consentAt`.
- `actorUserID` diambil dari context sesi (auth middleware, sudah ada dari Auth & RBAC), BUKAN dari body/params — pola identik `dokterId` yang sudah ditetapkan di docs/AGENTS.md §7 untuk kasus serupa.

### Konsent (WAJIB true, keputusan berdasarkan ERD — `consent_at` TIDAK ditandai nullable, beda dari `nik` yang eksplisit nullable)

`POST /pasien` dengan `consent=false` atau field `consent` tidak dikirim → TOLAK 400 (code kurasi, mis. `CONSENT_REQUIRED`), TIDAK ADA row `pasien` maupun `audit_log` yang ter-insert. `consent=true` → `consent_at = time.Now()` (digenerate di Go, konsisten dengan pola `created_at` di audit_log — supaya nilai yang disimpan predictable, bukan tergantung `DEFAULT now()` DB).

### `PATCH /pasien/:id` — field yang BOLEH diubah

`nik`, `nama`, `tanggalLahir`, `jenisKelamin`, `alamat`, `noTelp` — SEMUA opsional per-request (cuma field yang dikirim yang diupdate, field lain tetap). `consent`/`consentAt` TIDAK BOLEH diubah lewat endpoint ini (itu timestamp legal sekali-catat saat registrasi, bukan field yang di-"edit" — kalau request mengirim field ini, ABAIKAN, JANGAN error, JANGAN ubah nilainya).

### Optimistic locking (`PATCH /pasien/:id`)

Query WAJIB atomic: `UPDATE pasien SET ..., version = version + 1 WHERE id = $1 AND version = $2 AND deleted_at IS NULL RETURNING *`. Kalau 0 rows affected, WAJIB query tambahan untuk membedakan penyebab (BUKAN preemptive SELECT sebelum write untuk validasi — ini lookup SETELAH write gagal, murni untuk kejelasan pesan error, prinsipnya beda dari larangan preemptive-select di docs/AGENTS.md §7):

- `id` tidak ditemukan (atau `deleted_at` sudah terisi) → 404.
- `id` ditemukan tapi `version` tidak match → 409.

### `GET /pasien/search`

- `nik` → EXACT match (bukan partial) — kontras eksplisit dari `nama` yang didokumentasikan sebagai partial match di docs/api-contract.md, jadi `nik` disengaja beda.
- `nama` → partial match (`ILIKE '%...%'`).
- Kalau KEDUANYA dikirim → AND (mempersempit hasil, bukan OR).
- Filter `deleted_at IS NULL` di semua query baca (`search` dan `GET :id`) — defensif walau belum ada mekanisme delete aktif sekarang (lihat Edge case).
- Pagination wajib (`page`/`limit`) sesuai docs/AGENTS.md §7, default masuk akal kalau tidak dikirim (kamu tentukan angkanya, dokumentasikan di kode).

### `GET /pasien/:id`

`riwayatKunjunganRingkas` WAJIB return array kosong `[]` (tabel `kunjungan` belum ada sampai item #5) — field ini tetap harus ada di response sesuai shape docs/api-contract.md, cuma isinya kosong untuk sekarang.

## Tahapan implementasi

- **Tahap 1 (Migration & query dasar)**: Migration `pasien` sesuai skema di bawah. Query sqlc: insert, get by id (dengan filter `deleted_at IS NULL`), search (nik exact + nama partial + AND + pagination, filter `deleted_at IS NULL`), update dengan optimistic lock (atomic `WHERE version=?`).
- **Tahap 2 (Endpoint & audit integration)**: 4 endpoint, RBAC per role di atas, integrasi `audit.Record` untuk create & update, validasi consent wajib true, follow-up lookup 404 vs 409 di PATCH.
- **Tahap 3 (Regresi & test integrasi)**: Test lintas endpoint (registrasi → search → detail → edit → verifikasi audit trail), concurrency test optimistic lock (2 request PATCH bersamaan versi sama, cuma 1 boleh sukses), regresi penuh seluruh test suite project.

## Skema/struktur data

PASIEN {
int id PK
string nik -- nullable, TIDAK ada UNIQUE constraint (disengaja, warning bukan block, lihat Edge case)
string nama
date tanggal_lahir
string jenis_kelamin -- CHECK (jenis_kelamin IN ('L', 'P'))
string alamat
string no_telp
timestamp consent_at -- NOT NULL (wajib terisi, app reject request sebelum insert kalau consent≠true — lihat Requirement)
int version -- NOT NULL DEFAULT 1
timestamp deleted_at -- nullable, TIDAK ADA mekanisme yang men-set nilainya di scope ini (lihat Edge case)
}

## Edge case yang perlu dihandle

- **Duplikasi NIK — SENGAJA tidak divalidasi/diblokir di backend.** `POST /pasien` dengan NIK yang sudah ada di row lain WAJIB tetap sukses 201, menghasilkan 2+ row dengan NIK sama. Warning ke user adalah tanggung jawab FRONTEND SEPENUHNYA (pre-submission check via `GET /pasien/search?nik=` sebelum submit form, sudah diputuskan eksplisit) — JANGAN tambahkan validasi/warning apapun di sisi backend untuk ini, JANGAN tambahkan field baru di response `POST /pasien` untuk mensinyalkan duplikasi. Race condition antara dua petugas submit NIK sama nyaris bersamaan (precheck FE tidak sempat kejar) SUDAH DITERIMA sebagai konsekuensi dari keputusan ini, BUKAN bug yang perlu ditambal.
- **`deleted_at` — kolom ada, mekanisme belum ada.** Migration WAJIB buat kolom ini (dipakai filter defensif di semua query baca), TAPI tidak ada endpoint/logic apapun di scope ini yang mengisi nilainya. Ini bukan kelalaian — soft-delete pasien belum masuk scope backlog manapun sampai sekarang.
- **`jenis_kelamin` CHECK constraint** — nilai `'L'`/`'P'` adalah ASUMSI (tidak ada di dokumen manapun secara eksplisit, keputusan diambil langsung untuk menjaga konsistensi dengan pola CHECK constraint yang sudah dipakai project ini di kolom enum-like lain). Kalau ternyata FE nanti (item #14) butuh nilai lain, ini WAJIB migration baru untuk ubah constraint-nya — jangan diam-diam disesuaikan tanpa migration.
- **Field `consent`/`consentAt` di request `PATCH`** — kalau dikirim, WAJIB diabaikan diam-diam (bukan 400 error) — ini bukan "requirement ambigu" yang perlu ditolak, cuma field yang tidak relevan untuk endpoint ini.
- **NIK — TIDAK PERLU validasi format** (16 digit, dsb). Tidak ada requirement eksplisit soal ini di dokumen manapun — JANGAN menambahkan validasi format yang tidak diminta.

## Testing

- `POST /pasien`: sukses (201, `consent_at` terisi, `version=1`, row `audit_log` ter-insert `aksi='create'` dengan snapshot benar); `consent=false`/tidak dikirim → 400, TIDAK ADA row pasien maupun audit_log; NIK duplikat → tetap 201 (2 row ter-insert, buktikan tidak ada block); role `dokter` → 403.
- `GET /pasien/search`: filter `nik` exact match; filter `nama` partial match; kombinasi keduanya (AND, bukan OR); pagination bekerja; row dengan `deleted_at` terisi (manipulasi langsung di DB saat test) TIDAK muncul di hasil.
- `GET /pasien/:id`: sukses (200, `riwayatKunjunganRingkas=[]`); id tidak ada → 404; id dengan `deleted_at` terisi → 404 juga (defensif).
- `PATCH /pasien/:id`: sukses dengan version benar (200, `version` bertambah 1, row `audit_log` ter-insert `aksi='update'` dengan `beforeData`/`afterData` snapshot benar, field yang tidak dikirim tidak berubah); version salah/basi → 409 (row TIDAK berubah, TIDAK ADA audit_log baru); id tidak ada → 404; kirim field `consent` di body → diabaikan diam-diam, tidak error; role `dokter` → 403.
- Concurrency optimistic lock (WAJIB lawan Postgres asli via testcontainers-go): 2 goroutine PATCH bersamaan dengan `version` awal SAMA → assert PERSIS 1 yang sukses (200), 1 gagal (409), `version` akhir di DB = awal+1 (BUKAN +2).
- Regresi seluruh test suite yang sudah ada (scaffolding, Auth & RBAC, Audit Trail).

## Kriteria selesai

Seluruh 3 tahap selesai, seluruh skenario test di atas PASS (termasuk concurrency lawan Postgres asli), `go vet` lolos, regresi tidak ada yang kebreak, user sudah verifikasi manual (registrasi pasien, search by nik/nama, lihat detail, edit biodata, cek isi audit_log via query langsung ke DB).
