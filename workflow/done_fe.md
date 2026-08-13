# Done Log — Frontend

## Scaffolding Frontend — Tahap 1-6 (Selesai Penuh)

- **Tahap 1-3 (Inisialisasi Dasar)**: Inisialisasi Angular CLI v21 (`ng new frontend-app --routing --style=css --test-runner=vitest`), install dan konfigurasi Tailwind CSS v4, verifikasi runner native Vitest.
- **Tahap 4-6 (Struktur & Konfigurasi Dasar)**: Pembuatan struktur folder (`core`, `features`, `shared`), konfigurasi `environment.ts` & `environment.development.ts` (`apiUrl: '/api/v1'`), konfigurasi proxy development `proxy.conf.json`, tipe `ErrorEnvelope` terpusat, dan `auth.interceptor.ts` dengan penanganan khusus (skip redirect) untuk token papan antrian dan endpoint auth.
- **Verifikasi**: `npm start` (`ng serve`) jalan tanpa error. `npm run test` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:

- **Testing**: Karena Angular 21 (v21.2.x) sudah mendukung Vitest secara native via builder `@angular/build:unit-test`, project ini **tidak** menggunakan Karma, tidak menggunakan `vite.config.ts` manual, dan tidak perlu dependensi pihak ketiga (`@analogjs/vite-plugin-angular`).
- **Environment config**: Penamaan file menggunakan default generasi dari Angular CLI v21, yaitu `environment.ts` (untuk production) dan `environment.development.ts` (untuk environment dev lokal).
- **HTTP Interceptor**: Dibuat sebagai `HttpInterceptorFn` (functional) sesuai best practice Angular 17+ (standalone), didaftarkan via `provideHttpClient`. Interceptor akan menahan logic redirect 401 ke `/login` jika request berasal dari papan antrian (header `X-Display-Token`) atau mengarah ke public auth endpoints (`/auth/login`, `/auth/forgot-password`, `/auth/reset-password`, `/auth/me`), memberikan ruang pada masing-masing komponen untuk menangani UI error-nya sendiri.

## Core Shell — Tahap 1-2 (Auth Infra & UI Shell)

- **Auth Infra & Resolver**: `staffAuthResolver` di-await pada rute root staff (`path: ''`), `AuthService` berbasis Signal mengelola state auth (`id`, `nama`, `roles[]`), `roleGuard` mengarahkan unauthenticated ke `/login` dan role-mismatch ke `/forbidden` (`DESIGN.md` §9.5). `authInterceptor` meng-attach `withCredentials: true`, mem-parse `ErrorEnvelope`, dan menangani 401. Rute publik `/papan-antrian` tetap berdiri di luar guard & shell.
- **`ClinicStatusIndicator`**: Badge status Buka/Tutup di header staff. Status Buka menggunakan token semantik `--color-accent` (`#16A34A`) tanpa animasi pulse (restrained design `DESIGN.md` §1/§7), sedangkan status Tutup menggunakan `--color-muted-foreground` (`DESIGN.md` §9.4 — state normal penutupan hari, **bukan** merah).
- **Timezone Anchor (Asia/Jakarta)**: Pengaturan timezone global `Asia/Jakarta` dikonfigurasi secara menyeluruh pada `environment.ts` (`timezone: 'Asia/Jakarta'`), Angular `app.config.ts` (`LOCALE_ID: id-ID`, `DATE_PIPE_DEFAULT_OPTIONS`), serta helper utility `src/app/core/utils/date.utils.ts` (`getJakartaTimeString`, `formatJakartaDate`). Seluruh operasi tanggal & jam di frontend di-anchor ke waktu WIB.
- **Verifikasi**: Build & unit test (37 tests total) PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- `ForbiddenComponent` ditempatkan di `shared/components/forbidden/` sebagai komponen infra state generik.
- `defaultKlinikId: 1` dikonfigurasi di file environment (`environment.ts` & `environment.development.ts`) untuk menghindari hardcoded magic number di `KlinikService`.
- Status `Klinik Buka` di-render dengan token semantik `bg-[#F0FDF4]` & `text-[var(--color-accent)]` tanpa pulse animation berulang untuk menjaga prinsip desain *restrained* (`DESIGN.md` §1).
- Perhitungan waktu buka/tutup dan format tanggal di FE selalu menggunakan waktu `Asia/Jakarta` (`Intl.DateTimeFormat` & `date.utils.ts`), selaras dengan aturan backend di `AGENTS.md` §7 & `DESIGN.md` §8.
- Interceptor mengecualikan `/auth/me` dari auto-redirect 401 agar `AuthService.fetchMe()` menangani status 401 secara terisolasi tanpa memicu redirect loop.
- Navigasi Antrian untuk `admin` disesuaikan mengarah ke `/antrian` (sama seperti `petugas`/`dokter`), memfasilitasi akses pendaftaran dan pemantauan antrian sesuai `api-contract.md`.
- **Catatan Pengembangan Modul Selanjutnya**: `DATE_PIPE_DEFAULT_OPTIONS.timezone` mengasumsikan timestamp backend membawa offset eksplisit (ISO format `Z` / `+07:00`). Saat memulai modul Antrian (`dipanggilAt`) / Audit Log (`createdAt`), wajib memverifikasi format raw response JSON backend terlebih dahulu sebelum memasang DatePipe secara masif.
