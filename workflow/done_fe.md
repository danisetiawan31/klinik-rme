# Done Log — Frontend

## Scaffolding Frontend — Tahap 1-6 (Selesai Penuh)

- **Tahap 1-3 (Inisialisasi Dasar)**: Inisialisasi Angular CLI v21 (`ng new frontend-app --routing --style=css --test-runner=vitest`), install dan konfigurasi Tailwind CSS v4, verifikasi runner native Vitest.
- **Tahap 4-6 (Struktur & Konfigurasi Dasar)**: Pembuatan struktur folder (`core`, `features`, `shared`), konfigurasi `environment.ts` & `environment.development.ts` (`apiUrl: '/api/v1'`), konfigurasi proxy development `proxy.conf.json`, tipe `ErrorEnvelope` terpusat, dan `auth.interceptor.ts` dengan penanganan khusus (skip redirect) untuk token papan antrian dan endpoint auth.
- **Verifikasi**: `npm start` (`ng serve`) jalan tanpa error. `npm run test` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:

- **Testing**: Karena Angular 21 (v21.2.x) sudah mendukung Vitest secara native via builder `@angular/build:unit-test`, project ini **tidak** menggunakan Karma, tidak menggunakan `vite.config.ts` manual, dan tidak perlu dependensi pihak ketiga (`@analogjs/vite-plugin-angular`).
- **Environment config**: Penamaan file menggunakan default generasi dari Angular CLI v21, yaitu `environment.ts` (untuk production) dan `environment.development.ts` (untuk environment dev lokal).
- **HTTP Interceptor**: Dibuat sebagai `HttpInterceptorFn` (functional) sesuai best practice Angular 17+ (standalone), didaftarkan via `provideHttpClient`. Interceptor akan menahan logic redirect 401 ke `/login` jika request berasal dari papan antrian (header `X-Display-Token`) atau mengarah ke public auth endpoints (`/auth/login`, `/auth/forgot-password`, `/auth/reset-password`, `/auth/me`), memberikan ruang pada masing-masing komponen untuk menangani UI error-nya sendiri.

## Core Shell — Tahap 1-3 (Auth Infra, UI Shell & Testing - Selesai Penuh)

- **Auth Infra & Resolver (Tahap 1)**: `staffAuthResolver` di-await pada rute root staff (`path: ''`), `AuthService` berbasis Signal mengelola state auth (`id`, `nama`, `roles[]`), `roleGuard` mengarahkan unauthenticated ke `/login` dan role-mismatch ke `/forbidden` (`DESIGN.md` §9.5). `authInterceptor` meng-attach `withCredentials: true`, mem-parse `ErrorEnvelope`, dan menangani 401. Rute publik `/papan-antrian` tetap berdiri di luar guard & shell.
- **UI Shell & Navigasi (Tahap 2)**: `ShellComponent` dengan sidebar desktop collapsible & mobile drawer (`hlm-sheet`), header berisi `ClinicStatusIndicator` dan menu dropdown user (avatar + logout). Dynamic navigation & shortcut card di index route `/` (`LandingComponent`) yang disesuaikan secara presisi per role (`petugas`, `dokter`, `admin`).
- **Timezone Anchor (Asia/Jakarta)**: Pengaturan timezone global `Asia/Jakarta` dikonfigurasi secara menyeluruh pada `environment.ts` (`timezone: 'Asia/Jakarta'`), Angular `app.config.ts` (`LOCALE_ID: id-ID`, `DATE_PIPE_DEFAULT_OPTIONS`), serta helper utility `src/app/core/utils/date.utils.ts` (`getJakartaTimeString`, `formatJakartaDate`). Seluruh operasi tanggal & jam di frontend di-anchor ke waktu WIB.
- **Suite Unit Testing (Tahap 3)**: Unit testing menyeluruh untuk seluruh komponen UI Shell:
  - `LandingComponent` (`landing.component.spec.ts`): Memverifikasi render shortcut per role (`petugas`, `dokter`, `admin`) dengan positive & negative assertions (memastikan shortcut role lain tidak pernah muncul).
  - `ShellComponent` (`shell.component.spec.ts`): Memverifikasi item navigasi sidebar/drawer per role dengan positive & negative assertions, serta menguji pemanggilan aksi logout.
  - `ClinicStatusIndicatorComponent` (`clinic-status-indicator.component.spec.ts`): Regresi test untuk meng-assert penggunaan token semantik `--color-accent` (Buka) & `--color-muted-foreground` (Tutup), serta memastikan bebas dari hardcoded class `emerald` dan animasi pulse.
