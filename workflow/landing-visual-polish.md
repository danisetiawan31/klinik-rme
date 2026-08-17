# Redesign Visual Landing (Hero, Module Cards, Ringkasan Hari Ini)

## Konteks & tujuan

Landing terasa padat karena nilai spacing ad hoc tidak konsisten mengacu ke
token `docs/DESIGN.md` yang sudah ada, dan card besar (`rounded-3xl`) tidak
punya token radius resmi (token cuma sampai `--radius-lg`/12px). Ilustrasi
hero tidak full-bleed karena aset lama punya whitespace internal. Card
"Ringkasan Hari Ini" duplikat data dengan strip hero tanpa nilai tambah.
Scope landing dulu sebagai referensi pola — halaman lain menyusul terpisah.

## Requirement

1. **Token radius baru**: tambah `--radius-xl: 24px` di `docs/DESIGN.md` +
   `global.css` (`@theme`), khusus untuk card level hero/feature (bukan
   pengganti `--radius-lg` yang tetap dipakai modal/panel standar).
2. **Audit & perbaiki pemakaian token spacing existing** di seluruh section
   landing (hero, 4 module card, card antrian hari ini, card ringkasan hari
   ini): ganti nilai ad hoc (`p-3.5`, `mt-0.5`, dst) jadi mengacu skala
   `--space-*` yang sudah terdefinisi. Naikkan padding card level hero
   (saat ini `p-6`/`--space-6`) ke `--space-8` (card besar) — proporsional
   terhadap `--radius-xl` yang baru.
3. **Hilangkan 1 layer nested-card** di strip mini-metrics hero: buang
   wrapper `bg-card/50` di sekeliling 4 tile metric, ganti jadi divider
   (`border-t`) + grid langsung di dalam hero card. Tile metric individual
   tetap pakai card style (bukan nested ganda lagi).
4. **Aset ilustrasi hero full-bleed** — dikerjakan Antigravity via built-in
   image generation (Nano Banana):
   - Investigasi dulu gaya visual dari `docs/DESIGN.md` (palet warna, token
     semantik) & aset lama sebelum generate — JANGAN generate dari asumsi
     bebas.
   - Generate 2-3 kandidat (wide ~16:9, tanpa whitespace internal, subjek
     dekat tepi frame), simpan sebagai Artifact/screenshot untuk direview.
   - **Tunggu approval kandidat sebelum lanjut** (referensi visual baru
     yang dikunci — exception `AGENTS.md` §10, review di titik ini, bukan
     di akhir fitur).
   - Mobile (`< sm`): sembunyikan ilustrasi total (hero jadi teks +
     gradient saja), muncul dari `sm:` ke atas dengan `object-position`
     yang jaga subjek tetap dalam frame di semua breakpoint (`sm`/`md`/`lg`).
5. **Redesign card "Ringkasan Hari Ini"** — sumber data pindah dari
   `AntrianService` (live) ke `LaporanService.getLaporanHarian()` (rekap
   resmi), supaya secara struktural berbeda dari strip hero (bukan cuma
   beda visual):
   - Slot 2x2 grid: **Total Kunjungan** (`totalKunjungan`), **Prioritas**
     (tetap dari `antrianList` — field ini tidak ada di `LaporanService`),
     **Tidak Hadir** (`totalTidakHadir`, BARU — ganti slot "Menunggu" yang
     dihapus karena semantiknya cuma live-state, sudah terwakili di hero),
     **Selesai** (`totalSelesai`).
   - Tambah badge tren vs kemarin: panggil `LaporanService.getLaporanHarian()`
     untuk hari ini & kemarin secara paralel (`forkJoin`, service sudah
     stateless/reusable — tidak perlu ubah arsitektur), bandingkan
     `totalKunjungan`. Kalau data kemarin `totalKunjungan: 0` (klinik baru
     buka, endpoint tetap return 200 dengan angka 0 — bukan 404), tampilkan
     tren sebagai "–" bukan divide-by-zero/NaN.
   - Progress bar horizontal → radial/donut progress SVG custom
     (`stroke-dasharray`/`stroke-dashoffset`, animasi via CSS transition) —
     **TIDAK install chart library baru** (tidak ada yang terpasang saat
     ini, sesuai `AGENTS.md` §9 hindari dependency baru tanpa alasan kuat).
     Progress dihitung dari `totalSelesai / totalKunjungan` (sumber sama,
     `LaporanService`, konsisten 1 sumber untuk seluruh card ini).
   - Pertimbangkan jadikan radial progress ini `shared/ui/radial-progress/`
     (reusable) daripada inline sekali pakai — keputusan detail teknis,
     bebas diputuskan Antigravity asal dicatat di laporan tahap (`AGENTS.md`
     §4).
6. **Helper tanggal baru**: tambah `getJakartaYesterdayISODate()` (atau
   `subtractDaysISO(dateStr, n)` generik) di `date.utils.ts` — belum ada
   saat ini.

## Tahapan implementasi

- **Tahap 1** (Token & Hero): requirement 1, 2 (bagian hero), 3, 4.
  Termasuk gate approval visual ilustrasi (lihat requirement 4).
- **Tahap 2** (Module Cards, spacing sisanya, Ringkasan Hari Ini):
  requirement 2 (sisanya), 5, 6.
- **Tahap 3** (Test & Regresi): unit test kalkulasi tren (naik/turun/data
  kemarin kosong), regresi Vitest landing, self-check grep hardcode warna
  (`AGENTS.md` §8).

## Skema/struktur data

Tidak ada perubahan backend/skema. Requirement 5 murni memakai
`GET /laporan/harian?tanggal=` yang sudah ada (dipanggil 2x paralel: hari
ini & kemarin, via `forkJoin`).

## Edge case

- `totalKunjungan` kemarin = 0 → tren tampil "–"/teks netral, bukan
  NaN/Infinity.
- Ilustrasi hero: transisi show/hide persis di boundary `sm` tidak boleh
  kedip/patah — dicek manual, dicatat di laporan Tahap 1.
- Kandidat gambar hasil generate belum ada gaya yang cocok setelah 2-3
  percobaan → STOP, laporkan ke user, jangan lanjut Tahap 1 dengan aset
  seadanya.

## Testing

- Unit test kalkulasi delta tren (naik, turun, kemarin 0 data).
- Vitest existing landing test tetap PASS (update assertion yang
  bergantung markup lama, termasuk yang cek slot "Menunggu" di card
  Ringkasan — sekarang jadi "Tidak Hadir").
- Manual: review visual mobile/tablet/desktop untuk hero (1x di akhir
  fitur per `AGENTS.md` §10, terpisah dari gate approval gambar di Tahap 1).

## Kriteria selesai

3 tahap dilaporkan & di-ACC user, test otomatis PASS, review visual manual
dilakukan sebelum masuk `done_fe.md`.
