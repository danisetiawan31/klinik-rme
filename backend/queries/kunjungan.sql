-- name: InsertKunjungan :one
INSERT INTO kunjungan (
    pasien_id, klinik_id, dokter_id, tanggal_kunjungan, nomor_antrian, is_priority, priority_reason, skip_count, status
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING id, pasien_id, klinik_id, dokter_id, tanggal_kunjungan, nomor_antrian, is_priority, priority_reason, skip_count, status, dipanggil_at, created_at;

-- name: GetKunjunganByID :one
SELECT id, pasien_id, klinik_id, dokter_id, tanggal_kunjungan, nomor_antrian, is_priority, priority_reason, skip_count, status, dipanggil_at, created_at
FROM kunjungan
WHERE id = $1;

-- name: ListKunjunganByKlinikAndTanggal :many
SELECT id, pasien_id, klinik_id, dokter_id, tanggal_kunjungan, nomor_antrian, is_priority, priority_reason, skip_count, status, dipanggil_at, created_at
FROM kunjungan
WHERE klinik_id = $1 AND tanggal_kunjungan = $2
ORDER BY nomor_antrian ASC;

-- name: ListKunjunganWithPasienNamaByKlinikAndTanggal :many
SELECT k.id, k.nomor_antrian, k.status, k.is_priority, p.nama AS pasien_nama
FROM kunjungan k
JOIN pasien p ON k.pasien_id = p.id
WHERE k.klinik_id = $1 AND k.tanggal_kunjungan = $2
ORDER BY k.nomor_antrian ASC;

-- name: ClaimNextKunjungan :one
UPDATE kunjungan
SET status = 'dipanggil', dipanggil_at = now(), dokter_id = $1
WHERE id = (
  SELECT k.id FROM kunjungan k
  WHERE k.klinik_id = $2 AND k.tanggal_kunjungan = $3 AND k.status = 'menunggu'
  ORDER BY k.is_priority DESC, k.skip_count ASC, k.nomor_antrian ASC
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
RETURNING id, pasien_id, klinik_id, dokter_id, tanggal_kunjungan, nomor_antrian, is_priority, priority_reason, skip_count, status, dipanggil_at, created_at;

-- name: UpdateKunjunganSkip :one
UPDATE kunjungan
SET status = 'menunggu', skip_count = skip_count + 1
WHERE id = $1 AND status = 'dipanggil'
RETURNING id, pasien_id, klinik_id, dokter_id, tanggal_kunjungan, nomor_antrian, is_priority, priority_reason, skip_count, status, dipanggil_at, created_at;

-- name: UpdateKunjunganTidakHadir :one
UPDATE kunjungan
SET status = 'tidak_hadir'
WHERE id = $1 AND status IN ('menunggu', 'dipanggil')
RETURNING id, pasien_id, klinik_id, dokter_id, tanggal_kunjungan, nomor_antrian, is_priority, priority_reason, skip_count, status, dipanggil_at, created_at;
