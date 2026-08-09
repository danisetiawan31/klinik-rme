# api-contract.md — Modul RME & Antrian Klinik

## Konvensi
- Base path: `/api/v1`
- JSON: camelCase
- Auth per channel:
  - REST, staff: cookie `httpOnly` (otomatis via origin sama, lihat TDD)
  - REST, papan antrian: header `X-Display-Token`
  - WebSocket, staff: cookie (otomatis, sama seperti REST)
  - WebSocket, papan antrian: query param `displayToken` — browser WebSocket API tidak bisa set custom header, jadi transportnya beda dari REST secara sengaja, bukan kelalaian
- Sukses: return resource langsung, tanpa envelope
- Error: `{ "error": { "code", "message", "requestId" } }` — `message` dikurasi per `code` (aman ditampilkan ke user, mis. "NIK sudah terdaftar"), **bukan** raw error dari DB/Go (mencegah kebocoran data seperti NIK ikut ke-embed di pesan error). `requestId` (UUID per request) dicatat di server log bareng detail lengkap (stack trace, raw error, query params) untuk debugging — `grep requestId` di log, tanpa expose apa pun ke client.
- Status: 200/201/204 sukses, 400 validasi, 401 unauth, 403 role tidak sesuai, 404 not found, 409 conflict (unique/optimistic-lock violation)

## Auth
```
POST /auth/login                          [public]
  → { email, password }
  ← 200 { user: { id, nama, roles[] } }, set cookie

POST /auth/logout                         [any authenticated]
  ← 204, clear cookie

GET  /auth/me                             [any authenticated]
  ← 200 { id, nama, roles[] } | 401 kalau sesi tidak valid
  Perlu karena cookie httpOnly tidak bisa dibaca JS — Angular butuh cara cek "siapa yang login" saat app load.

PATCH /auth/me/password                   [any authenticated]
  → { passwordLama, passwordBaru }
  ← 204 | 400 kalau passwordLama tidak cocok

POST /auth/forgot-password                [public]
  → { email }
  ← 200 selalu (generik, sama respons baik email terdaftar atau tidak — cegah user enumeration)
  Token TIDAK dikembalikan di response ini, terlepas email valid atau tidak. Dikirim via Resend kalau akun ada. Untuk demo: token ada di server log, bukan di response.

POST /auth/reset-password                 [public]
  → { token, passwordBaru }
  ← 204 | 400 kalau token invalid/expired/sudah dipakai
```

## Klinik
```
GET  /klinik/:id                                       [petugas, dokter, admin]
  ← 200 { id, nama, jamBuka, jamTutup }
  Dipakai FE untuk enforce "pendaftaran terkunci setelah jam tutup" di sisi client — validasi asli tetap di server.

POST /admin/klinik/:id/display-token/regenerate        [admin]
  ← 200 { displayToken }  (token mentah, cuma muncul sekali di response ini — DB cuma simpan hash)
```

## Pasien
```
POST /pasien                              [petugas, admin]
  → { nik?, nama, tanggalLahir, jenisKelamin, alamat, noTelp, consent: bool }
  ← 201 { id, ...biodata }

GET  /pasien/search?nik=&nama=&page=&limit=   [petugas, dokter, admin]
  ← 200 [{ id, nik, nama, tanggalLahir }]   (nik dan nama boleh salah satu atau dua-duanya; nama = partial match — wajib ada karena pasien tanpa NIK cuma bisa dicari lewat nama)

GET  /pasien/:id                          [petugas, dokter, admin]
  ← 200 { id, ...biodata, version, riwayatKunjunganRingkas: [{ kunjunganId, tanggal, status }] }
  riwayatKunjunganRingkas cuma status kunjungan, bukan isi klinis — aman diakses petugas.

PATCH /pasien/:id                         [petugas, admin]
  → { ...field yang diubah, version }
  ← 200 { ...pasien terbaru, version: version+1 } | 409 kalau version tidak cocok (optimistic lock, staff lain sudah edit duluan)
```

## Kunjungan & Antrian
```
POST /kunjungan                                    [petugas, admin]
  → { pasienId, isPriority?, priorityReason? }
  ← 201 { id, nomorAntrian, status: "menunggu", tanggalKunjungan } | 400 { code: "KLINIK_TUTUP" } kalau sudah lewat jamTutup
  nomorAntrian dari atomic upsert counter (lihat TDD) — tidak pernah duplikat walau dua request bersamaan.
  Jam operasional ditegakkan di sini (bukan di POST /pasien) — biodata pasien tetap boleh diinput kapan saja, yang dibatasi cuma masuk antrian hari itu.

GET  /kunjungan/:id                                [petugas, dokter, admin]
  ← 200 { id, pasienId, nomorAntrian, status, isPriority, dokterId, dipanggilAt }

GET  /klinik/:id/antrian                           [petugas, dokter, admin via cookie | display-token]
  Response beda tergantung channel auth — disengaja, bukan lupa:
  ← via X-Display-Token: 200 [{ nomorAntrian, status, isPriority }]                    (tanpa identitas pasien — publik, ruang tunggu)
  ← via cookie staff:    200 [{ id, nomorAntrian, status, isPriority, pasienNama }]     (staff butuh nama untuk memanggil)

POST /klinik/:id/panggil-berikutnya                [dokter]
  ← 200 { id, nomorAntrian, pasienNama, dokterId, dipanggilAt } | 204 kalau antrian kosong
  dokterId dari session (bukan body/params), tanggal = hari ini dari jam server. Petugas tidak bisa memanggil pasien.

POST /kunjungan/:id/tidak-hadir                    [dokter, admin]
  ← 200 { id, status: "tidak_hadir" }
  Dipakai saat penutupan hari untuk pasien yang berkali-kali di-skip dan tidak pernah kembali.
```

