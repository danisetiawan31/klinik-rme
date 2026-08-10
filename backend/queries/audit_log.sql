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