- **Verifikasi**: Build & unit test (14 test files, 46 unit tests) PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- `ForbiddenComponent` ditempatkan di `shared/components/forbidden/` sebagai komponen infra state generik.
- `defaultKlinikId: 1` dikonfigurasi di file environment (`environment.ts` & `environment.development.ts`) untuk menghindari hardcoded magic number di `KlinikService`.
- Status `Klinik Buka` di-render dengan token semantik `bg-[#F0FDF4]` & `text-[var(--color-accent)]` tanpa pulse animation berulang untuk menjaga prinsip desain *restrained* (`DESIGN.md` §1).
- Perhitungan waktu buka/tutup dan format tanggal di FE selalu menggunakan waktu `Asia/Jakarta` (`Intl.DateTimeFormat` & `date.utils.ts`), selaras dengan aturan backend di `AGENTS.md` §7 & `DESIGN.md` §8.
- Interceptor mengecualikan `/auth/me` dari auto-redirect 401 agar `AuthService.fetchMe()` menangani status 401 secara terisolasi tanpa memicu redirect loop.
- Navigasi Antrian untuk `admin` disesuaikan mengarah ke `/antrian` (sama seperti `petugas`/`dokter`), memfasilitasi akses pendaftaran dan pemantauan antrian sesuai `api-contract.md`.
- **Modifikasi Primitive (`src/app/shared/ui/sheet/src/lib/hlm-sheet-content.ts`)**: Penyesuaian `injectExposesStateProvider` dan `injectExposedSideProvider` dari `{ host: true }` menjadi `{ optional: true }` dengan explicit fallback signals `state = this._stateProvider?.state ?? signal('closed')` dan `side = computed(() => this._sideProvider?.side() ?? 'left')`. Hal ini mencegah runtime DI error (`NG0201: No provider found`) saat `HlmSheetContent` di-render pada konteks testing/isolated host tanpa mengorbankan fungsionalitas asli di real usage.
- **Catatan Pengembangan Modul Selanjutnya**: `DATE_PIPE_DEFAULT_OPTIONS.timezone` mengasumsikan timestamp backend membawa offset eksplisit (ISO format `Z` / `+07:00`). Saat memulai modul Antrian (`dipanggilAt`) / Audit Log (`createdAt`), wajib memverifikasi format raw response JSON backend terlebih dahulu sebelum memasang DatePipe secara masif.

## Auth Recovery Pages — Tahap 1 & 2 (ForgotPasswordComponent & SetPasswordComponent - Selesai Penuh)

- **Auth Infra & API**: Method `forgotPassword(email)` (`POST /auth/forgot-password`) & `resetPassword(token, passwordBaru)` (`POST /auth/reset-password`) di `AuthService`, plus tipe terkait di `auth.types.ts`.
- **ForgotPasswordComponent (`/forgot-password`)**: Reactive Form email (`required`, `email`), Zona Hero (`docs/DESIGN.md`), card swap sukses generik 200 (cegah user enumeration), `ToastComponent` untuk error teknis (500/network).
- **SetPasswordComponent (`/set-password`)**: Pembacaan query param `token`, Reactive Form (`passwordBaru` min 8 char, `konfirmasiPassword` mismatch validator), re-use `SensitiveValueComponent`. 3 Card State UI (Token Error State dengan CTA `/forgot-password`, Form State, Sukses State dengan CTA `/login`).
- **Routing & Test**: Public routes `/forgot-password` & `/set-password` di `app.routes.ts`, link "Lupa password?" di `LoginComponent`. 9 unit tests (17 test files, 64 unit tests total) PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- `routerLink="/forgot-password"` ditambahkan di `LoginComponent` sebagai entry point navigasi langsung dari halaman login.
- Error `INVALID_TOKEN` (400) & missing token ditangani kontekstual via Card State khusus (bukan Toast), memandu user meminta link baru ke `/forgot-password`.
- Styling murni menggunakan token semantik CSS variable (`var(...)`) tanpa hardcode warna.

---

## RealtimeService — Tahap 1 (Selesai Penuh)

