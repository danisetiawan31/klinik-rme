# Done Log — Frontend

## Scaffolding — Tahap 1-6

- Angular CLI v21 + Tailwind v4 + Vitest native. Struktur `core/features/shared`, `environment.ts`, proxy `proxy.conf.json`, `ErrorEnvelope`, `auth.interceptor` (skip 401 redirect untuk display-token & auth endpoints).
- Vitest native via `@angular/build:unit-test` — tidak pakai Karma/Analog.

## Core Shell — Tahap 1-3

- `staffAuthResolver` di-await di root route, `AuthService` Signal-based, `roleGuard`, `authInterceptor` (`withCredentials`, parse `ErrorEnvelope`).
- `ShellComponent`: sidebar collapsible + mobile drawer, `LandingComponent` shortcut per-role, timezone `Asia/Jakarta` di-anchor global (`LOCALE_ID: id-ID`, `date.utils.ts`).
- `HlmSheetContent` DI diubah ke `optional: true` untuk mencegah `NG0201` di testing context.
- **Verifikasi**: 14 files, 46 tests PASS.

## Auth Recovery Pages

- `forgotPassword` + `resetPassword` di `AuthService`, `ForgotPasswordComponent` (card-swap 200 generik, cegah enumeration), `SetPasswordComponent` (3 card state: token error / form / sukses).
- **Verifikasi**: 17 files, 64 tests PASS.

---

## RealtimeService

- Native WebSocket wrapper, Signals `connectionStatus` + `lastUpdateAt`. Exponential backoff ±20% jitter, cap 30s. Dual-auth via cookie (staff) / `displayToken` (papan antrian). Proxy `/ws` dengan `"ws": true`.
- **Verifikasi**: 16 files, 59 tests PASS.

---

## Profil / Account Settings

- `changePassword()` (`PATCH /auth/me/password`), `ProfilComponent`: info akun read-only + form ubah password. Error `INVALID_PASSWORD` → inline; error teknis → Sonner toast. Routing `/profil` + link di shell dropdown.
- **Verifikasi**: 18 files, 70 tests PASS.

---

## Pasien — Tahap 1-3

- CRUD pasien: registrasi (`/pasien/baru`, NIK validator, warning duplikat non-blocking), search (debounce 300ms nama / auto-trigger 16-digit NIK, `PaginationComponent` via `X-Total-Count`), detail + riwayat kunjungan, edit biodata (409 Optimistic Lock hybrid UX).
- `nikFormatValidator()` di-extract ke `pasien.validators.ts`. Feedback sukses edit via `history.state.successMessage`.
- **Verifikasi**: 24 files, 111 tests PASS.

---

## Migrasi Token CSS & Tailwind v4 `@theme`

- Konsolidasi `:root` alias token Spartan, daftarkan `--input`, `--warning`, radius, shadow, `font-heading` ke `@theme inline`. Migrasi bracket class `[var(--...)]` seluruh modul ke utility standar. Audit regex hex/rgb: **0 match**.
- **Verifikasi**: 24 files, 111 tests + build produksi PASS.

---

## Pemisahan File Template HTML (`templateUrl`)

- Seluruh komponen frontend (Pasien, Auth, Profil, Shell, Shared, Antrian, Admin) dipisah ke `.component.html` terpisah.
- **Verifikasi**: 24 files, 111 tests + build produksi PASS.

---

## Antrian (Staff-Facing) — Tahap 1-3

- **Tahap 1**: `StatusBadge` + `PriorityBadge`, dashboard realtime (sort `isPriority DESC, skipCount ASC, nomorAntrian ASC`, dual-trigger WS + `DestroyRef`).
- **Tahap 2**: `AntrianService` (panggilBerikutnya, lewati, tidakHadir), RBAC per tombol, double-submit guard, 409 auto-refetch.
- **Tahap 3**: `POST /kunjungan` dari `PasienDetailComponent`, modal prioritas (validasi `priorityReason` wajib jika prioritas aktif), sukses toast + auto-refresh riwayat.
- **Verifikasi**: 28 files, 144 tests PASS.

---

## Laporan Harian

- `LaporanService`, filter tanggal default WIB, 3 kartu metrik, loading state, error toast. Route `roleGuard('petugas', 'dokter', 'admin')`.
- **Verifikasi**: 30 files, 152 tests PASS.

---

## Addendum — Spartan UI Modernization

- **Sidebar-Inset**: Struktur `sidebar-inset` pada Shell (collapsible icon mode, inset layout, brand icon, avatar profil bottom).
- **Sonner Toast**: `ngx-sonner` + `HlmToaster` global, seluruh modul migrasikan dari toast lokal ke `toast.success/error/info()`.
- **Verifikasi**: 30 files, 154 tests PASS.

