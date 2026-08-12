-- name: LockAuditLogTail :one
SELECT last_hash
FROM audit_log_tail
WHERE id = 1
FOR UPDATE;

-- name: InsertAuditLog :exec
INSERT INTO audit_log (
    tabel_target,
    record_id,
    actor_user_id,
    aksi,
    before_data,
    after_data,
    hash_entry,
    previous_hash,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- name: UpdateAuditLogTailHash :exec
UPDATE audit_log_tail
SET last_hash = $1
WHERE id = 1;

-- name: ListAuditLogs :many
SELECT id, tabel_target, record_id, actor_user_id, aksi, created_at
FROM audit_log
WHERE (sqlc.narg('tabel_target')::text IS NULL OR tabel_target = sqlc.narg('tabel_target'))
  AND (sqlc.narg('record_id')::int IS NULL OR record_id = sqlc.narg('record_id'))
  AND (sqlc.narg('actor_user_id')::int IS NULL OR actor_user_id = sqlc.narg('actor_user_id'))
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: GetAuditLogByID :one
SELECT id, tabel_target, record_id, actor_user_id, aksi, before_data, after_data, hash_entry, created_at
FROM audit_log
WHERE id = $1;

