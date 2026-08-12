-- name: GetUserByEmail :one
SELECT id, nama, email, password_hash
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, nama, email, password_hash
FROM users
WHERE id = $1;

-- name: UpdateUserPasswordHash :exec
UPDATE users
SET password_hash = $1
WHERE id = $2;

-- name: CreateUser :one
INSERT INTO users (nama, email)
VALUES ($1, $2)
RETURNING id, nama, email;

-- name: ListUsersWithRoles :many
SELECT u.id, u.nama, u.email,
       COALESCE(array_agg(ur.role ORDER BY ur.role) FILTER (WHERE ur.role IS NOT NULL), '{}')::text[] AS roles
FROM users u
LEFT JOIN user_roles ur ON u.id = ur.user_id
GROUP BY u.id, u.nama, u.email
ORDER BY u.id ASC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: UpdateUserBiodata :one
UPDATE users
SET
    nama = COALESCE(sqlc.narg('nama'), nama),
    email = COALESCE(sqlc.narg('email'), email)
WHERE id = $1
RETURNING id, nama, email;