---

## Addendum — Spartan Primitives (Dialog, Card, Button, Table, Skeleton, Form Controls, Alert, Empty, Icon)

Seluruh migrasi Spartan primitive selesai sekaligus dilakukan dan diverifikasi per batch:

| Primitive | Cakupan | Verifikasi |
|-----------|---------|------------|
| **Dialog** (`shared/ui/dialog/`) | Modal tidak hadir (antrian) + modal antrian (pasien detail) → `<hlm-dialog>`; test assertion pindah ke `document.body` | 154 tests PASS |
| **Card** (`shared/ui/card/`) | Semua `div.bg-card` di antrian, laporan, auth, profil, pasien → `<hlm-card>` | 154 tests PASS |
| **Button** (`hlmBtn`) | Seluruh `kl-btn-*` dan inline Tailwind button → `button[hlmBtn]` varian `default/outline/secondary/destructive`. Anti-spam via `data-disabled`. `kl-btn` sisa: **0** | 154 tests PASS |
| **Table** (`shared/ui/table/`) | Tabel antrian + pasien list + riwayat kunjungan → `hlmTable` | 154 tests PASS |
| **Skeleton** (`shared/ui/skeleton/`) | Loading spinner/teks statis → shimmer placeholder Zero CLS (antrian, pasien list/detail, laporan) | 154 tests PASS |
| **Input / Label / Textarea** (`shared/ui/{input,label,textarea}/`) | Seluruh form (auth, profil, pasien, laporan) → `hlmInput`, `hlmLabel`, `hlmTextarea`. Hapus legacy `.kl-input/.kl-pw-wrap/.kl-pw-toggle`. | 154 tests PASS |
| **Alert** (`shared/ui/alert/`) | NIK duplikat + 409 conflict → `<hlm-alert variant="warning">`; error modal prioritas → `variant="destructive"` | 154 tests PASS |
| **ConnectionStatusIndicator** (`shared/components/`) | Inline markup status realtime antrian → `<app-connection-status-indicator>` (3 varian: connected/reconnecting/disconnected, a11y `role="status"`) | 159 tests PASS |
| **Empty** (`shared/ui/empty/`) | Empty state antrian, pasien list/detail → `<hlm-empty>` | 159 tests PASS |
| **Icon** (`shared/ui/icon/`) | 100% raw SVG diganti `<ng-icon hlm>` + `@ng-icons/lucide`. Fix `HlmIconDirective` pemetaan token ukuran → CSS valid (`xs`=`0.875rem` … `xl`=`2rem`) pada `--ng-icon__size`. Eliminasi `[innerHTML]="... \| safeHtml"`. Audit `<svg`: **0 match**. | 159 tests PASS |

**`docs/DESIGN.md` Component Registry**: Semua primitive di atas berstatus **Selesai**.

---

## Addendum — Modernisasi Beranda Dokter & Staff (Hero Banner, Eye-Catching Cards & Operational Widgets)

- **Hero Banner Dokter/Staff**: Greeting personal, pill live time & date WIB (`formatJakartaDayDate`, interval 30s), backdrop ilustrasi dokter, dan 4 mini metrics strip terintegrasi data live `AntrianService` (Pasien Hari Ini, Antrian Menunggu, Selesai Dilayani, Pasien Prioritas).
- **Eye-Catching 4 Module Cards**: Layout kartu dengan badge icon melingkar (`size-13 rounded-full`), gradient latar khusus per modul (Cyan, Blue, Purple, Emerald), link `Buka Modul →`, dan watermark nomor semantik (`01`–`04`).
- **2 Widget Operasional Bawah**:
  - Kolom Kiri: Live Antrian Pasien Hari Ini (top 4 antrian dengan `#nomorAntrian`, nama, status/priority badge, CTA `Lihat Semua →`, dan empty state).
  - Kolom Kanan: Ringkasan Operasional (2x2 grid metrik live + progress bar persentase penyelesaian antrian klinik).

**File**: `public/images/doctor_banner.jpg` (baru), `date.utils.ts`, `landing.*`, `styles.css`.
**Verifikasi**: `npm test -- --run` → **31 files, 160 tests PASS** (exit 0).

---

## Addendum — Redesign Visual Hero Banner & Jadwal Operasional (Landing Polish Tahap 1)

