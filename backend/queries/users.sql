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
