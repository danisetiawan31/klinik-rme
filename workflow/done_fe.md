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

## Auth Recovery Pages — Tahap 1 (ForgotPasswordComponent)

- **AuthService Integration**: Menambahkan method `forgotPassword(email: string): Observable<{ message: string }>` yang mengirim HTTP POST ke `/api/v1/auth/forgot-password` dan menambahkan tipe `ForgotPasswordRequest` / `ForgotPasswordResponse` pada `auth.types.ts`.
- **`ForgotPasswordComponent`**: Komponen standalone halaman lupa password (`src/app/features/auth/forgot-password/forgot-password.component.ts`) mengikuti styling Zona Hero (`docs/DESIGN.md`). Dilengkapi Reactive Form validasi email (`required`, `email`).
- **Respon Generik 200 & Security**: Sesuai kebijakan keamanan cegah *user enumeration* (`AGENTS.md` §7), submit sukses akan meng-swap kartu form ke pesan sukses generik ("Jika email terdaftar, instruksi reset password telah dikirim") dengan info TTL 1 jam dan tombol CTA kembali ke `/login`.
- **Handling Error Teknis**: Jika HTTP request mengalami kegagalan teknis (network failure/timeout/500), error ditangkap dan ditampilkan via `ToastComponent` (`type="error"`) top-center tanpa meng-swap form sehingga user dapat mencoba submit ulang.
- **Routing & Link**: Mendaftarkan route publik top-level `path: 'forgot-password'` pada `app.routes.ts` dan menyambungkan link "Lupa password?" dari `LoginComponent`.
- **Verifikasi**: Unit test `forgot-password.component.spec.ts` (4 unit tests) + seluruh test suite (15 test files, 50 unit tests) PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- Endpoint `/auth/forgot-password` selalu mengembalikan HTTP 200 generik untuk mencegah enumeration, sehingga notifikasi Toast `type="error"` sengaja dikhususkan untuk menangani kegagalan infrastruktur/koneksi teknis murni.

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