- **Hero Banner Glassmorphism**: Desain single-card glassmorphism (`bg-card/70 backdrop-blur-xl border border-border/80 shadow-sm`), ambient glow mesh gradient, dan ilustrasi dokter panoramik widescreen (`hero_doctor_wide1.jpg`).
- **Jadwal & Status Operasional**: Live indicator status buka/tutup klinik dengan pulsing emerald dot dan jadwal operasional (`KlinikService`) di bawah baris tanggal & jam.
- **Perbaikan CSS Cascade Layer**: Eliminasi unlayered margin/padding reset pada `styles.css` yang menimpa utility class Tailwind v4; stabilisasi layout menggunakan `flex flex-col gap-*`.
- **Status & Priority Badge**: Standardisasi badge ke `text-xs px-2.5 py-0.5 rounded-md`.

**File**: `styles.css`, `landing.*`, `priority-badge.*`, `status-badge.*`, `hero_doctor_wide1.jpg`.
**Verifikasi**: `npm test -- --run` → **31 files, 163 tests PASS** (exit 0).

---

## Addendum — Efisiensi Pelayanan & ApexCharts Radial Gauge (Landing Polish Tahap 2)

- **ApexCharts RadialBar Gauge**: Visualisasi *completion rate* pelayanan dengan gauge melingkar modern (`ng-apexcharts` + `apexcharts`), gradient stroke (`teal` ke `emerald`), track latar transparan semantik, dan label persentase dinamis.
- **Analitik 100% Non-Redundan & Berbasis Data Riil**:
  1. *Tingkat Kehadiran (Attendance Rate)*: Rasio kehadiran pasien terlayani vs pasien no-show/batal (`attendanceRate%` dari `(totalKunjungan - totalTidakHadir) / totalKunjungan`).
  2. *Pasien Tidak Hadir (No-Show)*: Metrik resmi pasien batal/dilewati dari `LaporanHarian.totalTidakHadir`.
  3. *Badge Tren Komparasi*: Komparasi volume kunjungan terhadap hari kemarin (`+X%` / `-X%` / `– Data Awal`).
- **Date Utility**: Helper `getJakartaYesterdayISODate()` untuk tanggal kemarin dalam timezone `Asia/Jakarta`.
- **Refinement Densitas UI/UX (High Density & Ergonomics)**:
  1. *Akses Cepat Modul*: Diperkecil menjadi `p-4.5 rounded-2xl` dengan icon `size-10` dan watermark `text-xl` untuk menghemat ruang vertikal.
  2. *Antrian Pasien Hari Ini*: Item baris dirampingkan ke `py-2.5 px-3.5 rounded-xl` dengan nomor antrian badge `size-8 text-xs`, nama pasien `text-sm font-semibold`, dan container `p-5 sm:p-6 rounded-2xl` yang simetris dengan kolom kanan.
  3. *Deduplikasi Status Buka/Tutup*: Menghilangkan duplikasi status klinik buka/tutup di Hero Banner dan menggabungkan meta pills tanggal, waktu live, dan jam layanan (`Jam Layanan: Senin – Sabtu · 08:00 – 21:00 WIB`) dalam satu baris bersih tanpa dead code.

**File**: `landing.*`, `date.utils.*`, `package.json`.
**Verifikasi**: `npm test -- --run` → **31 files, 166 tests PASS** (exit 0) & GitNexus `impact`/`detect_changes` (risk: LOW).

---

## Rekam Medis — Tahap 1 (Types & Service Layer)

- `rekam-medis.types.ts`: Interface domain `RekamMedis`, `DiagnosisItem`, `TindakanItem`, `RiwayatRekamMedisItem`, `CreateRekamMedisDto`, `CreateAddendumDto` sesuai kontrak `docs/api-contract.md`.
- `RekamMedisService`: Client HTTP Angular untuk `getRekamMedisByKunjungan`, `createRekamMedis`, `createAddendum`, dan `getRiwayatByPasien`.
- **Verifikasi**: `npm test -- --run` → **32 files, 171 tests PASS** (exit 0).

---

## Rekam Medis — Tahap 2 (Form Pemeriksaan Pasien)

- `RekamMedisFormComponent`: Formulir klinis RME terstruktur berbasis Reactive Forms dengan `FormArray` untuk diagnosis (min. 1 baris, uppercase ICD-10) dan tindakan/resep (`jenis: 'tindakan' | 'resep'`).
- Panel identitas konteks pasien dengan masking NIK (`app-sensitive-value`), gender, tanggal lahir, dan collapsible riwayat medis terdahulu (`GET /pasien/:id/riwayat`).
- Penanganan submit POST `/kunjungan/:id/rekam-medis`, feedback via Sonner toast, dan penanganan error conflict 409 `REKAM_MEDIS_ALREADY_EXISTS`.
- `AntrianService.getKunjungan(id)`: Helper fetch `KunjunganDetail` (`GET /kunjungan/:id`).
- **Verifikasi**: `npm test -- --run` → **33 files, 178 tests PASS** (exit 0).

