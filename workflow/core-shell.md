# Core Shell

## Konteks & tujuan

Fondasi routing & layout untuk seluruh halaman staff (petugas/dokter/admin) setelah login: resolusi auth state, route guard per role, layout dengan navigasi kondisional per role, dan landing page. Ini prasyarat teknis sebelum modul fitur (Pasien, Antrian, Rekam Medis, Admin) mulai dibangun — semua halaman protected nested di shell ini.

## Requirement

- **Auth resolver**: resolve `GET /auth/me` sebelum guard dievaluasi. Pakai `provideAppInitializer`/resolver yang di-await, **di-scope ke root route staff saja** (bukan `APP_INITIALIZER` global) — supaya route `papan-antrian` (publik, tanpa sesi staff) tidak ikut tertahan menunggu resolver ini.
- **Auth state**: simpan hasil resolve (`id`, `nama`, `roles[]`) di service berbasis Signal (`core/auth/`), diakses lintas komponen (guard, nav, layout).
- **Route guard per role**: `CanActivateFn`, baca role dari Signal auth state.
  - Belum authenticated → redirect `/login`.
  - Authenticated tapi role tidak sesuai kebutuhan route → state "Anda tidak punya akses" (bukan redirect diam-diam, bukan 404) — sesuai `DESIGN.md` §9.5.
  - Route group `papan-antrian` **tidak** pakai guard ini sama sekali.
- **HTTP interceptor** (`core/auth/` atau `core/http/`):
  - Attach `withCredentials: true` di semua request (perlu untuk cookie cross-origin di local dev).
  - Handle `401` → redirect `/login`.
  - Parse `error.code`/`error.message` sesuai format `api-contract.md`, expose ke caller supaya komponen bisa nampilkan lewat `ToastNotification` (sudah ada, status Selesai di Component Registry) — `message` yang dikurasi backend yang ditampilkan, bukan raw error.
- **Layout component (shell)**: sidebar collapsible, jadi drawer di breakpoint `< md` (konsisten `DESIGN.md` §4). Header berisi:
  - `ClinicStatusIndicator` (badge buka/tutup klinik, `DESIGN.md` §9.4) — fetch `GET /klinik/:id` untuk data jam operasional.
  - User menu: nama user + aksi logout (`POST /auth/logout`).
- **Nav item per role** (sudah disepakati saat planning):
  | Role | Menu |
  |---|---|
  | petugas | Pasien, Antrian, Laporan Harian |
  | dokter | Antrian, Rekam Medis, Riwayat Pasien, Laporan Harian |
  | admin | Pasien, Antrian (khusus tidak-hadir), Users, Audit Log, Pengaturan Klinik, Laporan Harian |

  Catatan: menu ini nav placeholder — route tujuannya boleh belum ada halamannya (dibangun di fitur-fitur berikutnya). Nav tetap dibangun sekarang supaya struktur shell final, tidak perlu dirombak tiap modul baru selesai.

