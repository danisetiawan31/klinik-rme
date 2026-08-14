# Antrian (Staff-facing)

## Konteks & tujuan

Dashboard antrian real-time untuk staff: petugas melihat + mendaftarkan pasien ke antrian hari ini (lewat halaman detail pasien), dokter memanggil/melewati pasien, dokter+admin menandai tidak hadir. Backend sudah lengkap (item 5 & 6, "Selesai Penuh"). Fitur ini consumer PERTAMA dari `RealtimeService` (item 13) — invalidation ping via WS memicu refetch REST, bukan pembawa data.

`AntrianDashboardComponent` yang ada sekarang (placeholder dari sesi migrasi styling) di-REPLACE TOTAL, bukan di-extend.

## Requirement

**Tahap 1 — List + Realtime**

- Replace `AntrianDashboardComponent`: fetch `GET /klinik/:id/antrian` on load, tampilkan list kunjungan hari ini.
- Sort tampilan client-side: `isPriority DESC, skipCount ASC, nomorAntrian ASC` (selaras urutan panggil backend — bukan ekspektasi urutan dari response API, itu improvisasi tampilan, bukan kontrak).
- Integrasi `RealtimeService`: `connect()` saat komponen init, `disconnect()` via `DestroyRef.onDestroy()`. `effect()` refetch REST dipicu oleh **DUA** kondisi: perubahan `lastUpdateAt`, ATAU `connectionStatus` bertransisi ke `'connected'` (nutup gap update yang lewat pas disconnect, sesuai TDD.md).
- `StatusBadge` & `PriorityBadge` (composed component, wrap `HlmBadge`) — bangun sesuai token `docs/design.md` §9.4 (4 varian status + priority badge dengan kontras WCAG AA, teks `text-foreground` bukan `text-secondary`). Daftarkan ke Component Registry.
- Indikator klinik buka/tutup — reuse pola `isKlinikBuka()` dari `KlinikService`, reactive (recompute juga saat `lastUpdateAt` berubah, bukan cuma sekali di mount).
- Empty state: antrian kosong hari ini → pesan jelas, bukan blank.

**Tahap 2 — Aksi dokter/petugas**

- RBAC per-tombol (bukan per-halaman):
  - **Dokter**: tombol global "Panggil Berikutnya" (`POST /klinik/:id/panggil-berikutnya`); tombol per-baris "Lewati" HANYA muncul di entri berstatus `dipanggil` (`POST /kunjungan/:id/lewati`).
  - **Dokter + Admin**: tombol per-baris "Tandai Tidak Hadir" HANYA muncul di entri berstatus `menunggu` (`POST /kunjungan/:id/tidak-hadir`).
  - **Petugas**: view-only di halaman ini, tanpa tombol aksi apa pun.
- "Panggil Berikutnya" respons 204 (antrian kosong) → toast info eksplisit "Antrian kosong", bukan diam.
- "Tandai Tidak Hadir" WAJIB dialog konfirmasi sebelum submit (aksi final, tidak ada undo — `tidak_hadir` cuma diset manual saat penutupan hari, TDD.md). "Lewati" TIDAK perlu konfirmasi (reversible, balik ke `menunggu`).
- Semua aksi sukses → toast sukses; error → toast error dari `error.message` terkurasi backend.

**Tahap 3 — Pendaftaran via PasienDetail**

- Tombol "Daftarkan ke Antrian" di `PasienDetailComponent`, di sebelah tombol "Edit" existing — kondisional `petugas`/`admin` (pola `@if` + `hasRole()` yang sama).
- Dialog/form: toggle `isPriority`, kalau dicentang → `priorityReason` WAJIB diisi di client (validasi lebih ketat dari backend yang nullable — sama semangat validasi NIK di form Pasien).
- Tombol "Daftarkan ke Antrian" DISABLED proaktif kalau `isKlinikBuka() === false` (reuse `KlinikService`), dengan tooltip/keterangan alasan — jangan nunggu 400 `KLINIK_TUTUP` dari server sebagai satu-satunya sinyal.
- `POST /kunjungan` sukses → toast sukses (nomorAntrian didapat), tetap di halaman detail (gak perlu navigasi).
- Method baru di `antrian.service.ts` (BUKAN nambah ke `pasien.service.ts` — beda domain), tapi dipanggil dari komponen Pasien (cross-module call itu wajar, service boleh diimport lintas fitur).

## Skema/struktur data

`features/antrian/antrian.types.ts`:

```typescript
interface KunjunganListItem {
  id: number;
  nomorAntrian: number;
  status: "menunggu" | "dipanggil" | "selesai" | "tidak_hadir";
  isPriority: boolean;
  priorityReason?: string | null;
  pasienNama: string;
  skipCount?: number;
}
interface CreateKunjunganRequest {
  pasienId: number;
  isPriority?: boolean;
  priorityReason?: string;
}
interface CreateKunjunganResponse {
  id: number;
  nomorAntrian: number;
  status: "menunggu";
  tanggalKunjungan: string;
}
interface PanggilBerikutnyaResponse {
  id: number;
  nomorAntrian: number;
  pasienNama: string;
  dokterId: number;
  dipanggilAt: string;
}
```

`features/antrian/antrian.service.ts` — method: `getAntrian(klinikId)`, `create(payload)`, `panggilBerikutnya(klinikId)`, `lewati(kunjunganId)`, `tidakHadir(kunjunganId)`.

## Edge case

- 204 dari panggil-berikutnya → toast info, bukan error.
- "Lewati" dipanggil saat status BUKAN `dipanggil` → backend 409, tampilkan toast error, refetch list (state kemungkinan udah berubah dari staff lain).
- `POST /kunjungan` kena `KLINIK_TUTUP` 400 walau tombol udah proaktif disabled (race: klinik tutup tepat saat submit) → toast error jelas, jangan crash.
- WebSocket disconnect lama (reconnecting) → tetap tampilkan data REST terakhir, indikasikan status koneksi (reuse `connectionStatus` signal — bisa small indicator, gak perlu blocking UI).
- `priorityReason` kosong padahal `isPriority` dicentang → blok submit di client, pesan jelas.

## Testing

- RBAC per tombol: assert petugas gak lihat tombol aksi apa pun; dokter lihat Panggil+Lewati(kondisional dipanggil)+TidakHadir(kondisional menunggu); admin cuma lihat TidakHadir(kondisional menunggu).
- Sort order list sesuai `isPriority DESC, skipCount ASC, nomorAntrian ASC`.
- `effect()` trigger refetch: assert dipicu saat `lastUpdateAt` berubah DAN saat `connectionStatus` transisi ke `'connected'`.
- Dialog konfirmasi Tidak Hadir: submit hanya terjadi setelah konfirmasi, batal → tidak ada API call.
- 204 panggil-berikutnya → toast info spesifik, bukan toast error generik.
- Form daftar antrian: `priorityReason` wajib kalau `isPriority` true; tombol disabled saat `isKlinikBuka() === false`.
- Lifecycle: `disconnect()` terpanggil saat komponen destroy (no memory leak).

## Kriteria selesai

3 tahap selesai, seluruh skenario testing lolos (Vitest), regresi frontend existing tetap hijau, dicek manual user (terutama interaksi realtime — buka 2 tab, verifikasi update lintas tab) sebelum masuk `done_fe.md`.
