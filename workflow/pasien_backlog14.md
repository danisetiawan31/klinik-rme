# Pasien

## Konteks & tujuan
Modul CRUD pasien untuk staff. Petugas & admin: registrasi + edit biodata. Dokter: akses baca (search/detail) untuk konteks klinis, tanpa create/edit. Backend sudah lengkap (item 4, "Selesai Penuh", + addendum X-Total-Count) — fitur ini murni consumer FE terhadap `POST /pasien`, `GET /pasien/search`, `GET /pasien/:id`, `PATCH /pasien/:id`.

## Requirement

**Registrasi**
- Form: nik? (nullable), nama, tanggalLahir (native `<input type="date">`), jenisKelamin (native `<select>`, 2 opsi L/P — styling manual pakai token semantik, BUKAN fetch primitive Spartan baru, cuma 2 opsi gak sepadan biaya fetch+register komponen), alamat (native textarea), noTelp, consent (native checkbox, wajib true sebelum submit).
- Validasi client: consent wajib dicentang; NIK kalau diisi wajib persis 16 digit angka (validasi format saja, BUKAN validitas/checksum — backend sengaja longgar soal ini, lihat done_be.md catatan Pasien).
- Warning duplikasi NIK: pre-submission check via `GET /pasien/search?nik=` begitu NIK genap 16 digit ke-input. Non-blocking — tampilkan banner "NIK sudah terdaftar atas nama: X" kalau ketemu, staff tetap bisa lanjut submit.

**Pencarian + Detail + Riwayat ringkas**
- Form search: input `nik` dan `nama`, bisa dipakai salah satu atau dua-duanya.
- Trigger search — Opsi A: `nama` live search dengan debounce 300ms; `nik` auto-trigger search HANYA saat genap 16 digit ke-input (bukan tiap keystroke, bukan tiap kurang dari 16 digit).
- Hasil: list ringkas (NIK ter-mask via `SensitiveValueComponent` mode="display", nama, tanggalLahir), tiap item klik → navigasi ke `/pasien/:id`.
- `PaginationComponent` baru (`shared/components/`) — generic & reusable (dipakai ulang nanti oleh Antrian #15 & Admin #18). Baca header `X-Total-Count` dari response (`observe: 'response'` di HTTP call), hitung `totalPages = Math.ceil(totalCount / limit)`, tampilkan label "Halaman X dari Y" + tombol prev/next dengan disable state yang benar di halaman awal/akhir.
- Halaman detail (`/pasien/:id`): biodata lengkap + `riwayatKunjunganRingkas` (kunjunganId, tanggal, status) sebagai list sederhana — bukan data klinis, aman ditampilkan apa adanya.
- Route `/pasien/riwayat` di shortcut dokter (Landing/Shell) TIDAK di-wire di tahap ini — itu scope item 17 (Rekam Medis). Dokter tetap dapat akses baca lewat route generik `/pasien` (RBAC search/detail sudah include dokter).

**Edit biodata**
- Form edit pre-filled dari data detail, `PATCH /pasien/:id` dengan `version`.
- 409 optimistic lock (staff lain sudah edit duluan) — UX hybrid: tampilkan banner inline di form ("Data sudah diubah staff lain") dengan tombol eksplisit "Muat versi terbaru". Refetch + reset form HANYA terjadi saat staff klik tombol itu — field yang sedang diisi TETAP ke-preserve sampai staff sendiri yang pilih discard & reload.

**RBAC (roleGuard, sudah siap, variadic)**
- Registrasi & Edit: `roleGuard('petugas', 'admin')`
- Search, Detail, Riwayat: `roleGuard('petugas', 'dokter', 'admin')`

## Tahapan implementasi
- **Tahap 1 (Registrasi)**: form + consent + validasi NIK format + warning duplikasi NIK pre-submission. Route `/pasien/baru`.
- **Tahap 2 (Pencarian + Detail + Riwayat ringkas)**: `PaginationComponent` baru, search trigger logic (debounce nama / auto-trigger NIK 16 digit), route `/pasien` (search) & `/pasien/:id` (detail).
- **Tahap 3 (Edit biodata)**: form edit + optimistic lock 409 hybrid UX. Route `/pasien/:id/edit`.

## Skema/struktur data
- `features/pasien/pasien.types.ts`: `Pasien`, `PasienSearchItem`, `CreatePasienRequest`, `UpdatePasienRequest` — shape persis sesuai `api-contract.md` section Pasien.
- `core/types/pagination.types.ts` (baru, kalau belum ada): `PaginationParams { page, limit }` — generic, dipakai ulang modul lain.
- Tidak ada perubahan skema DB/backend di tahap ini (sudah selesai lewat addendum X-Total-Count).

## Edge case yang perlu dihandle
- NIK kosong (nullable) — pasien fallback-ID, cuma bisa dicari lewat nama.
- Hasil search kosong — tampilkan empty state yang jelas, bukan halaman blank.
- `consent: false` saat submit — backend return 400 `CONSENT_REQUIRED`, FE validasi duluan di client tapi tetap handle response error ini sebagai fallback.
- NIK diisi tapi bukan 16 digit angka — blok submit di client dengan pesan jelas, jangan biarkan nembak request ke backend.
- Warning duplikasi NIK genuinely non-blocking — staff yang sengaja lanjut submit walau ada warning HARUS tetap bisa (sesuai desain backend).
- Optimistic lock 409 — field yang sedang diisi staff TIDAK BOLEH hilang sebelum staff eksplisit klik "Muat versi terbaru".

## Testing
- Role guard: kombinasi akses benar per route (petugas/admin untuk create-edit; +dokter untuk search/detail/riwayat).
- Form validation: consent wajib, format NIK (16 digit numeric kalau diisi), field required lainnya.
- Search trigger: assert nama TIDAK nembak request tiap keystroke (nunggu debounce 300ms); NIK TIDAK nembak request sebelum genap 16 digit, nembak tepat saat genap 16.
- `PaginationComponent`: `totalPages` dihitung benar dari header `X-Total-Count` + `limit`; disable state prev/next benar di halaman pertama & terakhir.
- 409 hybrid UX: banner muncul saat response 409; field form tidak ter-reset sebelum tombol "Muat versi terbaru" diklik; refetch + reset form terjadi tepat saat tombol diklik.
- Warning duplikasi NIK: muncul saat NIK match ditemukan, tapi submit tetap bisa lanjut (assert tidak ada blocking).

## Kriteria selesai
Ketiga tahap selesai, seluruh skenario testing di atas lolos (Vitest), regresi test frontend existing tetap hijau, dan dicek ulang manual oleh user (routing, RBAC per role, UX 409) sebelum masuk `done_fe.md`.