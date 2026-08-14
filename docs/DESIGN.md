# DESIGN.md — Modul RME & Antrian Klinik

Referensi desain untuk pengembangan frontend (Angular + Spartan/ui). Dibaca oleh developer manusia maupun Antigravity sebelum menulis/fetch komponen UI apapun — lihat `AGENTS.md` §8 untuk aturan wajib cek **Component Registry** (bagian akhir dokumen ini) sebelum membuat atau fetch komponen baru.

Dark mode: **di luar scope.** Semua token di bawah untuk light mode saja.

---

## 1. Prinsip Desain

1. **Accessible & Ethical sebagai fondasi** — WCAG AAA jadi target, bukan AA. Alasannya konkret, bukan generik: antrian prioritas proyek ini eksplisit menyasar lansia/difabel/ibu hamil, dan papan antrian dibaca dari jarak jauh di ruang tunggu. Kontras tinggi & teks besar itu kebutuhan fungsional.
2. **Dua surface, satu sistem token** — app staff (petugas/dokter/admin) dan papan antrian publik adalah dua masalah desain berbeda (lihat §11), tapi keduanya menarik dari token warna/semantik yang sama supaya pasien tetap mengenali makna warna di kedua tempat.
3. **Jangan andalkan warna sendirian** — status apapun yang punya makna operasional (status antrian, prioritas) wajib dibarengi ikon dan/atau label teks, bukan warna semata.
4. **Warna destruktif itu langka dan sengaja** — merah (`--color-destructive`) direservasi untuk error/aksi destruktif sungguhan. State rutin yang terlihat "negatif" tapi sebenarnya normal (klinik tutup di luar jam operasional, kunjungan `tidak_hadir` di penutupan hari) pakai warna netral, bukan merah — supaya merah tetap berarti "perlu perhatian serius" saat benar-benar muncul.
5. **Restrained, bukan playful** — tanpa animasi bouncy, tanpa gradient dekoratif, tanpa emoji sebagai ikon. Klinik kecil butuh terasa terpercaya, bukan trendi.

## 1.1 Dua Tier Visual — Public/Hero vs Internal/Content

- **Zona Hero** (Login, forgot-password, empty state, dashboard summary
  card): boleh pakai background foto/dekorasi, elemen visual lebih kaya —
  first-impression/low-frequency screen.
- **Zona Content** (tabel, form input, list antrian, rekam medis): tetap
  solid background (--color-background), TANPA foto/glass di belakang
  teks yang harus dibaca cepat — dipakai berkali-kali sehari (§9.3),
  kontras & keterbacaan tetap prioritas (§13). Refinement modern di sini
  lewat shadow/radius/tipografi, bukan dekorasi background.
- Prinsip #5 ("Restrained") berlaku default untuk Zona Content. Zona Hero
  dikecualikan secara eksplisit dari Prinsip #5, bukan pelanggaran diam-diam.

---

## 2. Warna

Palet dasar "Medical Clinic" — teal sebagai primary, hijau kesehatan sebagai accent.

| Token                            | Hex       | Pemakaian                                       |
| -------------------------------- | --------- | ----------------------------------------------- |
| `--color-primary`                | `#0891B2` | Aksi utama, link, brand                         |
| `--color-primary-foreground`     | `#FFFFFF` | Teks di atas primary                            |
| `--color-secondary`              | `#22D3EE` | Aksen sekunder, badge prioritas                 |
| `--color-secondary-foreground`   | `#0F172A` | Teks di atas secondary                          |
| `--color-accent`                 | `#16A34A` | Sukses, status positif, CTA sekunder            |
| `--color-accent-foreground`      | `#FFFFFF` | Teks di atas accent                             |
| `--color-background`             | `#F0FDFA` | Latar halaman                                   |
| `--color-foreground`             | `#134E4A` | Teks utama                                      |
| `--color-card`                   | `#FFFFFF` | Latar card/panel                                |
| `--color-card-foreground`        | `#134E4A` | Teks di dalam card                              |
| `--color-muted`                  | `#E8F1F6` | Latar elemen non-aktif/disabled                 |
| `--color-muted-foreground`       | `#64748B` | Teks sekunder, meta, state netral               |
| `--color-border`                 | `#CCFBF1` | Border, divider                                 |
| `--color-input`                  | `#CCFBF1` | Border input                                    |
| `--color-ring`                   | `#0891B2` | Focus ring (3–4px)                              |
| `--color-destructive`            | `#DC2626` | Error sungguhan, aksi destruktif                |
| `--color-destructive-foreground` | `#FFFFFF` | Teks di atas destructive                        |
| `--color-warning`                | `#D97706` | Menunggu/pending, perlu perhatian (bukan error) |
| `--color-warning-foreground`     | `#431407` | Teks di atas warning                            |

