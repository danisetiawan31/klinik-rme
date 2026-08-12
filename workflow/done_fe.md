# Done Log — Frontend

## Scaffolding Frontend — Tahap 1-6 (Selesai Penuh)

- **Tahap 1-3 (Inisialisasi Dasar)**: Inisialisasi Angular CLI v21 (`ng new frontend-app --routing --style=css --test-runner=vitest`), install dan konfigurasi Tailwind CSS v4, verifikasi runner native Vitest.
- **Tahap 4-6 (Struktur & Konfigurasi Dasar)**: Pembuatan struktur folder (`core`, `features`, `shared`), konfigurasi `environment.ts` & `environment.development.ts` (`apiUrl: '/api/v1'`), konfigurasi proxy development `proxy.conf.json`, tipe `ErrorEnvelope` terpusat, dan `auth.interceptor.ts` dengan penanganan khusus (skip redirect) untuk token papan antrian dan endpoint auth.
- **Verifikasi**: `npm start` (`ng serve`) jalan tanpa error. `npm run test` PASS 100%.

**Catatan Deviasi & Keputusan Teknis**:

- **Testing**: Karena Angular 21 (v21.2.x) sudah mendukung Vitest secara native via builder `@angular/build:unit-test`, project ini **tidak** menggunakan Karma, tidak menggunakan `vite.config.ts` manual, dan tidak perlu dependensi pihak ketiga (`@analogjs/vite-plugin-angular`).
- **Environment config**: Penamaan file menggunakan default generasi dari Angular CLI v21, yaitu `environment.ts` (untuk production) dan `environment.development.ts` (untuk environment dev lokal).
- **HTTP Interceptor**: Dibuat sebagai `HttpInterceptorFn` (functional) sesuai best practice Angular 17+ (standalone), didaftarkan via `provideHttpClient`. Interceptor akan menahan logic redirect 401 ke `/login` jika request berasal dari papan antrian (header `X-Display-Token`) atau mengarah ke public auth endpoints (`/auth/login`, `/auth/forgot-password`, `/auth/reset-password`, `/auth/me`), memberikan ruang pada masing-masing komponen untuk menangani UI error-nya sendiri.