- **RealtimeService (`core/realtime/realtime.service.ts`)**: Service Angular terpusat pembungkus native WebSocket untuk koneksi `GET /ws?klinikId=X[&displayToken=...]`. Expose reactive state via Angular Signals (`connectionStatus` dengan state `'connecting' | 'connected' | 'reconnecting' | 'disconnected'`, dan `lastUpdateAt`).
- **Mekanisme Reconnect & Backoff**: Implements exponential backoff (initial delay 1s, x2 per failure attempt, max delay cap 30s) dilengkapi jitter acak ±20% (`0.8` - `1.2`) untuk mencegah *thundering herd reconnect*. Counter backoff otomatis di-reset ke 1s saat koneksi `onopen` berhasil. Auto-reconnect hanya ter-trigger pada pemutusan jaringan tak terduga (bukan pemanggilan `disconnect()` manual).
- **Dual-Auth & Param Options**: `connect({ klinikId?, displayToken? })` mendukung opsi `klinikId` (default fallback ke `environment.defaultKlinikId`) dan query param `displayToken` (opsional untuk Papan Antrian). Jalur staff tanpa `displayToken` otomatis menggunakan cookie session browser (`withCredentials` default browser WebSocket API).
- **Invalidation Trigger**: Saat server mengirim pesan `{"type":"queue_updated"}`, Signal `lastUpdateAt` di-update dengan timestamp `Date.now()` untuk dikonsumsi oleh UI consumer via `effect()` / refetch REST.
- **Development Proxy**: Memperbarui `frontend/proxy.conf.json` dengan menambahkan entry `/ws` bertipe `"ws": true` mengarah ke `http://localhost:8080`.
- **Verifikasi**: Spec unit test `realtime.service.spec.ts` (9 unit tests) + seluruh test suite frontend (16 test files, 59 unit tests) PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- `RealtimeService` tidak melakukan REST fetch secara langsung; peran service ini sesuai `TDD.md` murni sebagai pemberi sinyal invalidation timestamp (`lastUpdateAt`), sehingga komponen consumer bertanggung jawab melakukan refetch data antrian sendiri via REST endpoint.
- Jitter ±20% dipasang secara eksplisit pada perhitungan delay reconnect untuk memastikan skenario server restart/single-instance drop tidak menyebabkan semua client melakukan thundering herd reconnect pada detik yang persis sama.

---

## Profil / Account Settings — Backlog Item 12 (ProfilComponent - Selesai Penuh)

- **AuthService Integration**: Menambahkan method `changePassword(passwordLama: string, passwordBaru: string): Observable<void>` (`PATCH /api/v1/auth/me/password`) dan tipe `ChangePasswordRequest` di `auth.types.ts`.
- **`ProfilComponent` (`/profil`)**: Komponen standalone halaman profil & pengaturan kata sandi (`src/app/features/profil/profil.component.ts`) mengikuti **Zona Content** (`docs/DESIGN.md` §1.1). 
  - Bagian 1: Ringkasan info akun read-only (nama, email, roles) bersumber langsung dari `AuthService.currentUser()`.
  - Bagian 2: Form Ubah Password (Reactive Forms + `SensitiveValueComponent` mode input), validasi `passwordBaru` (min 8 karakter) & `konfirmasiPassword` (`passwordsMismatch`).
- **Error Handling Inline & Toast**:
  - Error HTTP 400 `INVALID_PASSWORD` di-render sebagai **inline error** spesifik di bawah field `passwordLama` ("Password lama tidak sesuai").
  - Error teknis murni (500/network) di-render via `ToastComponent` (`type="error"`).
  - Submit sukses (HTTP 204) me-render `ToastComponent` (`type="success"`, "Password berhasil diubah"), me-reset 3 field password ke kosong, dan tetap berada di halaman `/profil`.
- **Shell & Routing Integration**: Menambahkan tautan "Pengaturan Akun" (`routerLink="/profil"`) pada dropdown `#userMenu` di `ShellComponent` di atas tombol Logout. Mendaftarkan child route `path: 'profil'` di bawah rute root shell di `app.routes.ts`.
- **Verifikasi**: Spec unit test `profil.component.spec.ts` (5 unit tests) + update `shell.component.spec.ts` (1 unit test link /profil) + seluruh test suite frontend (18 test files, 70 unit tests) PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- **Zona Content Strict Styling**: Halaman `/profil` secara ketat mengikuti panduan Zona Content (`docs/DESIGN.md` §1.1) dengan latar belakang solid (`var(--color-background)`) dan shadow (`var(--shadow-2)`), tanpa radial background gradient aksen teal di belakang panel utama demi menjaga kontras & keterbacaan tinggi.
- **Inline Error vs Toast**: Sesuai keputusan produk, `INVALID_PASSWORD` ditangani secara inline spesifik di bawah input `passwordLama` karena merupakan error validasi input milik field tertentu, sedangkan `ToastComponent` dikhususkan untuk notifikasi sukses (HTTP 204) & kegagalan teknis murni.

---

## Pasien — Tahap 1-3 (Backlog Item 14 - Selesai Penuh)

