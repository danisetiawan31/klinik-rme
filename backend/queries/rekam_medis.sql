-- name: InsertRekamMedis :one
INSERT INTO rekam_medis (
    kunjungan_id,
    dokter_id,
    keluhan,
    hasil_pemeriksaan,
    is_addendum,
    addendum_of,
    alasan_addendum
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetRekamMedisByID :one
SELECT * FROM rekam_medis
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetLeafRekamMedisByKunjunganID :one
SELECT r.* FROM rekam_medis r
WHERE r.kunjungan_id = $1
  AND r.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM rekam_medis r2
      WHERE r2.addendum_of = r.id AND r2.deleted_at IS NULL
  )
ORDER BY r.created_at DESC
LIMIT 1;

-- name: ListLeafRekamMedisWithKunjunganByPasienID :many
SELECT
    r.id AS rekam_medis_id,
    r.kunjungan_id,
    r.dokter_id,
    r.keluhan,
    r.hasil_pemeriksaan,
    r.is_addendum,
    r.addendum_of,
    r.alasan_addendum,
    r.created_at AS rekam_medis_created_at,
    k.tanggal_kunjungan,
    k.nomor_antrian,
    k.status AS kunjungan_status
FROM kunjungan k
JOIN rekam_medis r ON k.id = r.kunjungan_id
WHERE k.pasien_id = $1
  AND r.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM rekam_medis r2
      WHERE r2.addendum_of = r.id AND r2.deleted_at IS NULL
  )
ORDER BY k.tanggal_kunjungan DESC, k.created_at DESC;