### Warna semantik status antrian

Prinsip #4 di atas diterapkan langsung di sini — perhatikan `tidak_hadir` sengaja **bukan** merah:

| Status        | Token warna                                          | Alasan                                                      |
| ------------- | ---------------------------------------------------- | ----------------------------------------------------------- |
| `menunggu`    | `--color-warning` (amber)                            | Pending, wajar, perlu dilihat                               |
| `dipanggil`   | `--color-primary` (teal, bold)                       | Momen aktif — "ini giliran Anda"                            |
| `selesai`     | `--color-accent` (hijau)                             | Positif, tuntas                                             |
| `tidak_hadir` | `--color-muted` bg / `--color-muted-foreground` teks | State terminal rutin, **bukan** error — netral, bukan merah |

Badge **prioritas** sengaja **tidak** memakai token status manapun di atas (biar tidak bentrok visual dengan `menunggu` yang sama-sama amber) — ikon flag + border outline `--color-secondary`, teks label `--color-foreground` (karena secondary `#22D3EE` terlalu terang di atas background terang, kontras <2:1, gagal WCAG AA). Ini juga penerapan langsung Prinsip #3: makna prioritas dibawa ikon+teks, warna cuma aksen.

---

## 3. Tipografi

| Peran     | Font               | Catatan                                                                                                                                                                       |
| --------- | ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Heading   | **Figtree**        | Weight 500–700                                                                                                                                                                |
| Body      | **Noto Sans**      | Weight 400–500, cakupan karakter luas (aman untuk nama & istilah medis Indonesia)                                                                                             |
| Monospace | **JetBrains Mono** | Khusus konten teknis: `requestId`, `hashEntry`, token mentah — dipilih karena membedakan `0`/`O` dan `1`/`l`/`I` dengan jelas, penting untuk string yang harus disalin akurat |

```
@import url('https://fonts.googleapis.com/css2?family=Figtree:wght@400;500;600;700&family=Noto+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap');
```

### Skala (app staff)

| Token         | Size | Pemakaian                               |
| ------------- | ---- | --------------------------------------- |
| `--text-xs`   | 12px | Caption/meta saja — **bukan** body text |
| `--text-sm`   | 14px | Label, helper text                      |
| `--text-base` | 16px | Body default — minimum untuk teks baca  |
| `--text-lg`   | 18px | Subheading                              |
| `--text-xl`   | 20px | Heading kecil                           |
| `--text-2xl`  | 24px | Heading section                         |
| `--text-3xl`  | 30px | Heading halaman                         |

Skala khusus papan antrian di §11 — jauh lebih besar, bukan ekstensi linear dari skala ini.

---

## 4. Spacing, Grid & Breakpoint

```
--space-1: 4px   --space-2: 8px   --space-3: 12px  --space-4: 16px
--space-6: 24px  --space-8: 32px  --space-12: 48px --space-16: 64px
```

**Breakpoint** (mobile-first, berlaku untuk app staff — papan antrian tidak ikut skema ini, lihat §11):

| Breakpoint | Min-width | Konteks       |
| ---------- | --------- | ------------- |
| Base       | 0         | Mobile        |
| `md`       | 768px     | Tablet        |
| `lg`       | 1024px    | Desktop kecil |
| `xl`       | 1440px    | Desktop lebar |

App staff **wajib fully responsive** — petugas/dokter/admin bisa akses dari desktop, tablet, atau mobile. Density bukan satu nilai tetap: tabel padat di `lg`+ otomatis collapse jadi list card (label:value per baris) di bawah `md` — pola standar, bukan komponen custom per halaman.