- **Landing page** (index route `/` setelah login): **1 route, 1 komponen**, konten di-render **kondisional berdasar role** (bukan 3 route/dashboard terpisah — lihat alasan di Edge case). Isi: greeting ("Selamat datang, {nama}") + shortcut ke menu yang relevan buat role itu. **Bukan** dashboard stats/summary asli — data itu belum ada karena modul Pasien/Antrian/Admin belum dibangun.
- **Spartan primitives** — belum pernah difetch sama sekali di project ini (dikonfirmasi user). Dibutuhkan minimal: primitive untuk sidebar/nav item, avatar atau dropdown-menu (user menu), badge (`ClinicStatusIndicator` — mungkin sudah bisa reuse dari primitive Badge kalau nanti StatusBadge/PriorityBadge fetch duluan, tapi shell ini kemungkinan besar yang fetch Badge pertama kali).

  **WAJIB checklist `DESIGN.md` §10 untuk SETIAP primitive di atas**, tanpa kecuali karena ini fetch pertama project:
  1. Cek Component Registry (`DESIGN.md` §12) dulu sebelum fetch — kalau primitive itu ternyata **sudah ada** entry-nya di registry (dari sesi lain yang belum tersync ke chat ini), **jangan fetch ulang**, pakai/extend yang sudah ada.
  2. Setelah fetch (kalau memang belum ada): verifikasi primitive pakai class/token semantik (`bg-primary`, `text-foreground`, `rounded-[--radius-md]`, dst — lihat `DESIGN.md` §2–§5), **bukan** warna/radius default hardcoded dari template Spartan.
  3. Kalau template default ternyata **tidak** pakai token semantik sama sekali → **STOP**, laporkan ke user (jangan lanjut asumsi aman), ini keputusan arsitektur yang butuh konfirmasi eksplisit.
  4. Setelah sesuai/diverifikasi, tambahkan barisnya ke Component Registry (`DESIGN.md` §12) saat itu juga — kolom Status jadi `Selesai`.

## Tahapan implementasi

- **Tahap 1 (Auth infra)**: auth resolver + Signal-based auth state service + route guard + HTTP interceptor. Foundational logic, belum ada UI.
- **Tahap 2 (UI Shell)**: layout component (sidebar + header), nav per role, landing page kondisional, fetch & verifikasi Spartan primitives sesuai checklist di atas.
- **Tahap 3 (Test)**: unit test guard, interceptor, state derivation auth service, dan render kondisional landing page per role.

## Skema/struktur data

Tidak ada perubahan skema backend (backend sudah selesai). Tipe FE baru: `AuthUser { id, nama, roles[] }` di `core/auth/types` atau `core/types` — persis shape response `GET /auth/me`.

## Edge case yang perlu dihandle

- Refresh halaman staff (F5) — resolver wajib re-resolve tiap app load, jangan cache stale antar sesi browser.
- Role tidak sesuai untuk route tertentu → state "tidak punya akses" (`DESIGN.md` §9.5), bukan redirect diam-diam ke landing atau 404 generik.
- Sesi expired di tengah pemakaian (401 dari interceptor saat request apapun) → redirect `/login`, bukan silent fail atau toast doang.
- `papan-antrian` harus tetap bisa diakses tanpa sesi staff sama sekali — regresi paling gampang lolos kalau guard baru ini keliru ter-apply ke root route alih-alih di-scope ke area staff saja.
- Kenapa landing page 1 route (bukan 3 dashboard/role terpisah): struktur folder `AGENTS.md` §8 domain-based (`features/pasien`, `features/antrian`, dst), bukan role-based — bikin folder per-role bakal jadi konvensi kedua yang paralel dan membingungkan. Juga menghindari scope creep membangun 3 dashboard dengan data asli padahal modul sumber datanya (Pasien/Antrian/Admin) belum ada.

## Testing

- Guard redirect ke `/login` kalau belum authenticated.
- Guard redirect ke state "tidak punya akses" kalau role tidak sesuai kebutuhan route.
- Interceptor: response `401` memicu redirect `/login`.
- Interceptor: error response di-parse sesuai format `{ error: { code, message, requestId } }`, `message` yang dikurasi backend yang dipakai (bukan raw error).
- Auth state Signal reflect hasil `GET /auth/me` dengan benar (`nama`, `roles`).
- Landing page render konten berbeda sesuai role (assert minimal: menu shortcut yang muncul beda antara petugas/dokter/admin).

## Kriteria selesai

- Semua requirement di atas terimplementasi sesuai 3 tahap.
- Seluruh test di atas lolos (Vitest).
- User login manual sebagai tiap role dan verifikasi: sidebar nav sesuai role, landing page konten sesuai role, `papan-antrian` tetap bisa diakses tanpa login.
- Component Registry di `DESIGN.md` §12 ter-update dengan primitive baru yang difetch (nama, status `Selesai`).