## Rekam Medis
```
POST /kunjungan/:id/rekam-medis                    [dokter]
  → { keluhan, hasilPemeriksaan, diagnosis: [{ kodeIcd?, deskripsi }], tindakan: [{ jenis, deskripsi }] }
  ← 201 { id, ...isi, createdAt }

POST /rekam-medis/:id/addendum                     [dokter]
  → { keluhan?, hasilPemeriksaan?, diagnosis?, tindakan?, alasanAddendum }
  ← 201 { id, addendumOf: :id, ... } | 409 kalau :id sudah bukan versi terkini (staff lain sudah addend duluan — refetch versi terbaru & retry)

GET  /kunjungan/:id/rekam-medis                    [dokter]
  ← 200 { id, keluhan, hasilPemeriksaan, diagnosis[], tindakan[], isAddendum, createdAt }
  Versi terkini (leaf) saja — bukan seluruh histori addendum.

GET  /pasien/:id/riwayat                           [dokter]
  ← 200 [{ kunjunganId, tanggal, rekamMedis: { ...versi terkini saja } }]
  Histori lengkap tiap koreksi ada di audit log (lewat admin), bukan di endpoint ini — supaya dokter yang cuma butuh kondisi terkini tidak kebanjiran versi lama yang sudah dikoreksi.
```

**Catatan keamanan penting soal akses Rekam Medis:** section ini sengaja dibatasi `[dokter]` saja — admin tidak punya akses baca langsung, meski admin adalah role tertinggi untuk urusan lain (kelola user, dst). Ini konsisten dengan scope admin di PRD ("kelola user, lihat audit log", bukan "akses rekam medis"). **Penting dipahami batasnya:** ini bukan proteksi mutlak — admin tetap bisa merekonstruksi isi rekam medis lewat `GET /admin/audit-log/:id` (karena `beforeData`/`afterData` di situ persis konten klinis), karena admin punya akses ke database yang sama dan itu tidak realistis dicegah dari sisi mana pun. Yang sebenarnya dicapai desain ini adalah **mencegah exposure insidental/tidak sengaja** — admin yang menjalankan tugas rutin tidak akan "kebetulan" ketemu isi diagnosis pasien lewat endpoint yang dipakainya sehari-hari; untuk melihat konten klinis, admin harus secara sadar tahu `recordId` spesifik dan sengaja query lewat jalur audit, yang meninggalkan jejak permintaan jelas (`tabelTarget=rekam_medis`). Legitimate security posture untuk skala portfolio, bukan security theater — tapi juga bukan proteksi terhadap admin yang benar-benar berniat jahat.

## Admin
```
GET  /admin/audit-log?tabelTarget=&recordId=&actorId=&page=&limit=   [admin]
  ← 200 [{ id, tabelTarget, recordId, actorUserId, aksi, createdAt }]
  Tanpa beforeData/afterData — list tetap ringan, tidak diam-diam nge-dump data medis lewat overview.

GET  /admin/audit-log/:id                                [admin]
  ← 200 { id, tabelTarget, recordId, actorUserId, aksi, beforeData, afterData, hashEntry, createdAt }
  Detail penuh, termasuk isi perubahan — dipakai untuk investigasi/dispute spesifik.

GET  /admin/users?page=&limit=                           [admin]
  ← 200 [{ id, nama, email, roles[] }]

POST /admin/users                                        [admin]
  → { nama, email, roles[] }
  ← 201 { id, nama, email, roles[], inviteLink }
  Tanpa password — sistem generate token invite, kirim via Resend, user set password sendiri. inviteLink dikembalikan juga ke admin (caller sudah admin ter-otentikasi, beda konteks dari forgot-password yang publik) buat kebutuhan demo cepat tanpa cek inbox.

POST /admin/users/:id/resend-invite                      [admin]
  ← 204
  Kalau email invite pertama gagal terkirim atau expired — generate token baru, kirim ulang. Pembuatan user tidak pernah gagal gara-gara ini.

PATCH /admin/users/:id                                   [admin]
  → { nama?, email? }
  ← 200 { id, nama, email, roles[] }
  Buat koreksi data user, termasuk email salah/tidak bisa diakses — setelah dikoreksi, trigger resend-invite atau user coba forgot-password lagi.

PATCH /admin/users/:id/roles                             [admin]
  → { roles[] }
  ← 200 { id, roles[] }
```

## Laporan
```
GET  /laporan/harian?tanggal=...                   [petugas, dokter, admin]
  ← 200 { tanggal, totalKunjungan, totalSelesai, totalTidakHadir }
  Rekap operasional, bukan data klinis — aman diakses semua role staff.
```

## Operasional
```
GET  /health                              [public]
  ← 200 { status: "ok", db: "ok" } | 503 kalau koneksi DB gagal
  Tanpa auth — dipakai container/reverse-proxy untuk health check, bukan buat konsumsi FE.
```

## WebSocket
```
WS /ws?klinikId=X[&displayToken=...]      [petugas, dokter, admin via cookie | display-token]
  Server → client: { "type": "queue_updated" }   (invalidation ping saja — client wajib refetch GET /klinik/:id/antrian, tidak ada data dikirim langsung lewat socket)
```