---

## 5. Shape & Elevation

```
--radius-sm: 4px    /* input, button */
--radius-md: 8px    /* card standar */
--radius-lg: 12px   /* modal, panel besar */
--radius-full: 9999px /* badge, pill */

--shadow-1: 0 1px 3px rgba(0,0,0,0.08)   /* elemen minimal */
--shadow-2: 0 4px 6px rgba(0,0,0,0.08)   /* card default */
--shadow-3: 0 10px 15px rgba(0,0,0,0.10) /* dropdown, popover */
--shadow-4: 0 20px 25px rgba(0,0,0,0.12) /* modal — level tertinggi, dipakai terbatas */
```

Shadow tetap ringan di semua tempat — hindari `--shadow-4` di luar modal/dialog.

---

## 6. Ikonografi

- Library: **Lucide**, via `ng-icons` — outline style, konsisten sepanjang app.
- **Tanpa emoji sebagai ikon**, tanpa pengecualian.
- Ukuran default 20px di dalam teks/tombol, 16px di badge kecil.
- Ikon dengan makna fungsional (bukan dekoratif) **wajib** `aria-label` atau teks pendamping.

---

## 7. Motion

```
--duration-fast: 150ms
--duration-base: 200ms
--duration-slow: 300ms
--easing-standard: cubic-bezier(0.4, 0, 0.2, 1)
```

- Semua transisi hormati `prefers-reduced-motion: reduce` — matikan transisi non-esensial.
- Tanpa animasi bouncy/spring. Restrained.
- Update realtime (WS) di papan antrian: fade halus saat data baru masuk (§11), bukan animasi mencolok.

---

## 8. Konten & Locale

- `LOCALE_ID` Angular: `id-ID`.
- Format tanggal tampilan: panjang Bahasa Indonesia ("12 Agustus 2026"). Input tetap native/ISO (`<input type="date">` atau setara).
- Microcopy formal, pakai "Anda" — konsisten dengan contoh pesan di `api-contract.md` (mis. "NIK sudah terdaftar").
- Timezone tampilan: Asia/Jakarta (konsisten dengan `TZ` backend).

---

## 9. Pola Komponen

### 9.1 Data sensitif — pola "Sensitive Value" (NIK & password)

Satu pola reusable untuk dua konteks: password di form (login, ganti password, reset password) dan NIK di tampilan (list/detail pasien).

- **Default: tersamar.** Password → dot standar browser. NIK → tampilkan **4 digit terakhir saja**, sisanya dot (`••••••••••••1234`) — sengaja **bukan** pola "4 depan + 4 belakang" ala kartu kredit, karena 6 digit pertama NIK memuat kode wilayah + tanggal lahir; menampilkannya berarti membocorkan tanggal lahir walau "tersamar". Menyembunyikan semua kecuali 4 digit terakhir tetap cukup untuk staff membedakan record secara visual tanpa membocorkan data personal.
- **Toggle: ikon mata** — Lucide `eye` (state tersamar, klik untuk buka) / `eye-off` (state terbuka, klik untuk sembunyikan lagi). Tanpa teks tombol — ikon saja, sesuai permintaan, cukup umum dikenali.
- Data lengkap tetap datang dari API apa adanya — masking ini murni presentasi, bukan perubahan response backend.
- Daftarkan sebagai komponen `SensitiveValue` di Component Registry — dipakai di form password DAN tampilan NIK, satu implementasi dua mode (`input` / `display`).

### 9.2 Reveal-once secret (displayToken, inviteLink)

Untuk token/link yang API sengaja cuma balikin sekali (`displayToken` saat regenerate, `inviteLink` saat buat user):

- Box `font-mono` (`--font-mono`), read-only, latar `--color-muted`.
- Tombol copy (ikon Lucide `copy`) di sebelah kanan, feedback singkat "Disalin!" setelah diklik.
- Teks peringatan di bawah, warna `--color-warning-foreground`: "Token ini hanya ditampilkan sekali — simpan sekarang."
- Komponen `RevealOnceSecret`, reusable untuk kedua kasus.

