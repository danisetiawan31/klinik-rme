-- name: InsertSession :exec
INSERT INTO sessions (id_hash, user_id, created_at, expires_at, absolute_expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetSessionByIDHash :one
SELECT id_hash, user_id, created_at, expires_at, absolute_expires_at
FROM sessions
WHERE id_hash = $1;

-- name: DeleteSessionByIDHash :exec
DELETE FROM sessions
WHERE id_hash = $1;

-- name: UpdateSessionExpiresAt :exec
UPDATE sessions
SET expires_at = $1
WHERE id_hash = $2;