---

## Rekam Medis — Tahap 3 (Tampilan Versi Terkini & Modal Addendum)

- `RekamMedisDetailComponent`: Tampilan leaf record terkini (`GET /kunjungan/:id/rekam-medis`), banner khusus bila record merupakan hasil addendum (`isAddendum === true`), ringkasan SOAP, tabel diagnosis & tindakan terstruktur.
- Modal Addendum Spartan Dialog (`HlmDialog`): Form koreksi medis dengan validasi wajib `alasanAddendum`, pre-fill data lama, modifikasi dinamis diagnosis & tindakan FormArray, penanganan submit POST `/rekam-medis/:id/addendum`, dan penanganan error 409 `ADDENDUM_CONFLICT` (auto-reload leaf data).
- **Verifikasi**: `npm test -- --run` → **34 files, 184 tests PASS** (exit 0).

---

## Rekam Medis — Tahap 4 (Routing & Integrasi End-to-End)

- `rekam-medis.routes.ts`: Rute form pemeriksaan (`/pemeriksaan/:kunjunganId`) dan detail leaf query (`/kunjungan/:kunjunganId`).
- `app.routes.ts`: Pendaftaran rute child `/rekam-medis` di bawah shell layout dengan proteksi `canActivate: [roleGuard('dokter')]`.
- `AntrianDashboardComponent`: Penambahan aksi "Periksa" (`/rekam-medis/pemeriksaan/:id`) untuk status `dipanggil` dan "Lihat RME" untuk status `selesai` (khusus role dokter).
- `PasienDetailComponent`: Integrasi tombol "Lihat RME" pada tabel riwayat kunjungan selesai untuk role dokter.
- `docs/DESIGN.md`: Update Component Registry (`DiagnosisTindakanFormArray`, `RekamMedisForm`, `RekamMedisDetail`).
- `workflow/backlog.md`: Update item 17 ke status `[x]` (Selesai Penuh).
- **Verifikasi**: `npm test -- --run` → **35 files, 185 tests PASS** (exit 0).

---

## Papan Antrian (Publik) — Tahap 1 (Service Layer & Display Token Support)

- `AntrianService.getAntrian(klinikId, displayToken?)`: Penambahan parameter opsional `displayToken` yang secara otomatis menyematkan header `X-Display-Token` untuk autentikasi surface publik papan antrian ke endpoint `GET /api/v1/klinik/:id/antrian`.
- `RealtimeService`: Memverifikasi integrasi koneksi WebSocket via query param `GET /ws?klinikId=:id&displayToken=:token` (karena browser WebSocket API tidak mendukung custom HTTP header).
- **Verifikasi**: `npm test -- --run` → **35 files, 186 tests PASS** (exit 0).

---

## Papan Antrian (Publik) — Tahap 2 (Komponen Papan Antrian & Layout Jarak Jauh TV)

- `PapanAntrianComponent`: Komponen antarmuka publik ruang tunggu klinik (`/papan-antrian`), didesain khusus untuk jarak pandang jauh TV/monitor landscape sesuai `docs/DESIGN.md §11`.
- Section dominan "Sedang Dipanggil": Nomor antrian 3-digit zero-padded (`007`), status animasi panggilan aktif, badge prioritas, dan header jam realtime (WIB).
- Section "Daftar Menunggu": Grid nomor antrian menunggu dengan badge prioritas dan counter antrian.
- Token resolution & storage: Otomatis membaca token dari query parameter (`?token=...`) atau `localStorage`, dengan modal konfigurasi token jika token tidak ditemukan atau tidak valid (401).
- Dual-trigger reaktivitas: Subscribe WebSocket `queue_updated` + refetch on reconnect + fallback interval timer 30 detik.
- **Verifikasi**: `npm test -- --run` → **36 files, 192 tests PASS** (exit 0).

---

## Papan Antrian (Publik) — Tahap 3 (Routing Publik & Integrasi End-to-End)

- `app.routes.ts`: Pendaftaran rute publik `/papan-antrian` di root (di luar shell staff layout, tanpa `staffAuthResolver` dan tanpa `roleGuard`).
- `app.spec.ts`: Unit test konfigurasi rute `/papan-antrian` bebas guard staff.
- `docs/DESIGN.md`: Update Component Registry (`PapanAntrian`).
- `workflow/backlog.md`: Update Item 16 ke status `[x]` (Selesai Penuh).
- **Verifikasi**: `npm test -- --run` → **36 files, 193 tests PASS** (exit 0).

---

