-- name: CountKlinik :one
SELECT COUNT(*) FROM klinik;

-- name: InsertKlinik :one
INSERT INTO klinik (nama, jam_buka, jam_tutup)
VALUES ($1, $2, $3)
RETURNING id, nama, jam_buka, jam_tutup;

-- name: GetKlinikByID :one
SELECT id, nama, jam_buka, jam_tutup
FROM klinik
WHERE id = $1;

-- name: GetSingleKlinik :one
SELECT id, nama, jam_buka, jam_tutup
FROM klinik
LIMIT 1;

-- name: UpdateKlinikDisplayTokenHash :one
UPDATE klinik
SET display_token_hash = $1
WHERE id = $2
RETURNING *;

-- name: GetKlinikDisplayTokenHash :one
SELECT display_token_hash
FROM klinik
WHERE id = $1;

