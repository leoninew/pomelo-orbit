-- name: RoleByID :one
SELECT id, code, name, description, created_at, updated_at
FROM role
WHERE id = ?;

-- name: RoleByCode :one
SELECT id, code, name, description, created_at, updated_at
FROM role
WHERE code = ?;

-- name: RoleByName :one
SELECT id, code, name, description, created_at, updated_at
FROM role
WHERE name = ?;

-- name: ListPermissions :many
SELECT id, code, name, description, created_at, updated_at
FROM permission
ORDER BY code;

-- name: DeleteRole :exec
DELETE FROM role
WHERE id = ?;

-- name: PermissionIDByCode :one
SELECT id
FROM permission
WHERE code = ?;
