# Done Log — Frontend

## Scaffolding Frontend — Tahap 1-6 (Selesai Penuh)

- **Tahap 1-3 (Inisialisasi Dasar)**: Inisialisasi Angular CLI v21 (`ng new frontend-app --routing --style=css --test-runner=vitest`), install dan konfigurasi Tailwind CSS v4, verifikasi runner native Vitest.
- **Tahap 4-6 (Struktur & Konfigurasi Dasar)**: Pembuatan struktur folder (`core`, `features`, `shared`), konfigurasi `environment.ts` & `environment.development.ts` (`apiUrl: '/api/v1'`), konfigurasi proxy development `proxy.conf.json`, tipe `ErrorEnvelope` terpusat, dan `auth.interceptor.ts` dengan penanganan khusus (skip redirect) untuk token papan antrian dan endpoint auth.
- **Verifikasi**: `npm start` (`ng serve`) jalan tanpa error. `npm run test` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:

- **Testing**: Karena Angular 21 (v21.2.x) sudah mendukung Vitest secara native via builder `@angular/build:unit-test`, project ini **tidak** menggunakan Karma, tidak menggunakan `vite.config.ts` manual, dan tidak perlu dependensi pihak ketiga (`@analogjs/vite-plugin-angular`).
- **Environment config**: Penamaan file menggunakan default generasi dari Angular CLI v21, yaitu `environment.ts` (untuk production) dan `environment.development.ts` (untuk environment dev lokal).
- **HTTP Interceptor**: Dibuat sebagai `HttpInterceptorFn` (functional) sesuai best practice Angular 17+ (standalone), didaftarkan via `provideHttpClient`. Interceptor akan menahan logic redirect 401 ke `/login` jika request berasal dari papan antrian (header `X-Display-Token`) atau mengarah ke public auth endpoints (`/auth/login`, `/auth/forgot-password`, `/auth/reset-password`, `/auth/me`), memberikan ruang pada masing-masing komponen untuk menangani UI error-nya sendiri.

## Core Shell — Tahap 1 (Auth Infra)

- **Auth resolver (`staffAuthResolver`)**: resolver di-await yang ter-scope pada rute root staff (`path: ''`), menginisialisasi `GET /auth/me` via `AuthService.fetchMe()`. Rute publik `/papan-antrian` berada di luar tree ini sehingga tidak tertahan.
- **Signal-based Auth State Service (`AuthService`)**: menyimpan state pengguna (`id`, `nama`, `roles[]`), status loading, dan error auth secara reaktif via Signals.
- **Route Guard per Role (`roleGuard`)**: mengarahkan pengguna belum login ke `/login`, dan pengguna terautentikasi tanpa role yang sesuai ke `/forbidden` (`DESIGN.md` §9.5).
- **Forbidden State Component (`ForbiddenComponent`)**: dibuat di `src/app/shared/components/forbidden/forbidden.component.ts` dengan desain visual modern berbasis token semantik `DESIGN.md` §2–§5 (Figtree font, Noto Sans, background/card tokens, Lucide icon) dan meng-reuse `HlmButton` Spartan UI primitive.
- **HTTP Interceptor (`authInterceptor`)**: attach `withCredentials: true`, handle redirect 401 ke `/login` untuk request umum staff, menahan redirect untuk public auth endpoints & display token request, serta mem-parse error body ke format `ErrorEnvelope`.
- **Verifikasi**: Unit test (32 tests total) PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:
- `ForbiddenComponent` ditempatkan di `shared/components/forbidden/` sebagai komponen infra/shared state yang dipicu oleh guard lintas rute.
- Spartan Button primitive di-reuse dari `@spartan-ng/helm/button` (`src/app/shared/ui/button`) tanpa fetch ulang.
- Interceptor mengecualikan `/auth/me` agar `AuthService.fetchMe()` dapat menangani error 401 secara terisolasi tanpa memicu double-navigation atau redirect loop.