## Admin Dashboard — Tahap 1 (Types & Service Layer)

- `admin.types.ts`: Definisi antarmuka DTO domain admin (`AdminUser`, `AdminUserListResult`, `CreateAdminUserRequest`, `CreateAdminUserResponse`, `UpdateAdminUserRequest`, `UpdateUserRolesRequest`, `AuditLogSummary`, `AuditLogDetail`, `AuditLogFilterParams`, `RegenerateDisplayTokenResponse`) sesuai kontrak `docs/api-contract.md`.
- `AdminService`: Client HTTP Angular untuk interaksi endpoint admin: `getUsers`, `createUser`, `resendInvite`, `updateUser`, `updateUserRoles`, `getAuditLogs`, `getAuditLogDetail`, dan `regenerateDisplayToken` (termasuk ekstraksi header `X-Total-Count` untuk pagination).
- **Verifikasi**: `npm test -- --run` → **37 files, 202 tests PASS** (exit 0).

---

## Admin Dashboard — Tahap 2 (Manajemen Pengguna, Invite & Reveal Once, Edit Biodata, Roles & Resend)

- `RevealOnceSecretComponent` (`shared/components/reveal-once-secret/`): Komponen composed reusable untuk menampilkan rahasia sekali-lihat (invite link / display token) dengan tombol copy 1-klik, feedback tersalin, dan banner peringatan keamanan.
- `AdminUsersComponent` (`features/admin/components/admin-users/`): Manajemen lengkap akun pengguna staf:
  - Tabel daftar user dengan pagination numerik (`PaginationComponent`).
  - Badge visual roles per akun (`petugas`, `dokter`, `admin`).
  - Modal Form Invite User Baru (`POST /admin/users`) dengan proteksi validasi mutual exclusivity `dokter` vs `admin`.
  - Integrasi modal `RevealOnceSecret` untuk menampilkan `inviteLink` setelah pembuatan akun berhasil.
  - Modal Form Edit Biodata User (`PATCH /admin/users/:id`).
  - Modal Form Kelola Peran (`PATCH /admin/users/:id/roles`) dengan validasi eksklusivitas peran `dokter` vs `admin`.
  - Aksi Kirim Ulang Tautan Undangan (`POST /admin/users/:id/resend-invite`).
- `AdminDashboardComponent`: Kerangka navigasi tab dashboard admin (`users`, `audit-log`, `klinik`).
- `docs/DESIGN.md`: Update Component Registry (`RevealOnceSecret`).
- **Verifikasi**: `npm test -- --run` → **40 files, 213 tests PASS** (exit 0).

---

## Admin Dashboard — Tahap 3 (Jejak Audit, Filter Bar & Visual Diff Viewer)

- `AuditDiffViewerComponent` (`features/admin/components/audit-diff-viewer/`): Komponen composed penampil perbandingan perubahan data (before/after field diff) dengan penyorotan warna semantik, visualisasi nilai terhapus (strikethrough/muted), nilai ditambah/diubah (green/accent), peringatan banner akses data klinis rekam medis, dan bukti tamper-evident SHA-256 hash chain per `docs/DESIGN.md §9.9`.
- `AdminAuditLogComponent` (`features/admin/components/admin-audit-log/`): Modul penampil riwayat audit:
  - Filter bar multi-kriteria: `tabelTarget` (`pasien`, `rekam_medis`), `recordId`, `actorId`.
  - Tabel log ringkas dengan pagination numerik (`X-Total-Count`) dan formatting tanggal jam lokal WIB (`Asia/Jakarta`).
  - Integrasi modal pop-up `AuditDiffViewerComponent` saat aksi "Lihat Diff" diklik.
- `AdminDashboardComponent`: Integrasi rendering `AdminAuditLogComponent` pada tab `audit-log`.
- `docs/DESIGN.md`: Update Component Registry (`AuditDiffViewer`).
- **Verifikasi**: `npm test -- --run` → **42 files, 220 tests PASS** (exit 0).

---

## Admin Dashboard — Tahap 4 (Pengaturan Klinik & Display Token Antrian)

- `AdminKlinikComponent` (`features/admin/components/admin-klinik/`): Modul manajemen operasional klinik dan kunci otentikasi layar ruang tunggu:
  - Profil operasional klinik: jam buka/tutup (WIB), status operasional (`ClinicStatusIndicatorComponent`), dan aturan penguncian antrian.
  - Kartu manajemen display token antrian publik dengan informasi enkripsi hash SHA-256.
  - Dialog konfirmasi bahaya dengan penjelasan dampak nyata (*"Token lama akan langsung tidak berlaku seketika, dan papan antrian ruang tunggu akan terputus sampai token baru dipasang"* per `docs/DESIGN.md §9.7`).
  - Integrasi modal `RevealOnceSecretComponent` untuk menampilkan display token mentah baru setelah regenerate berhasil.
  - Kartu panduan pemasangan Smart TV / mini PC ruang tunggu.
