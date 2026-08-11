-- name: InsertDiagnosis :one
INSERT INTO diagnosis (
    rekam_medis_id,
    kode_icd,
    deskripsi
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetDiagnosisByRekamMedisID :many
SELECT * FROM diagnosis
WHERE rekam_medis_id = $1
ORDER BY id ASC;
