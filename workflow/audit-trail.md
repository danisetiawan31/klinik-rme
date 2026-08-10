# Audit Trail Infrastructure

## Konteks & tujuan

Infrastruktur audit trail tamper-evident (hash-chain) — dipakai fitur berikutnya (item #4 Pasien, item #7 Rekam Medis) untuk mencatat perubahan data bisnis. **Wajib selesai sebelum item #4 dan #7** (retrofit audit ke transaksi yang sudah ada lebih mahal daripada built-in dari awal). **Tidak ada endpoint HTTP di spec ini** — endpoint `GET /admin/audit-log*` itu item #8 backlog, di luar scope.

## Requirement

### Migration (2 tabel, sesuai ERD di docs/TDD.md)

- `audit_log`, `audit_log_tail`

### DB trigger

- Tolak `UPDATE` dan `DELETE` langsung di tabel `audit_log` (`BEFORE UPDATE OR DELETE`, `RAISE EXCEPTION`). Tamper-evident, bukan tamper-proof (batasan ini sudah disadari & diterima, tidak perlu diselesaikan lebih jauh — akses superuser DB langsung tetap di luar jangkauan trigger biasa).

### Helper hash-chain (Go, reusable, TIDAK membuka transaksi sendiri)

Fungsi generik yang akan dipanggil fitur lain (item #4, #7) DI DALAM transaksi milik caller (perubahan data bisnis sudah dilakukan di tx yang sama SEBELUM memanggil helper ini — caller yang bertanggung jawab membungkus `BEGIN...COMMIT`, bukan helper).

Parameter yang wajib diterima: context, transaksi aktif (`pgx.Tx`) milik caller, `actorUserID`, `tabelTarget`, `recordID`, `aksi` (`'create'` atau `'update'`), `beforeData` (nullable), `afterData`.

Behavior, urutan wajib:

1. Lock `audit_log_tail` baris `id=1` via `SELECT ... FOR UPDATE` (**bukan** `SKIP LOCKED` — chain butuh urutan sekuensial ketat, sama seperti pattern di docs/TDD.md).
2. Hitung `hash_entry` pakai formula di bawah (WAJIB persis, sudah diverifikasi, jangan diubah/disederhanakan).
3. `INSERT` ke `audit_log`.
4. `UPDATE audit_log_tail SET last_hash = <hash_entry baru> WHERE id = 1`.

`created_at` WAJIB digenerate di kode Go (`time.Now()`) SEBELUM dipakai baik untuk kolom `created_at` maupun sebagai bagian input hash — JANGAN pakai `DEFAULT now()` di level DB (nilai yang di-hash harus persis sama dengan nilai yang tersimpan, supaya bisa diverifikasi ulang nanti).

### Formula `hash_entry` (FINAL — wajib diimplementasikan persis seperti ini, JANGAN disederhanakan jadi string concatenation manual)

```go
type auditHashInput struct {
    PreviousHash string          `json:"previousHash"`
    TabelTarget  string          `json:"tabelTarget"`
    RecordID     int32           `json:"recordId"`
    ActorUserID  int32           `json:"actorUserId"`
    Aksi         string          `json:"aksi"`
    BeforeData   json.RawMessage `json:"beforeData"` // literal JSON null kalau aksi='create'
    AfterData    json.RawMessage `json:"afterData"`
    CreatedAt    string          `json:"createdAt"` // format RFC3339
}
// hash_entry = hex(SHA256(json.Marshal(auditHashInput{...})))
```

**Kenapa WAJIB struct JSON, bukan concat manual dengan delimiter apapun (mis. `"|"`)**: delimiter yang bisa muncul di dalam isi field (field bebas teks klinis seperti keluhan/diagnosis wajar mengandung karakter apa saja) membuka celah "pergeseran boundary" — dua pasang `(before_data, after_data)` yang isinya BEDA bisa menghasilkan string gabungan yang IDENTIK, sehingga `hash_entry` yang dihasilkan juga identik walau datanya beda. Itu merusak total tujuan tamper-evident untuk skenario tampering paling realistis (akses DB langsung). JSON aman karena delimiter terstruktur (`"`, `:`, `,`, `{`, `}`) dengan escaping eksplisit, tidak bisa ambigu oleh isi field.

`beforeData`/`afterData` pakai `json.RawMessage` (passthrough byte asli), BUKAN unmarshal-ke-map-lalu-remarshal — supaya byte yang di-hash persis sama dengan yang disimpan ke kolom JSONB.

Package lokasi: sarankan `internal/audit/` (sejajar `internal/auth/`, `internal/mailer/`) — boleh disesuaikan kalau ada alasan konkret, catat di done.md kalau berbeda.

## Tahapan implementasi

- **Tahap 1 (Migration & trigger)**: Migration `audit_log` + `audit_log_tail` sesuai skema di bawah, termasuk genesis seed. DB trigger tolak `UPDATE`/`DELETE` di `audit_log`.
- **Tahap 2 (Helper hash-chain)**: Implementasi fungsi helper sesuai requirement & formula di atas. Test unit (formula hash benar, single call, sequential calls — chain linkage) + test concurrency lawan Postgres asli (lihat section Testing).
- **Tahap 3 (Verifikasi trigger & regresi menyeluruh)**: Test trigger (lawan Postgres asli) + regresi penuh seluruh test suite project sampai sejauh ini (scaffolding + Auth & RBAC).

## Skema/struktur data
AUDIT_LOG {
int id PK
string tabel_target
int record_id
int actor_user_id FK -- REFERENCES users(id)
string aksi -- CHECK (aksi IN ('create', 'update'))
jsonb before_data -- nullable
jsonb after_data
string hash_entry
string previous_hash -- selalu terisi, dari audit_log_tail
timestamp created_at -- WAJIB diisi dari Go (time.Now()), bukan DEFAULT now()
}
AUDIT_LOG_TAIL {
int id PK -- CHECK (id = 1), singleton
string last_hash -- genesis seed = SHA256('klinik-rme-genesis'), nilai TETAP jangan diubah/random
}
Catatan: `record_id` TIDAK punya FK ke tabel manapun secara spesifik (polymorphic reference lintas `tabel_target` yang beda-beda) — ini disengaja, bukan kelalaian skema.

## Edge case yang perlu dihandle

- **Genesis seed** — migration insert 1 baris `audit_log_tail` (`id=1`, `last_hash=SHA256('klinik-rme-genesis')`) sebagai bagian migration itu sendiri, BUKAN dihasilkan runtime. Nilai genesis ini FIXED, harus persis string itu (case-sensitive), supaya verifikasi ulang chain dari awal tidak ambigu di masa depan.
- **Concurrency serialization** — dua (atau lebih) pemanggilan helper secara paralel WAJIB benar-benar serialize lewat lock `FOR UPDATE` di `audit_log_tail`. TIDAK BOLEH ada dua row `audit_log` dengan `previous_hash` yang SAMA (itu berarti chain "bercabang", bukan linear — merusak integritas seluruh chain). Test concurrency WAJIB memverifikasi ini eksplisit (lihat section Testing), bukan diasumsikan aman cuma karena pakai `FOR UPDATE`.
- **Helper tidak membuka transaksi sendiri** — kalau dipanggil tanpa transaksi aktif yang valid, itu adalah kesalahan konfigurasi di sisi CALLER (item #4/#7 nanti), bukan tanggung jawab helper untuk auto-wrap atau mentolerir.
- **`before_data` nullable** — `aksi='create'` → `before_data` NULL (representasikan sebagai JSON literal `null` di `BeforeData json.RawMessage` saat hashing, bukan Go `nil` yang berpotensi marshal beda). `aksi='update'` → `before_data` WAJIB terisi.
- **Trigger scope** — cukup blokir `UPDATE`/`DELETE` biasa lewat aplikasi (role aplikasi normal). Tidak perlu (dan tidak realistis) mencegah superuser DB yang sengaja disable trigger — itu batasan yang sudah disadari di docs/TDD.md ("tamper-evident, bukan tamper-proof"), jangan over-engineer untuk mencegah threat model itu.

## Testing

- Migration: `audit_log` & `audit_log_tail` ter-create dengan constraint benar (`CHECK` pada `aksi`, `CHECK (id=1)` pada tail, FK `actor_user_id` ke `users`, nilai genesis seed persis benar) — lawan Postgres asli via testcontainers-go.
- Trigger: `UPDATE` langsung ke `audit_log` ditolak; `DELETE` langsung ditolak — lawan Postgres asli.
- Helper — single call: `audit_log` row ter-insert dengan `hash_entry` yang match hasil hitung ulang manual pakai formula yang sama (verifikasi formula benar-benar dieksekusi persis sesuai spec, bukan diasumsikan dari deskripsi kode). `audit_log_tail.last_hash` ter-update sama dengan `hash_entry` yang baru.
- Helper — sequential calls (mis. 3x berurutan): assert `previous_hash` entry ke-2 = `hash_entry` entry ke-1, `previous_hash` entry ke-3 = `hash_entry` entry ke-2 (chain linkage benar, bukan cuma masing-masing entry valid secara terpisah).
- Helper — concurrency (WAJIB lawan Postgres asli via testcontainers-go, bukan mock): minimal 10 goroutine paralel memanggil helper ini bersamaan (transaksi masing-masing terpisah). Assert: semua insert sukses (tidak ada yang gagal/ke-drop diam-diam), TIDAK ADA dua row dengan `previous_hash` yang sama, dan rantai `previous_hash → hash_entry` membentuk satu urutan linear valid dari genesis sampai entry terakhir (tidak ada cabang/gap).
- Helper — robustness formula hash (WAJIB, bukan opsional, karena ini persis skenario yang mendasari kenapa formula ini dipilih): buat 2 pasang `(before_data, after_data)` berbeda isi yang KALAU di-concat manual pakai delimiter tertentu akan menghasilkan string sama (mis. `before="A|B", after="C"` vs `before="A", after="B|C"`) — assert `hash_entry` yang dihasilkan BERBEDA di antara keduanya (bukan collide).
- Regresi seluruh test suite yang sudah ada (scaffolding Tahap 1-3, Auth & RBAC Tahap 1-5).

## Kriteria selesai
Seluruh 3 tahap selesai, seluruh skenario test di atas PASS (termasuk concurrency & robustness formula lawan Postgres asli), `go vet` lolos, regresi tidak ada yang kebreak.