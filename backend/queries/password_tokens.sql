-- name: InsertPasswordToken :exec
INSERT INTO password_tokens (token_hash, user_id, type, expires_at)
VALUES ($1, $2, $3, $4);

-- name: ConsumePasswordToken :one
UPDATE password_tokens
SET consumed_at = now()
WHERE token_hash = $1
  AND consumed_at IS NULL
  AND expires_at > now()
RETURNING user_id, type;

-- name: GetActiveInviteTokenByUserID :one
SELECT token_hash, user_id, type, expires_at, created_at
FROM password_tokens
WHERE user_id = $1
  AND type = 'invite'
  AND consumed_at IS NULL
  AND expires_at > now()
LIMIT 1;

-- name: InvalidateActiveInviteTokensByUserID :exec
UPDATE password_tokens
SET consumed_at = now()
WHERE user_id = $1
  AND type = 'invite'
  AND consumed_at IS NULL;

