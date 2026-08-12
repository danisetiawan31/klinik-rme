# Admin — Manajemen User & Audit Log

## Konteks & tujuan

Melengkapi kemampuan admin: list & edit user, resend invite, atur roles (dengan proteksi last-admin & minimal 1 role), serta baca audit log (list + detail) untuk investigasi. `POST /admin/users` sudah dibangun di fitur Auth & RBAC Foundation — tidak dibangun ulang di sini.

## Requirement

- `GET /admin/users?page=&limit=` [admin] — list user + roles, pagination wajib.
- `POST /admin/users/:id/resend-invite` [admin] — generate token invite baru, invalidasi token invite lama yang belum dipakai milik user itu (`consumed_at IS NULL`, `type='invite'`), kirim ulang via Resend. Tolak (400) kalau `password_hash` user sudah terisi (bukan lagi status invite pending — arahkan ke `forgot-password` sesuai `TDD.md`).
- `PATCH /admin/users/:id` [admin] — update `nama`/`email` (partial). 409 kalau email sudah dipakai user lain (unique constraint, reactive check).
- `PATCH /admin/users/:id/roles` [admin] — replace penuh array roles.
  - Validasi mutual exclusion `admin`/`dokter` (400 kalau keduanya diminta sekaligus).
  - Validasi minimal 1 role (400 kalau `roles: []`).
  - Validasi last-admin: tolak (400) kalau hasil akhir bikin 0 user dengan role `admin` tersisa di sistem (termasuk kasus self-demote).
  - Tidak perlu invalidasi session — RBAC middleware query `user_roles` fresh tiap request (sudah diverifikasi via investigasi Antigravity), efek langsung tanpa re-login.
- `GET /admin/audit-log?tabelTarget=&recordId=&actorId=&page=&limit=` [admin] — list ringkas (tanpa `beforeData`/`afterData`), semua filter optional & bisa dikombinasikan.
- `GET /admin/audit-log/:id` [admin] — detail penuh termasuk `beforeData`/`afterData`.

## Tahapan implementasi

- Tahap 1 (User management dasar): `GET /admin/users`, `PATCH /admin/users/:id`, `POST /admin/users/:id/resend-invite`
- Tahap 2 (Roles management): `PATCH /admin/users/:id/roles` + 3 validasi (mutual exclusion, minimal 1 role, last-admin guard)
- Tahap 3 (Audit log read): `GET /admin/audit-log`, `GET /admin/audit-log/:id`
- Tahap 4 (Integrasi & regresi menyeluruh): test E2E lintas 6 endpoint + regresi penuh suite

## Skema/struktur data

Tidak ada migration baru. Full reuse tabel `users`, `user_roles`, `password_tokens`, `audit_log` yang sudah ada.

## Edge case yang perlu dihandle

- Resend-invite ke user yang statusnya sudah aktif (`password_hash` terisi) → 400, bukan generate token baru (token invite tidak relevan lagi untuk akun aktif).
- Resend-invite generate token baru → token invite lama yang belum dipakai untuk user itu wajib diinvalidasi (`consumed_at = now()`) di transaksi yang sama, supaya tidak ada 2 token invite valid berbarengan.
- PATCH roles: mutual exclusion admin/dokter, minimal 1 role, dan last-admin guard — divalidasi sebagai pre-check sebelum write (aman, operasi jarang & bukan concurrency-sensitive, konsisten sama alasan di `TDD.md` soal validasi role).
- PATCH /admin/users/:id email duplikat → 409, andalkan constraint DB (reactive, bukan preemptive SELECT) sesuai pola `AGENTS.md` §7.
- Audit log list TIDAK PERNAH include `beforeData`/`afterData` (sudah ditegaskan di `api-contract.md`) — cuma endpoint detail yang boleh.

## Testing

- `GET /admin/users`: pagination benar, 403 untuk role non-admin.
- `PATCH /admin/users/:id`: sukses update nama/email, 409 email duplikat, 404 user tidak ada.
- `POST /admin/users/:id/resend-invite`: token baru ter-generate & lama ter-invalidasi (assert token lama gagal dipakai setelah resend), 400 kalau user sudah aktif (`password_hash` terisi), 404 user tidak ada.
- `PATCH /admin/users/:id/roles`: sukses ganti roles, 400 mutual exclusion admin+dokter, 400 roles kosong, 400 last-admin guard (self-demote maupun demote admin lain yang jadi satu-satunya), sukses kalau masih ada admin lain tersisa.
- `GET /admin/audit-log`: filter kombinasi (`tabelTarget`+`recordId`+`actorId`), pagination, tanpa `beforeData`/`afterData` di response.
- `GET /admin/audit-log/:id`: detail penuh termasuk `beforeData`/`afterData`, 404 kalau id tidak ada.
- RBAC: seluruh 6 endpoint reject role non-admin (403).
- E2E lifecycle (Tahap 4): skenario berurutan mencakup semua endpoint di atas dalam 1 alur realistis + regresi penuh suite (`go test -v -p 1 ./...` & `go vet ./...`).

## Kriteria selesai

Semua endpoint di atas berjalan sesuai `api-contract.md`, seluruh skenario testing lolos, `go vet ./...` bersih, dicek ulang manual oleh user sebelum push.