- **Fitur**: Modul CRUD pasien untuk staff — registrasi (`/pasien/baru` + consent + NIK format validation + warning duplikasi NIK non-blocking), pencarian (`/pasien` nik/nama dengan debounce 300ms nama & 16-digit NIK auto-trigger, pagination via generic `PaginationComponent` membaca header `X-Total-Count`), halaman detail (`/pasien/:id` + `riwayatKunjunganRingkas`), dan edit biodata (`/pasien/:id/edit` dengan 409 Optimistic Lock hybrid UX).
- **Verifikasi**: 41 unit test modul Pasien (16 form, 7 list, 6 detail, 5 edit, 2 routes, 5 pagination) mencakup validasi form, search trigger, kalkulasi pagination, 409 hybrid UX (field preservation & manual refetch), route collision (`/baru` & `/:id/edit` vs `/:id`), serta RBAC. Total regresi frontend 24 test files / 111 unit tests PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- `nikFormatValidator()` di-extract ke `pasien.validators.ts` agar reusable di form registrasi & edit tanpa duplikasi logic.
- Form edit sengaja mengecualikan field `consent` karena `PATCH /pasien/:id` backend silently ignore field tersebut (consent hanya diinput sekali saat registrasi).
- `PaginationComponent` (`shared/components/pagination/`) dibangun generic dengan Angular 17+ Signal API (`input()`/`output()`), didesain untuk reuse pada modul Antrian (#15) & Admin (#18).
- Feedback toast sukses (200) setelah edit dikirim via Angular Router `navigation.state` (`history.state.successMessage`) sehingga otomatis bersih saat di-refresh.
- Self-check hardcode styling (§8) memverifikasi 0 hex literal pada seluruh komponen pasien (menggunakan CSS variables semantik `docs/DESIGN.md`).

---

## Migrasi Token CSS & Tailwind v4 `@theme` — Tahap A s.d. B4 (Selesai Penuh)

- **Fondasi Token (Tahap A)**: Konsolidasi `:root` di `styles.css` sebagai alias tipis ke token Spartan (single source of truth). Daftarkan token `--input` (`#CCFBF1`), `--warning` & `--warning-foreground`, radius 4-tingkat (`rounded-sm/md/lg/full`), `shadow-1` s.d. `shadow-4`, dan `font-heading` ke blok `@theme inline`. Sinkronisasi `docs/DESIGN.md` Section 2.
- **Migrasi Komponen (Tahap B1-B3)**: Migrasi seluruh arbitrary bracket class (`[var(--...)]`) dan inline styling di seluruh modul (Shell, Shared, Auth, Profil, Pasien, Antrian, dan Admin) ke utility class standar Tailwind.
- **Verifikasi (Tahap B4)**: Sweeping regex `#[0-9a-fA-F]{3,6}|rgb\(|rgba\(` dan `\[var\(--` di seluruh `src/app/` menghasilkan 0 matches (NIHIL). Vitest (24 files, 111 unit tests) dan build production (`npx ng build`) PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- **Backward-Compatible CSS Alias**: Blok `:root { --color-* }` dipertahankan murni sebagai alias (`var(--primary)`) untuk SVG attribute dan dynamic inline CSS tanpa menduplikasi nilai.
- **Utility `.kl-auth-bg`**: Radial gradient latar Auth di-encapsulate dalam class `.kl-auth-bg` di `styles.css` menggunakan `color-mix(in srgb, var(--primary) 10%, transparent)`.

---

## Pemisahan File Template HTML (`templateUrl`) — Tahap 1 s.d. 3 (Selesai Penuh)

- **Modul Pasien (Tahap 1)**: Pemisahan template HTML untuk `PasienFormComponent`, `PasienListComponent`, `PasienDetailComponent`, dan `PasienEditComponent` ke file `.component.html` terpisah menggunakan `templateUrl`.
- **Modul Auth & Profil (Tahap 2)**: Pemisahan template HTML untuk `LoginComponent`, `ForgotPasswordComponent`, `SetPasswordComponent`, dan `ProfilComponent` ke file `.component.html` terpisah menggunakan `templateUrl`.
- **Shell & Shared Components (Tahap 3)**: Pemisahan template HTML untuk `ShellComponent`, `LandingComponent`, `AntrianDashboardComponent`, `AdminDashboardComponent`, `ForbiddenComponent`, `ClinicStatusIndicatorComponent`, `PaginationComponent`, `ToastComponent`, dan `SensitiveValueComponent` ke file `.component.html` terpisah.
- **Verifikasi**: Vitest (24 files, 111 unit tests) dan build production (`npx ng build`) lolos 100%. File controller `.ts` seluruh frontend kini ramping (~12–130 baris) dan template `.html` mendapatkan full IDE syntax highlighting, Emmet, dan formatting.

---

## Antrian (Staff-Facing) — Tahap 1 s.d. 3 (Backlog Item 15 - Selesai Penuh)

- **Tahap 1 (List & Realtime Dashboard)**: Addendum BE (`GET /klinik/:id/antrian` bawa `skipCount` & `priorityReason`), `StatusBadgeComponent` (4 status) & `PriorityBadgeComponent` (WCAG AA `text-foreground`, `DESIGN.md` §12), `AntrianDashboardComponent` (sort `isPriority DESC, skipCount ASC, nomorAntrian ASC`, dual-trigger WS `lastUpdateAt`/`connected`, `DestroyRef` teardown, summary cards).
- **Tahap 2 (Aksi Dokter & Petugas/Admin)**: `AntrianService` (`panggilBerikutnya`, `lewati`, `tidakHadir`), RBAC per-tombol (Dokter: Panggil & Lewati `dipanggil`; Dokter/Admin: Tidak Hadir `menunggu` + modal konfirmasi; Petugas: view-only), response 204 info toast, 409 conflict auto-refetch, proteksi double-submit `isSubmittingAction`.
- **Tahap 3 (Pendaftaran via PasienDetail)**: `AntrianService.create()` (`POST /kunjungan`), tombol "Daftarkan ke Antrian" di `PasienDetailComponent` (RBAC `isStaff()`, proaktif disabled saat `!isKlinikBuka()`), modal registrasi (validasi wajib `priorityReason` saat `isPriority` aktif), sukses toast dengan `nomorAntrian` tanpa navigasi (auto-refresh riwayat), modal error recovery (tetap terbuka jika submit gagal).
- **Verifikasi**: Vitest (28 files, 144 unit tests) dan build production (`npx ng build`) PASS 100%. Self-check styling 0 hex literal & 0 bracket CSS variables.

**Catatan Deviasi & Keputusan Teknis**:
- `StatusBadge` dan `PriorityBadge` ditempatkan di `shared/components/` untuk reuse lintas modul (Antrian, Pasien Detail, Rekam Medis).
- Validasi `priorityReason` di client dibuat lebih ketat dari backend (wajib isi jika prioritas aktif) demi integritas audit operasional.
- Helper `isStaff` di `PasienDetailComponent` memisahkan hak akses pendaftaran/edit data (petugas/admin) dari role dokter.

---

## Laporan Harian (Staff-Facing) — Backlog Item 19 (Selesai Penuh)

- **Fitur**: Halaman rekapitulasi harian staff (`/laporan-harian`), integrasi `LaporanService` (`GET /api/v1/laporan/harian`), filter tanggal native `<input type="date">` (default hari ini dalam timezone `Asia/Jakarta` via `getJakartaISODate()`), auto-fetch saat inisialisasi dan saat tanggal filter diubah via event `(change)`, 3 kartu ringkasan inline (Total Kunjungan, Selesai Dilayani, Tidak Hadir) menggunakan token semantik Tailwind, loading state, dan penanganan error via `<app-toast>`.
- **Routing & RBAC**: Mendaftarkan rute `laporan-harian` di `app.routes.ts` di bawah Shell dengan guard `roleGuard('petugas', 'dokter', 'admin')` (akses penuh seluruh role staff).
- **Verifikasi**: Vitest (30 files, 152 unit tests) dan build production (`npx ng build`) PASS 100%. Self-check styling 0 hex literal & 0 bracket CSS variables.

**Catatan Deviasi & Keputusan Teknis**:
- Kartu ringkasan statistik ditulis inline di template `LaporanHarianComponent` (mengikuti pola kartu ringkasan di `AntrianDashboardComponent`) untuk menjaga kesederhanaan scope fitur tanpa menambah overhead registry.
- Helper `formatJakartaDate()` dieksploitasi ulang untuk format label tanggal bahasa Indonesia di banner informasi bawah.

---

## Addendum — Spartan UI Modernization (Sidebar-Inset & Sonner Toast)

- **Spartan Sidebar-Inset (`shared/ui/sidebar`)**: Implementasi struktur sidebar resmi Spartan (`sidebar-inset`) pada `ShellComponent` lengkap dengan collapsible icon mode, layout inset responsif, header medical health cross icon, dan bottom avatar profil terpusat.
- **Spartan Sonner Toaster (`shared/ui/sonner`)**: Integrasi `ngx-sonner` & `@spartan-ng/brain/sonner` (`HlmToaster`) dengan konfigurasi semantik tema tokens, posisi default `top-right`, dan host global pada `app.html`.
- **Migrasi Terpusat Lintas Halaman**: Seluruh fitur (Antrian, Pasien Form/Detail/Edit/List, Login, Forgot Password, Set Password, Profil, Laporan Harian) dimigrasikan dari template toast lokal ke pemanggilan programatik langsung `toast.success()`, `toast.error()`, `toast.info()`.
- **Verifikasi**: Vitest (30 test files, 154 unit tests) PASS 100%. GitNexus change detection: 0 affected processes (risk level: low).

---

## Addendum — Migrasi Modal ke Spartan Dialog (HlmDialog)

- **Primitive**: `npx ng g @spartan-ng/cli:ui dialog` → `shared/ui/dialog/` (11 file, 0 hardcoded hex).
- **Antrian Dashboard**: modal "Tandai Tidak Hadir" `div.fixed.inset-0` → `<hlm-dialog #confirmTidakHadirDialog>` + `viewChild<HlmDialog>().open()/close()`; tombol pakai `hlmBtn variant="destructive"|"outline"`.
- **Pasien Detail**: modal "Daftarkan ke Antrian" → `<hlm-dialog #daftarAntrianDialog>` pola sama; state signals tetap.
- **Test**: 2 assertion `fixture.nativeElement.textContent` → `document.body.textContent` (CDK overlay portal render di luar nativeElement).
- **DESIGN.md**: `ConfirmDialog` → Selesai; `Dialog` ditambah ke baris Primitive.

**File**: `shared/ui/dialog/` (baru), `antrian-dashboard.*`, `pasien-detail.*`, `docs/DESIGN.md`.
**Verifikasi**: `npm test -- --run` → **30 files, 154 tests PASS** (exit 0).

---

## Addendum — Migrasi Card & Widget Statistik ke Spartan Card (HlmCard)

- **Primitive**: `npx ng g @spartan-ng/cli:ui card` → `shared/ui/card/` (9 file, 0 hardcoded hex).
- **Antrian Dashboard**: 4 kartu ringkasan antrian `div.bg-card` → `<hlm-card size="sm">` + `<div hlmCardContent>`.
- **Laporan Harian**: 3 kartu metrik statistik dan banner tanggal `div.bg-card` → `<hlm-card>` terstruktur.
- **Auth & Profil**: Kontainer kartu form pada Login, Forgot Password, Set Password, dan Profil → `<hlm-card>`.
- **Modul Pasien**: Kontainer form pendaftaran baru, form edit, serta kartu banner/biodata/riwayat pada Pasien Detail → `<hlm-card>`.
- **DESIGN.md**: `Card` ditambahkan ke baris Spartan Primitive di Component Registry.

**File**: `shared/ui/card/` (baru), `antrian-dashboard.*`, `laporan-harian.*`, `login.*`, `forgot-password.*`, `set-password.*`, `profil.*`, `pasien-form.*`, `pasien-edit.*`, `pasien-detail.*`, `docs/DESIGN.md`.
**Verifikasi**: `npm test -- --run` → **30 files, 154 tests PASS** (exit 0).

---

## Addendum — Migrasi Button & Tombol Aksi ke Spartan Button (HlmButton)

- **Standarisasi Semantik**: Menggantikan seluruh class manual ad-hoc `kl-btn-*` dan inline utility tailwind dengan direktif `button[hlmBtn]` / `a[hlmBtn]` dengan varian semantik (`default`, `outline`, `secondary`, `destructive`) dan ukuran (`size="sm" | "default"`).
- **Fitur Anti-Spam & Disabled**: `HlmButton` mengaktifkan `data-disabled:pointer-events-none` dan `data-disabled:opacity-50` saat tombol dalam kondisi `[disabled]="isLoading()"` untuk mencegah klik ganda/spam request.
- **Cakupan Halaman**:
  - Antrian: Tombol Panggil Berikutnya (`hlmBtn`), Lewati (`variant="secondary" size="sm"`), Tidak Hadir (`variant="destructive" size="sm"`).
  - Pasien: Registrasi Pasien Baru (`hlmBtn`), Aksi Detail (`variant="secondary" size="sm"`), Daftarkan ke Antrian (`hlmBtn`), Edit Biodata (`variant="secondary"`), Tombol Batal (`variant="outline"`), Simpan Form Pasien (`hlmBtn`).
  - Auth & Profil: Tombol submit Login, Forgot Password, Set Password, dan Ubah Kata Sandi Profil (`hlmBtn`).
  - Pagination, Admin & Forbidden: Tombol Sebelumnya & Selanjutnya (`variant="outline" size="sm"`), Tombol Keluar Admin (`variant="secondary" size="sm"`), Tombol Kembali ke Beranda & Ganti Akun (`forbidden.component.*`).

**File**: `antrian-dashboard.*`, `pasien-list.*`, `pasien-detail.*`, `pasien-form.*`, `pasien-edit.*`, `login.*`, `forgot-password.*`, `set-password.*`, `profil.*`, `pagination.*`, `admin-dashboard.*`, `forbidden.*`, `shell.*`.
**Verifikasi**: `npm test -- --run` → **30 files, 154 tests PASS** (exit 0), `kl-btn` references: **0 match**.

---

## Addendum — Migrasi Table / Data Table ke Spartan Table (HlmTable)

- **Primitive**: Direktif Spartan Table (`HlmTableImports`) di `shared/ui/table/` (`HlmTableContainer`, `HlmTable`, `HlmTHead`, `HlmTBody`, `HlmTFoot`, `HlmTr`, `HlmTh`, `HlmTd`, `HlmCaption`).
- **Antrian Dashboard**: Tabel antrian hari ini dimigrasi ke `div[hlmTableContainer]` + `table[hlmTable]`, `thead[hlmTableHeader]`, `tbody[hlmTableBody]`, `tr[hlmTableRow]`, `th[hlmTableHead]`, `td[hlmTableCell]`.
- **Modul Pasien**:
  - `pasien-list`: Tabel live-search & daftar pasien dimigrasi ke struktur `hlmTable`.
  - `pasien-detail`: List riwayat kunjungan ringkas distandarisasi ke format `hlmTable` 3 kolom (ID Kunjungan, Tanggal, Status).
- **DESIGN.md**: `Table` ditambahkan ke baris Spartan Primitive di Component Registry.

**File**: `shared/ui/table/` (baru), `antrian-dashboard.*`, `pasien-list.*`, `pasien-detail.*`, `docs/DESIGN.md`.
**Verifikasi**: `npm test -- --run` → **30 files, 154 tests PASS** (exit 0).

---

## Addendum — Integrasi Shimmer Loading Placeholder dengan Spartan Skeleton (HlmSkeleton)

- **Primitive**: Direktif `HlmSkeleton` (`HlmSkeletonImports`) di `shared/ui/skeleton/` (`[hlmSkeleton], hlm-skeleton`).
- **Shimmer Content Placeholders**: Menggantikan spinner putar dan teks statis "Memuat..." dengan skeleton placeholder berdenyut yang mempertahankan dimensi layout (Zero CLS):
  - **Antrian Dashboard**: Skeleton tabel 4 baris lengkap (nomor, nama pasien, badge status, tombol aksi).
  - **Pasien List**: Skeleton tabel 5 baris pada area hasil live-search.
  - **Laporan Harian**: 3 kartu metrik statistik dan banner tanggal skeleton.
  - **Pasien Detail**: Header banner skeleton (nama, ID badge, tombol aksi) + grid detail biodata & riwayat kunjungan.
- **DESIGN.md**: `Skeleton` ditambahkan ke baris Spartan Primitive di Component Registry.

**File**: `shared/ui/skeleton/` (baru), `antrian-dashboard.*`, `pasien-list.*`, `laporan-harian.*`, `pasien-detail.*`, `docs/DESIGN.md`.
**Verifikasi**: `npm test -- --run` → **30 files, 154 tests PASS** (exit 0).

---

## Addendum — Form Controls Tahap 1: Primitives & Migrasi Modul Auth / Sensitive Value

- **Primitives**: Direktif `HlmInput` (`[hlmInput]`), `HlmLabel` (`[hlmLabel]`), dan `HlmTextarea` (`[hlmTextarea]`) di `shared/ui/{input,label,textarea}/`.
- **Sensitive Value**: Input password & NIK masking dimigrasi ke `hlmInput` dengan atribut aksesibilitas penuh (`[id]`, `[name]`, `[title]`, `placeholder`, `[attr.aria-label]`).
- **Modul Auth**:
  - `login`: Field email & password dimigrasi ke `hlmLabel` + `hlmInput`.
  - `forgot-password`: Field email terdaftar dimigrasi ke `hlmLabel` + `hlmInput`.
  - `set-password`: Field password baru & konfirmasi dimigrasi ke `hlmLabel` + `hlmInput`.
- **Aksesibilitas Shell**: Menambahkan `aria-label` dan `title` pada `hlmSidebarRail` dan `hlmSidebarTrigger` di `shell.component.html`.

**File**: `shared/ui/input/` (baru), `shared/ui/label/` (baru), `shared/ui/textarea/` (baru), `sensitive-value.*`, `login.*`, `forgot-password.*`, `set-password.*`, `shell.*`.
**Verifikasi**: `npm test -- --run` → **30 files, 154 tests PASS** (exit 0).

---

## Addendum — Form Controls Tahap 2 & 3: Migrasi Profil, Laporan Harian, Modul Pasien & Pembersihan Total CSS Legacy

- **Profil & Laporan Harian (Tahap 2)**:
  - `profil`: Field kata sandi lama, baru, dan konfirmasi dimigrasi ke `label[hlmLabel]` + `SensitiveValueComponent` (`hlmInput`).
  - `laporan-harian`: Filter tanggal dimigrasi menggunakan `label[hlmLabel]` + `input[hlmInput]`.
- **Modul Pasien (Tahap 3)**:
  - `pasien-form`: Form registrasi pasien (NIK, Nama, Tanggal Lahir, Jenis Kelamin, Alamat, No. Telp) dimigrasi ke `hlmLabel`, `hlmInput`, `hlmTextarea`, dan semantic `<select>` styling.
  - `pasien-edit`: Form edit biodata pasien dimigrasi ke `hlmLabel`, `hlmInput`, `hlmTextarea`, dan semantic `<select>` styling.
  - `pasien-list`: Form pencarian live-search Nama & NIK dimigrasi ke `hlmLabel` + `hlmInput`.
  - `pasien-detail`: Textarea alasan prioritas pada modal antrian dimigrasi ke `hlmLabel` + `hlmTextarea`.
- **Pembersihan CSS Legacy**:
  - Menghapus seluruh blok CSS legacy `.kl-input`, `.kl-pw-wrap`, dan `.kl-pw-toggle` dari `styles.css`.
  - 0 match sisa kelas `kl-input` di seluruh codebase frontend.
- **Component Registry**:
  - `docs/DESIGN.md` diperbarui dengan memasukkan `Input / Label / Textarea` pada baris Spartan Primitive berstatus **Selesai**.

**File**: `profil.*`, `laporan-harian.*`, `pasien-form.*`, `pasien-edit.*`, `pasien-list.*`, `pasien-detail.*`, `styles.css`, `docs/DESIGN.md`.
**Verifikasi**: `npm test -- --run` → **30 files, 154 tests PASS** (exit 0).

---

## Addendum — Integrasi Spartan Alert (HlmAlert)

- **Primitive Spartan Alert**: Direktif `HlmAlert` (`hlm-alert, [hlmAlert]`), `HlmAlertTitle`, `HlmAlertDescription`, dan `HlmAlertAction` di `shared/ui/alert/` dengan varian CVA semantik (`default`, `destructive`, `warning`).
- **Modul Pasien**:
  - `pasien-edit`: Banner konflik 409 Optimistic Locking dimigrasi ke `<hlm-alert variant="warning">` dengan tombol aksi reload terintegrasi di dalam `<div hlmAlertAction>`.
  - `pasien-form`: Banner peringatan NIK duplikat dimigrasi ke `<hlm-alert variant="warning">` dengan judul dan deskripsi terstruktur.
  - `pasien-detail`: Error alert validasi modal antrian prioritas dimigrasi ke `<hlm-alert variant="destructive">`.
- **Component Registry**: `docs/DESIGN.md` diperbarui dengan memasukkan `Alert` pada baris Spartan Primitive berstatus **Selesai**.

**File**: `shared/ui/alert/` (baru), `pasien-edit.*`, `pasien-form.*`, `pasien-detail.*`, `docs/DESIGN.md`.
**Verifikasi**: `npm test -- --run` → **30 files, 154 tests PASS** (exit 0).

---

## Addendum — Pembuatan & Integrasi ConnectionStatusIndicator

- **Reusable Component**: `ConnectionStatusIndicatorComponent` (`app-connection-status-indicator`) di `shared/components/connection-status-indicator/` dengan varian `connected` (live indicator + subtle ping glow), `reconnecting` (pulse warning), dan `disconnected` (muted offline) serta aksesibilitas `role="status"` + `aria-live="polite"`.
- **Modul Antrian**: Mengganti markup inline status realtime di `antrian-dashboard.component.html` dengan `<app-connection-status-indicator />`.
- **Component Registry**: `docs/DESIGN.md` diperbarui dengan menandai `ConnectionStatusIndicator` berstatus **Selesai**.

**File**: `shared/components/connection-status-indicator/` (baru), `antrian-dashboard.*`, `docs/DESIGN.md`.
**Verifikasi**: `npm test -- --run` → **31 files, 159 tests PASS** (exit 0).

---

## Addendum — Integrasi Spartan Empty State (HlmEmpty)

- **Primitive Spartan Empty**: Direktif `HlmEmpty` (`hlm-empty, [hlmEmpty]`), `HlmEmptyHeader`, `HlmEmptyMedia`, `HlmEmptyTitle`, `HlmEmptyDescription`, dan `HlmEmptyContent` di `shared/ui/empty/` dengan varian semantik (`border-dashed`, `bg-muted` media box).
- **Modul Antrian**: Mengganti tampilan kosong antrian hari ini di `antrian-dashboard.component.html` dengan `<hlm-empty>`.
- **Modul Pasien**:
  - `pasien-list`: Mengganti tampilan hasil pencarian kosong dengan `<hlm-empty>` lengkap dengan tombol aksi ajakan bertindak `Daftar Pasien Baru`.
  - `pasien-detail`: Mengganti teks riwayat kunjungan kosong dengan `<hlm-empty>`.
- **Component Registry**: `docs/DESIGN.md` diperbarui dengan menandai `Empty` pada baris Spartan Primitive berstatus **Selesai**.

**File**: `shared/ui/empty/` (baru), `antrian-dashboard.*`, `pasien-list.*`, `pasien-detail.*`, `docs/DESIGN.md`.
**Verifikasi**: `npm test -- --run` → **31 files, 159 tests PASS** (exit 0).



