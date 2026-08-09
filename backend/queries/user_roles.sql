-- name: GetRolesByUserID :many
SELECT role
FROM user_roles
WHERE user_id = $1;
