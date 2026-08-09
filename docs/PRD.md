# PRD — Modul RME & Antrian Klinik

## Latar Belakang
Surat Edaran Bersama (diteken 30 Juli 2026 oleh BPJS Kesehatan, Kemenkes, Kemendagri, KPK, BSSN) mendorong rumah sakit mitra BPJS Kesehatan beralih dari klaim berbasis dokumen pemindaian ke klaim elektronik berbasis Rekam Medis Elektronik (RME) yang terhubung SATUSEHAT. Masa transisi berlaku mulai Agustus 2026, dengan enforcement penuh ("No RME, No Claim") ditargetkan berlaku nasional pada 2027. Klinik kecil belum menjadi sasaran eksplisit regulasi ini, namun mengingat arah kebijakan digitalisasi rekam medis nasional yang menyeluruh, tren ini punya kemungkinan besar meluas ke fasilitas kesehatan primer dalam waktu dekat. Terlepas dari kapan regulasi resmi menjangkau klinik, masalah operasionalnya sudah nyata hari ini: pencatatan ganda (manual + digital) dan minimnya mekanisme audit yang bisa dipercaya untuk melacak perubahan data pasien. Project ini memposisikan klinik untuk siap lebih awal.

## Scope
Bukan integrasi resmi ke SATUSEHAT (butuh akses API Kemenkes, di luar jangkauan). Fokus: modul internal klinik yang mencatat RME terstruktur (skema selaras konsep SATUSEHAT), antrian real-time tanpa duplikasi nomor, dan audit trail tamper-evident.

**Skala:** 1 klinik kecil, 1 alur pemeriksaan generik (bukan RS multi-poli), tidak dipecah ke fase — dikerjakan sebagai satu MVP utuh.

**Data demo/testing:** seluruh data demo/testing (termasuk seed data kalau di-share publik) wajib data fiktif/sintetis, bukan data pasien asli.

## Role/Aktor
- **Petugas pendaftaran** — daftar pasien baru/lama, generate nomor antrian, akses read terbatas ke data administratif saja
- **Dokter** — panggil antrian, isi rekam medis
- **Admin** — kelola user, lihat audit log, regenerate display token papan antrian
- 1 user bisa merangkap >1 role

## Alur Operasional
1. Pasien datang → petugas cari data by NIK (fallback ID internal kalau NIK kosong). Warning (bukan block) kalau NIK sudah pernah terdaftar sebelumnya, tanpa syarat tanggal.
2. Baru: input biodata + consent pengumpulan data pribadi. Existing: buka rekam medis lama (versi terkini).
3. Nomor antrian digenerate otomatis, unik per klinik per hari.
4. Papan antrian menampilkan status secara real-time.
5. Dokter panggil pasien berikutnya — pasien prioritas (lansia/difabel/ibu hamil/darurat) didahulukan dari urutan nomor biasa; pasien yang di-skip (no-show) kembali ke antrian, bukan hilang. Pasien yang berkali-kali tidak hadir dan tidak kembali sampai jam tutup ditandai "tidak hadir" secara manual oleh admin/dokter saat penutupan hari.
6. Dokter isi rekam medis: keluhan, hasil pemeriksaan, diagnosis (bisa >1), tindakan/resep (bisa >1).
7. Simpan → tercatat otomatis di audit trail (siapa, kapan, apa yang berubah).
8. Status kunjungan menjadi "selesai".
9. Koreksi rekam medis dilakukan lewat addendum baru, bukan edit langsung record final.
10. Pendaftaran pasien baru terkunci setelah jam tutup klinik; pasien yang sudah terdaftar tetap dilayani sampai antrian habis.

## Fitur MVP
- Autentikasi (invite user baru & reset password via email, ganti password sendiri) + role-based access
- Manajemen pasien: registrasi baru, pencarian existing, warning duplikasi NIK
- Antrian real-time: generate nomor, tampilan real-time, panggil berikutnya, handling no-show, antrian prioritas
- Papan antrian publik (display board) — real-time, read-only, autentikasi terpisah dari sesi staff
- Kontrol jam operasional
- Rekam medis terstruktur: form per kunjungan, riwayat per pasien, koreksi via addendum
- Audit trail: log otomatis, tamper-evident
- Consent pengumpulan data pribadi
- Laporan/rekap kunjungan harian
- Dashboard admin: manajemen user, filter audit log, regenerate display token

## Eksplisit di Luar Scope
- Integrasi resmi SATUSEHAT
- Digital signature per-dokter dengan key custody penuh — disederhanakan jadi hash chaining untuk record integrity, bukan identitas personal dokter
- Pembayaran/billing, manajemen obat/apotek, hasil lab, klaim BPJS asli
- Multi-poli/departemen, rujukan antar-faskes, reassign shift dokter berhalangan mendadak, reliability/offline handling
