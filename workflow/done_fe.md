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