- `AdminDashboardComponent`: Integrasi rendering `AdminKlinikComponent` pada tab `klinik`.
- **Verifikasi**: `npm test -- --run` → **43 files, 224 tests PASS** (exit 0).

---

---

## Addendum — Rekam Medis Workspace & Perbaikan Navigasi Subtab Admin

- `RekamMedisListComponent` (`features/rekam-medis/components/rekam-medis-list/`):
  - Dashboard klinis dokter di rute `/rekam-medis` yang berdiri sendiri (menggantikan redirect sebelumnya ke `/antrian`).
  - Hero card pasien yang sedang dipanggil (`status: dipanggil`) dengan tombol langsung **"Mulai Pengisian SOAP"** (`/rekam-medis/pemeriksaan/:id`).
  - Input pencarian cepat riwayat rekam medis pasien (berdasarkan nama atau NIK 16-digit).
  - Tabel filter multi-tab kunjungan hari ini (`Sedang Dipanggil`, `Selesai Diperiksa`, `Semua Kunjungan`) dengan tombol aksi pemeriksaan SOAP dan addendum koreksi.
- `AdminDashboardComponent` (`features/admin/admin-dashboard.component.ts`):
  - Menambahkan *reactive parameter subscription* ke `paramMap` dan `NavigationEnd` agar pergantian antar-subtab (`/admin/users`, `/admin/audit-log`, `/admin/pengaturan`) dari sidebar langsung memperbarui tampilan tab tanpa macet karena *Angular component reuse*.
- `docs/DESIGN.md`: Menambahkan `RekamMedisList` ke Component Registry.
- `e2e-test.mjs`: Pengujian otomatis Playwright Headless Chromium (1440x900) mencakup seluruh 17 alur (Admin, Dokter, Petugas, Papan Antrian TV) dengan 100% PASS.
- **Verifikasi**: `npx ng test --watch=false` → **44 test files, 229 tests PASS** (exit 0).

## Addendum — Unifikasi Identitas Nama Klinik (Klinik Sehat Jaya)

- **Unifikasi Nama Klinik di Seluruh Lapisan**:
  - Standarisasi nama klinik menjadi **`Klinik Sehat Jaya`** secara konsisten di seluruh layer.
  - `backend/.env`: Update `KLINIK_NAMA=Klinik Sehat Jaya`.
  - Database PostgreSQL (`klinik` table): Update row `id=1` nama menjadi `'Klinik Sehat Jaya'`.
  - `frontend/src/index.html`: Update title menjadi `<title>Klinik Sehat Jaya — Rekam Medis Elektronik & Antrian</title>`.
  - `shell.component.ts` & `shell.component.html`: Update default fallback nama klinik ke `'Klinik Sehat Jaya'`.
  - `papan-antrian.component.html`: Update fallback display TV ke `'Klinik Sehat Jaya'`.
  - `landing.component.html`: Update subtitle banner ke `Klinik Sehat Jaya — Rekam Medis Elektronik & Antrian`.
  - `admin-klinik.component.html`: Update fallback profil operasional ke `'Klinik Sehat Jaya'`.
  - `login.component.html`, `forgot-password.component.html`, `set-password.component.html`: Update brand header ke `Klinik Sehat Jaya`.
  - `e2e-test.mjs` & Test specs: Menyelaraskan seluruh mock dan assertion nama klinik.
- **Verifikasi**:
  - `npx ng test --watch=false` → **44 test files, 229 tests PASS** (exit 0).
  - `node e2e-test.mjs` → **17/17 automated flows PASS** (exit 0).

## Addendum — Pemisahan Beranda Berbasis Peran (*Role-Specific Tailored Workspaces*)

