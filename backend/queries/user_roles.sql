-- name: GetRolesByUserID :many
SELECT role
FROM user_roles
WHERE user_id = $1;

-- name: InsertUserRole :exec
INSERT INTO user_roles (user_id, role)
VALUES ($1, $2);

-- name: DeleteUserRolesByUserID :exec
DELETE FROM user_roles
WHERE user_id = $1;

-- name: CountUsersWithRole :one
SELECT COUNT(*)
FROM user_roles
WHERE role = $1;

