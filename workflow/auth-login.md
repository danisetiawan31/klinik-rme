# Auth: Login & Fondasi Guard

## Konteks & tujuan

Halaman Login adalah entry point sistem staff (petugas, dokter, admin) sekaligus
tempat pertama kali infrastruktur auth frontend dibangun: interceptor exception,
auth guard, root resolver, dan pola redirect role-based yang akan dipakai ulang
oleh seluruh route staff berikutnya (Pasien, Antrian, Rekam Medis, Admin).

## Requirement

- Form Login: field Email + Password (Password pakai komponen `SensitiveValue`
  mode `input`, toggle show/hide ikon mata), submit ke `POST /auth/login`
  dengan `withCredentials: true`.
- Sukses: simpan `{ id, nama, roles[] }` ke auth state (Signal), redirect ke
  placeholder dashboard sesuai role. Kalau user punya >1 role (kombinasi selain
  admin+dokter, yang sudah dicegah di backend), pakai prioritas landing:
  admin > dokter > petugas — ini murni default UX navigasi, bukan aturan
  keamanan.
- Gagal (`401` / `INVALID_CREDENTIALS`): toast top-center, pesan generik
  persis dari response ("Email atau password salah"), `aria-live="assertive"`,
  dismissible (ikon X). Semua skenario gagal (kredensial salah, akun belum
  aktivasi) pakai pesan sama — tidak ada state UI yang membedakan.
- Loading: tombol submit disabled + spinner, seluruh field form disabled
  selama request berjalan.
- Link "Lupa password?" tetap tampil tapi non-fungsional di iterasi ini
  (halaman `/forgot-password` scope terpisah, backlog item 11 lanjutan).
- Interceptor: request ke `/auth/login` DIKECUALIKAN dari auto-redirect 401
  global (401 di endpoint ini = kredensial salah, ditangani komponen Login
  sendiri — bukan sesi habis).
- Root resolver staff (`provideAppInitializer`/resolver, discope ke root
  route staff saja — BUKAN global) resolve `GET /auth/me` sebelum guard
  dievaluasi. Route `papan-antrian` tidak ikut resolver ini sama sekali.
- Role guard (`CanActivateFn`) baca role dari auth state hasil resolver di atas.
- Placeholder dashboard: 1 halaman generik per landing target
  (Antrian-role: petugas & dokter / Admin-role: admin), isi minimal
  "Selamat datang, {nama}" + role aktif + tombol Logout — jadi landing
  sementara sampai halaman domain sungguhan (backlog 15, 18) dibangun.

## Tahapan implementasi

- Tahap 1 (Service & state layer): `AuthService` (login, logout, currentUser
  signal), root resolver staff, role guard, interceptor exception untuk
  `/auth/login`.
- Tahap 2 (UI): Halaman Login (3 state: default/error/loading sesuai mockup
  yang sudah direview), placeholder dashboard per landing target, routing
  config penghubung semuanya.
- Tahap 3 (Test): sesuai section Testing di bawah.

**Catatan review visual**: halaman ini adalah referensi visual pertama app
staff (belum ada halaman lain sebelumnya) — sesuai pengecualian di `AGENTS.md`
§10, review visual manual boleh diminta di akhir Tahap 2, tidak perlu nunggu
sampai fitur lain juga selesai.

## Edge case yang perlu dihandle

- 401 dari endpoint SELAIN `/auth/login` tetap trigger redirect global
  (sesi habis) — pembeda logic interceptor berdasarkan URL request, bukan
  status code semata.
- User refresh browser di route staff manapun (bukan `/login`) — resolver
  wajib selesai dulu sebelum guard dievaluasi, supaya user yang sebenarnya
  masih login tidak ke-redirect keliru (race condition, `AGENTS.md` §8).
- `password_hash` masih null (akun belum aktivasi) — treated sama seperti
  kredensial salah biasa, tanpa state UI berbeda (sudah dikonfirmasi lewat
  investigasi Antigravity sebelumnya).

## Testing

- `AuthService.login()` sukses → currentUser signal terisi sesuai response.
- `AuthService.login()` gagal (401/INVALID_CREDENTIALS) → currentUser tetap
  null, error message ter-set dari response.
- Interceptor: 401 dari `/auth/login` TIDAK redirect; 401 dari endpoint lain
  redirect ke `/login`.
- Guard: user tanpa role sesuai → ditolak; role sesuai → lolos.
- Resolver: guard tidak dievaluasi sebelum resolver auth staff selesai.
- Komponen Login: render 3 state (default, error dengan toast tampil,
  loading dengan form disabled).
- Redirect priority: user dengan kombinasi role (misal petugas+dokter) →
  landing sesuai prioritas admin > dokter > petugas.

## Kriteria selesai

Semua test di atas lolos; login sukses mengarahkan ke placeholder dashboard
sesuai role; login gagal menampilkan toast generik tanpa memicu redirect
loop; refresh halaman staff yang sudah login tidak salah redirect ke
`/login`; direview visual manual oleh user di akhir Tahap 2 (lihat catatan
review visual di atas).