- `LandingComponent` (`features/shell/landing/`):
  - **🩺 Beranda Khusus Dokter (*Clinical Workbench*)**:
    - Hero Ribbon Dokter: Badge *Poli Rawat Jalan · Ruang Konsultasi Dokter*, status *Poli Siap Konsultasi*, jam praktek, serta 4 KPI Ribbon klinis (*Selesai Diperiksa, Sisa Antrian Periksa, Pasien Prioritas, Total Pasien*).
    - *Active Called Patient Spotlight*: Kartu pasien dipanggil dengan tombol 1-klik **"Mulai Pengisian SOAP"** atau prompt kesiapan poli.
    - *Antrian Periksa Tabbed*: Tab *Menunggu* vs *Selesai* dengan tombol cepat *Panggil* dan *Lihat SOAP*.
    - *Quick EMR Lookup*: Pencarian instan riwayat medis pasien via NIK/Nama tanpa perlu berpindah modul.
  - **📋 Beranda Khusus Petugas Loket (*Triage & Pendaftaran*)**:
    - Hero Ribbon Loket: Badge *Loket Pendaftaran & Triage Pasien*, status *Loket Aktif Melayani*, serta 4 KPI Ribbon loket (*Total Tiket Terbit, Menunggu di Ruang Tunggu, Selesai Diperiksa, Pasien Prioritas*).
    - *Fast Triage Cards*: Tombol langsung **"Registrasi Pasien Baru"** dan **"Daftarkan Pasien ke Antrian"**.
    - *Live Queue Feed & Panduan Triage*: Urutan antrian live ruang tunggu, link TV Papan Antrian, dan kriteria identifikasi pasien prioritas (lansia, ibu hamil/balita, disabilitas, darurat).
  - **🛡️ Beranda Khusus Administrator (*System Governance & Control Tower*)**:
    - Hero Ribbon Admin: Badge *Pusat Kendali Administrasi & Tata Kelola*, status buka/tutup klinik, serta 4 KPI Ribbon operasional (*Total Pasien, Tingkat Kehadiran, Efisiensi Konsultasi, Pasien Batal/No-Show*).
    - *Governance Triad*: Kartu status & navigasi cepat ke *Jejak Audit Keamanan (SHA-256 Chain)*, *Display Token Antrian TV*, dan *Manajemen Akun Staf (RBAC)*.
    - *Analytics & Queue Monitoring*: Feed seluruh antrian klinik dan ApexCharts radial gauge chart efisiensi pelayanan.
- **Verifikasi**:
  - `npx ng test --watch=false` → **44 test files, 229 tests PASS** (exit 0).
  - `node e2e-test.mjs` → **19/19 automated flows PASS** (exit 0).

## Addendum — Modularisasi Sub-Komponen Reusable Beranda (`LandingComponent`)

- Mengurai *monolithic landing template* (~980 baris) menjadi 5 sub-komponen terpisah, terisolasi, dan *reusable*:
  - `<app-landing-hero>` (`components/landing-hero/`): Reusable glassmorphic hero banner dengan live Jakarta time, status operasional, dan projected KPI grid.
  - `<app-landing-kpi-card>` (`components/landing-kpi-card/`): Reusable KPI metric card dengan 5 varian warna semantik token (`primary`, `emerald`, `amber`, `purple`, `sky`).
  - `<app-doctor-dashboard>` (`components/doctor-dashboard/`): Workspace klinis dokter terdedikasi (hero banner, active patient spotlight, antrian periksa tabbed, quick EMR search).
  - `<app-petugas-dashboard>` (`components/petugas-dashboard/`): Triage loket pendaftaran terdedikasi (hero banner, fast triage cards, live queue feed, panduan triage).
  - `<app-admin-dashboard-view>` (`components/admin-dashboard-view/`): Pusat kendali admin terdedikasi (hero banner, governance triad cards, live antrian, ApexCharts radial gauge).
- Template `landing.component.html` dirampingkan dari ~980 baris menjadi ~49 baris bersih yang deklaratif.
- Memperbarui Component Registry pada `docs/DESIGN.md`.
- **Verifikasi**:
  - `npx ng test --watch=false` → **44 test files, 229 tests PASS** (exit 0).
  - `node e2e-test.mjs` → **19/19 automated E2E & visual tests PASS** (exit 0).
  - Regex audit styling hardcode (`#[0-9a-fA-F]{3,6}|rgb\(`) → **0 match** (100% token semantik).
- **Perbaikan Reaktivitas Status Operasional & Pembersihan Panah Dekoratif (AI Slop)**:
## Addendum — Elevasi UI/UX Rekam Medis (EMR) SOAP & Perbaikan Tombol