### 9.3 Tabel & pagination

- Pagination **bernomor** (Prev / 1 2 3 / Next), sesuai `page`/`limit` di backend — bukan infinite scroll.
- Sticky header, hover row halus, **tanpa** zebra-stripe.
- Row density nyaman (bukan dashboard super-padat) — tabel ini dipakai berkali-kali sehari oleh petugas/dokter, keterbacaan lebih penting dari memaksimalkan baris per layar.
- Di bawah breakpoint `md`: collapse ke list card, tiap card = label:value per field penting (lihat §4).
- Nomor antrian di tabel/list staff: angka biasa. Di papan antrian: zero-padded 3 digit (`007`) — lihat §11.

### 9.4 Status & badge

- `StatusBadge` — pill `--radius-full`, warna sesuai tabel §2, **selalu** disertai label teks (bukan warna+titik doang).
- `PriorityBadge` — pill outline, ikon `flag` & border memakai `--color-secondary`, TEKS label memakai `--color-foreground` (bukan secondary — secondary terlalu terang untuk teks langsung di atas background terang, kontras <2:1, gagal WCAG AA), independen dari `StatusBadge` (bisa tampil bersamaan).
- `ClinicStatusIndicator` — badge "Buka"/"Tutup" di header app staff. "Tutup" **bukan** merah (state normal akhir hari) — pakai `--color-muted-foreground`. Saat `Tutup`, tombol "Daftar ke Antrian" auto-disable + tooltip alasan, jangan tunggu sampai submit gagal (`KLINIK_TUTUP`) baru user tahu.
- Status user admin: badge "Aktif" (hijau) vs "Menunggu Aktivasi" (netral) berdasarkan `password_hash` null atau tidak — tampilkan tombol "Kirim Ulang Undangan" inline di baris user berstatus pending.

### 9.5 Toast & error

- Posisi **top-center**, konsisten di semua breakpoint.
- `role="status" aria-live="polite"` untuk info/sukses, `aria-live="assertive"` untuk error — pembeda ARIA ini penting, jangan disamakan.
- Isi toast pakai `message` yang sudah dikurasi backend per `code` (aman ditampilkan langsung, lihat `api-contract.md`) — jangan tampilkan raw error.
- Error 409 (optimistic-lock pasien, race addendum): toast + tombol aksi "Muat ulang data terbaru", bukan cuma pesan pasif.
- 401 → redirect ke `/login`. 403 → halaman/state "Anda tidak punya akses". 404 → state kosong kontekstual, bukan halaman generik.

### 9.6 Form

- Reactive Forms wajib untuk semua form (konsisten `AGENTS.md` §8), `FormArray` untuk `diagnosis[]`/`tindakan[]`.
- Pola **repeatable row group**: tombol "+ Tambah" di bawah list, ikon hapus (`x` atau `trash-2`) per baris, minimal 1 baris selalu tersisa (tidak bisa dihapus sampai kosong total).
- Error validasi muncul setelah field `touched`, posisinya tepat di bawah field — bukan dikumpulkan di atas form.
- Consent (pengumpulan data pribadi): checkbox + teks kebijakan singkat yang bisa di-expand ("Baca selengkapnya"), bukan checkbox polos tanpa konteks.
- Alasan addendum (`alasanAddendum`): field wajib, ditonjolkan visual — border kiri `--color-secondary` + label "Koreksi" pada card rekam medis yang berstatus addendum, beda jelas dari rekam medis asli.

### 9.7 Dialog konfirmasi

- Dipakai untuk aksi yang berdampak nyata dan sulit di-undo: regenerate display token (mematikan token lama seketika), submit rekam medis final, dsb.
- Isi dialog menyebutkan **konsekuensi konkret** (bukan "Anda yakin?" generik) — mis. "Token lama akan langsung tidak berlaku. Papan antrian yang sedang aktif akan terputus sampai token baru dipasang."

### 9.8 Empty & loading state

