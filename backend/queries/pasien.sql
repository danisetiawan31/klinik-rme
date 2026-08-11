-- name: InsertPasien :one
INSERT INTO pasien (
    nik,
    nama,
    tanggal_lahir,
    jenis_kelamin,
    alamat,
    no_telp,
    consent_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, nik, nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at, version, deleted_at;

-- name: GetPasienByID :one
SELECT id, nik, nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at, version, deleted_at
FROM pasien
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetPasienByIDIncludingDeleted :one
SELECT id, nik, nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at, version, deleted_at
FROM pasien
WHERE id = $1;

-- name: SearchPasien :many
SELECT id, nik, nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at, version, deleted_at
FROM pasien
WHERE deleted_at IS NULL
  AND (sqlc.narg('nik')::text IS NULL OR nik = sqlc.narg('nik'))
  AND (sqlc.narg('nama')::text IS NULL OR nama ILIKE '%' || sqlc.narg('nama') || '%')
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: UpdatePasienOptimistic :one
UPDATE pasien
SET
    nik = COALESCE(sqlc.narg('nik'), nik),
    nama = COALESCE(sqlc.narg('nama'), nama),
    tanggal_lahir = COALESCE(sqlc.narg('tanggal_lahir'), tanggal_lahir),
    jenis_kelamin = COALESCE(sqlc.narg('jenis_kelamin'), jenis_kelamin),
    alamat = COALESCE(sqlc.narg('alamat'), alamat),
    no_telp = COALESCE(sqlc.narg('no_telp'), no_telp),
    version = version + 1
WHERE id = $1 AND version = $2 AND deleted_at IS NULL
RETURNING id, nik, nama, tanggal_lahir, jenis_kelamin, alamat, no_telp, consent_at, version, deleted_at;
