-- name: InsertTindakan :one
INSERT INTO tindakan (
    rekam_medis_id,
    jenis,
    deskripsi
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetTindakanByRekamMedisID :many
SELECT * FROM tindakan
WHERE rekam_medis_id = $1
ORDER BY id ASC;
