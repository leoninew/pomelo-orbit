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

-- name: CountRoles :one
SELECT COUNT(*)
FROM role
WHERE (sqlc.narg(search_pattern) IS NULL
  OR LOWER(code) LIKE sqlc.narg(search_pattern)
  OR LOWER(name) LIKE sqlc.narg(search_pattern)
  OR LOWER(COALESCE(description, '')) LIKE sqlc.narg(search_pattern));

-- name: ListRoles :many
SELECT id, code, name, description, created_at, updated_at
FROM role
WHERE (sqlc.narg(search_pattern) IS NULL
  OR LOWER(code) LIKE sqlc.narg(search_pattern)
  OR LOWER(name) LIKE sqlc.narg(search_pattern)
  OR LOWER(COALESCE(description, '')) LIKE sqlc.narg(search_pattern))
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: ListPermissions :many
SELECT id, code, name, description, created_at, updated_at
FROM permission
ORDER BY code;

-- name: PermissionCodesByCodes :many
SELECT code
FROM permission
WHERE code IN (sqlc.slice('codes'));

-- name: RolePermissionCodesByRoleIds :many
SELECT role_permission.role_id, permission.code
FROM role_permission
JOIN permission ON permission.id = role_permission.permission_id
WHERE role_permission.role_id IN (sqlc.slice('role_ids'))
ORDER BY role_permission.role_id, permission.code;

-- name: CreateRole :exec
INSERT INTO role (id, code, name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateRole :exec
UPDATE role
SET code = ?, name = ?, description = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteRole :exec
DELETE FROM role
WHERE id = ?;

-- name: PermissionIDByCode :one
SELECT id
FROM permission
WHERE code = ?;

-- name: DeleteRolePermissions :exec
DELETE FROM role_permission
WHERE role_id = ?;

-- name: InsertRolePermission :exec
INSERT INTO role_permission (role_id, permission_id, created_at)
VALUES (?, ?, ?);