- **Fitur**: Refinement visual, perbaikan UX copy, dan eliminasi cacat UI pada modul Rekam Medis:
  - **Perbaikan Double Plus**: Menghilangkan duplikasi icon `<ng-icon name="lucidePlus">` yang diikuti teks literal `+ Resep` / `+ Tindakan` pada header Section 4 dan modal addendum (`+ + Resep`, `+ + Tindakan`, `+ + Diagnosis` kini menjadi `Resep`, `Tindakan`, `Diagnosis` bersih dengan 1 ikon plus).
  - **Penyederhanaan Teks Tombol (Concise UX Copy)**:
    - Tombol footer: `Simpan Rekam Medis (Selesai)` → `Simpan Rekam Medis`, `Batal & Kembali` → `Batal`.
    - Tombol modal addendum: `Buat Addendum / Koreksi` → `Buat Addendum`, `Simpan Addendum Resmi` → `Simpan Addendum`.
    - Tombol empty state: `Tambah Resep Obat` → `Resep Obat`, `Tambah Tindakan Medis` → `Tindakan Medis`.
  - **Elevasi Metodologi SOAP**:
    - Memberikan badge inisial mono yang tegas: **S** (Anamnesis), **O** (Pemeriksaan Fisik), **A** (Diagnosis ICD-10), **P** (Tindakan & Resep).
    - Merancang ulang empty state Section 4 menjadi clinical action card yang bersih dengan icon `lucidePill` dan panduan terarah.
  - **Pembersihan AI Slop Panah**: Menghapus `lucideArrowRight` dari tombol aksi di `rekam-medis-list.component.html`.
- **Verifikasi**:
  - `npx ng test --watch=false` → **44 test files, 229 tests PASS** (exit 0).
  - `node e2e-test.mjs` → **19/19 automated E2E & visual tests PASS** (exit 0).
  - Regex audit styling hardcode (`#[0-9a-fA-F]{3,6}|rgb\(`) → **0 match**.
  - Regex audit double plus (`\+\s*\+`) → **0 match**.

---

## Addendum — Redesain Kartu KPI Dashboard Antrian (`AntrianDashboardComponent`)

- **Fitur**: Menggantikan 4 kartu KPI ringkasan statistik yang sebelumnya terhimpit (*squished*) tanpa padding di `/antrian` dengan komponen Spartan `LandingKpiCardComponent` (`<app-landing-kpi-card>`):
  - **Total Antrian**: Icon `lucideUsers`, varian warna `primary` (teal), nilai `totalCount()`.
  - **Menunggu**: Icon `lucideClock`, varian warna `amber` (kuning hangat ruang tunggu), nilai `menungguCount()`.
  - **Sedang Dipanggil**: Icon `lucideActivity`, varian warna `sky` (biru aktif), nilai `dipanggilCount()`.
  - **Selesai**: Icon `lucideCheckCircle2`, varian warna `emerald` (hijau selesai), nilai `selesaiCount()`.
  - Mengatur grid responsif (`grid-cols-2 lg:grid-cols-4 gap-4`) dengan padding presisi dan border token semantik seragam.
- **Verifikasi**:
  - `npx ng test --watch=false` → **44 test files, 229 tests PASS** (exit 0).
  - `node e2e-test.mjs` → **19/19 automated E2E & visual snapshot tests PASS** (exit 0).
  - Visual snapshot `screenshots/07_antrian_dashboard.png` terverifikasi rapi, proporsional, dan seimbang.
  - Regex audit styling hardcode (`#[0-9a-fA-F]{3,6}|rgb\(`) → **0 match**.
  - **Pembersihan Indikator Header**: Menghapus `<app-clinic-status-indicator />` dari header `/antrian` untuk tampilan yang lebih bersih dan fokus.

---

## Addendum — Penyelesaian Masalah P1 Frontend (Dead Code Cleanup & StatusBadge Deduplication)

- **Dead Code Cleanup (`LandingComponent`)**:
  - Menghapus interface `NavShortcut` dan computed signal `readonly shortcuts` (~180 baris) dari `landing.component.ts` yang sudah tidak digunakan di template setelah modularisasi sub-dashboard.
  - Menghapus unused property `recentAntrian`, signal `doctorAntrianTab`, method `setDoctorTab`, serta puluhan import ikon Lucide & library yang tidak terpakai langsung di parent `LandingComponent`.
  - Memperbarui unit test di `landing.component.spec.ts` untuk menguji rendering sub-dashboard per peran (`DoctorDashboard`, `PetugasDashboard`, `AdminDashboardView`) dan kalkulasi metrik ringkasan.
- **StatusBadge Deduplication (`PasienDetailComponent`)**:
  - Menggantikan render status manual (`[class]="getStatusBadgeClass(kunjungan.status)"` & `getStatusLabel()`) di riwayat kunjungan `pasien-detail.component.html` dengan shared primitive `<app-status-badge [status]="kunjungan.status" size="sm" />`.
  - Menghapus method duplikat `getStatusLabel` dan `getStatusBadgeClass` dari `pasien-detail.component.ts`.
- **Verifikasi**:
  - `npm test -- --run` → **44 test files, 229 tests PASS** (exit 0).
  - Regex audit styling hardcode (`#[0-9a-fA-F]{3,6}|rgb\(`) → **0 match**.