- Loading: skeleton untuk list/table/card, bukan spinner polos di tengah layar kosong.
- Empty state: ilustrasi/ikon + teks kontekstual sesuai situasi ("Belum ada pasien terdaftar hari ini" ≠ "Tidak ada hasil untuk pencarian ini") — bukan tabel kosong tanpa penjelasan.

### 9.9 Audit log diff viewer

- `beforeData`/`afterData` ditampilkan sebagai **diff**: field yang berubah disorot (nilai lama dicoret/muted, nilai baru ditonjolkan), bukan dua blok JSON mentah bersebelahan.
- Saat `tabelTarget = rekam_medis`: banner kontekstual halus di atas detail ("Anda sedang melihat isi rekam medis klinis") — penguatan visual dari keputusan keamanan yang sudah didesain di `api-contract.md` (akses admin ke rekam medis sengaja butuh langkah sadar), bukan fitur baru.

### 9.10 Indikator koneksi realtime

- Badge kecil "Terhubung" / "Menyambung ulang…" — krusial khusus di papan antrian karena tidak ada staff yang berjaga di sana; kalau WS putus diam-diam, data bisa basi tanpa siapa pun sadar.
- Reconnect otomatis tetap refetch REST sekali (sesuai `TDD.md`) — indikator ini murni sinyal visual, bukan pengganti logic reconnect.

---

## 10. Fetch Spartan primitive baru — checklist

Berlaku tiap kali komponen Spartan primitif baru di-fetch lewat CLI/MCP (lihat juga aturan di `AGENTS.md` §8):

1. Cek Component Registry (§12) dulu — primitive ini sudah pernah di-fetch & disesuaikan sebelumnya? Kalau ya, **jangan fetch ulang**, pakai yang sudah ada.
2. Kalau benar-benar baru: setelah ter-copy, verifikasi dia memakai class/token semantik (`bg-primary`, `text-foreground`, `rounded-[--radius-md]`, dst) yang match ke variable §2–§5 di atas — **bukan** warna/radius default hardcoded dari template.
3. Kalau template default Spartan ternyata tidak memakai token semantik sama sekali — **STOP**, laporkan ke user, itu keputusan arsitektur yang butuh konfirmasi eksplisit.
4. Setelah sesuai, tambahkan barisnya ke Component Registry (§12) saat itu juga.

---

## 11. Papan Antrian — Override Terpisah

Surface publik, read-only, dilihat dari jarak beberapa meter di ruang tunggu. **Tidak** mengikuti breakpoint §4 (bukan konteks "responsive" dalam arti device yang di-resize) — asumsikan layar/TV landscape tetap, viewed dari jarak ruang tunggu klinik kecil. Kalau nanti ternyata perlu robust di berbagai ukuran layar fisik, ini perlu diverifikasi ulang terhadap hardware yang benar-benar dipakai.

Token tambahan, aktif hanya di scope `.papan-antrian`:

```css
.papan-antrian {
  --board-text-label: 2rem; /* 32px — label section, mis. "Sedang Dipanggil" */
  --board-text-status: 1.75rem; /* 28px */
  --board-text-row: 2.5rem; /* 40px — tiap baris di daftar menunggu */
  --board-text-number: 7.5rem; /* 120px — nomor antrian aktif, elemen dominan */
  --space-scale: 1.5; /* spacing dasar dikalikan 1.5x dari §4 */
}
```

- Warna tetap dari palet §2 (konsistensi makna lintas surface), tapi **penggunaan warna diminimalkan** — hanya untuk status, bukan dekorasi. Latar tetap `--color-background`, bukan warna-warni.
- Nomor antrian: zero-padded 3 digit (`007`) — rapi & konsisten secara visual dari jarak jauh, murni kosmetik (tidak mengubah integer di backend).
- Tanpa interaksi (read-only) — tanpa hover state, tanpa elemen yang mengundang klik.
- Indikator koneksi realtime (§9.10) wajib tampil di sini.
- **Belum diputuskan (di luar sesi ini):** chime/audio saat nomor dipanggil. Relevan untuk aksesibilitas (target prioritas termasuk lansia/difabel), tapi tidak ada di Fitur MVP PRD — perlu keputusan scope terpisah sebelum masuk dokumen ini.

---

## 12. Component Registry

> **Wajib dicek** sebelum membuat komponen baru atau fetch primitive Spartan baru (lihat `AGENTS.md` §8 dan §10 di atas). Kolom **Status** diupdate Antigravity seiring implementasi — `Direncanakan` → `Selesai`. Registry ini basi kalau tidak dijaga; itu justru bagian dari poin kenapa dia wajib dicek, bukan sekadar didokumentasikan.

| Komponen                                        | Tipe      | Fungsi                                                 | Membungkus                                  | Dipakai di                          | Status                                          |
| ----------------------------------------------- | --------- | ------------------------------------------------------ | ------------------------------------------- | ----------------------------------- | ----------------------------------------------- |
| `SensitiveValue`                                | Composed  | Mask/reveal NIK & password, toggle ikon mata (§9.1)    | Spartan Input, Button                       | Auth, Pasien                        | Selesai                                         |
| `ToastNotification`                             | Composed  | Toast top-center, ARIA live region, dismissible (§9.5) | — (custom)                                  | Auth, Lintas Fitur                  | Selesai                                         |
| `PasienForm`                                    | Composed  | Form registrasi pasien, consent, validasi NIK format, warning duplikasi NIK pre-submission (non-blocking) | ToastNotification, native select/textarea/checkbox | Pasien (route /pasien/baru) | Selesai (Tahap 1) |
| `RevealOnceSecret`                              | Composed  | Tampilkan token/link sekali + copy (§9.2)              | Spartan Input (readonly), Button            | Admin (invite, display-token)       | Direncanakan                                    |
| `DataTable`                                     | Composed  | Tabel + pagination bernomor + collapse ke card (§9.3)  | Spartan Table                               | Pasien, Antrian, Admin              | Direncanakan                                    |
| `StatusBadge`                                   | Composed  | Badge status antrian (§9.4)                            | Spartan Badge                               | Antrian                             | Direncanakan                                    |
| `PriorityBadge`                                 | Composed  | Badge prioritas, ikon+label (§9.4)                     | Spartan Badge                               | Antrian, Pasien                     | Direncanakan                                    |
| `ClinicStatusIndicator`                         | Composed  | Badge buka/tutup klinik (§9.4)                         | Spartan Badge                               | App shell/header staff              | Selesai                                         |
| `ConfirmDialog`                                 | Composed  | Dialog konfirmasi aksi berdampak (§9.7)                | Spartan Dialog                              | Admin, Rekam Medis                  | Direncanakan                                    |
| `DiagnosisTindakanFormArray`                    | Composed  | Repeatable row group add/remove (§9.6)                 | Spartan Input, Select, Button + `FormArray` | Rekam Medis                         | Direncanakan                                    |
| `AuditDiffViewer`                               | Composed  | Diff view before/after JSON (§9.9)                     | — (custom)                                  | Admin > Audit Log                   | Direncanakan                                    |
| `ConnectionStatusIndicator`                     | Composed  | Indikator status WS (§9.10)                            | — (custom)                                  | Papan Antrian, widget antrian staff | Direncanakan                                    |
| Button / Badge / Avatar / Dropdown-Menu / Sheet | Primitive | Primitive dasar Spartan (diverifikasi §10)             | —                                           | Lintas fitur                        | Selesai                                         |

---

## 13. Checklist Aksesibilitas (ringkas)

- [ ] Kontras teks minimum 4.5:1, target 7:1 untuk teks penting/papan antrian
- [ ] Focus ring 3–4px terlihat jelas di semua elemen interaktif
- [ ] Touch target minimum 44×44px
- [ ] Semua ikon fungsional punya `aria-label` atau teks pendamping
- [ ] Tanpa indikator warna-saja (status selalu + ikon/teks)
- [ ] `prefers-reduced-motion` dihormati
- [ ] Toast pakai `aria-live` yang benar (`polite` vs `assertive`)
- [ ] Fokus otomatis pindah ke heading utama tiap ganti rute

---

## 14. Di Luar Scope

- Dark mode
- Chime/audio papan antrian (§11 — keputusan scope tertunda, bukan keputusan desain)
