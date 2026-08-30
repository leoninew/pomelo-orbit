-- name: UserByUsername :one
SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
       email, auth_source, created_at, updated_at, last_login_at
FROM `user`
WHERE username = ?;

-- name: UserByID :one
SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
       email, auth_source, created_at, updated_at, last_login_at
FROM `user`
WHERE id = ?;

-- name: UserByEmail :one
SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
       email, auth_source, created_at, updated_at, last_login_at
FROM `user`
WHERE email = ?;

-- name: CountUsers :one
SELECT COUNT(*)
FROM `user`
WHERE (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
  OR LOWER(username) LIKE sqlc.narg(search_pattern)
  OR LOWER(COALESCE(email, '')) LIKE sqlc.narg(search_pattern));

-- name: ListUsers :many
SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
       email, auth_source, created_at, updated_at, last_login_at
FROM `user`
WHERE (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
  OR LOWER(username) LIKE sqlc.narg(search_pattern)
  OR LOWER(COALESCE(email, '')) LIKE sqlc.narg(search_pattern))
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: UserRoles :many
SELECT role.code
FROM role
JOIN user_role ON user_role.role_id = role.id
WHERE user_role.user_id = ?
ORDER BY role.code;

-- name: UserPermissions :many
SELECT DISTINCT permission.code
FROM permission
JOIN role_permission ON role_permission.permission_id = permission.id
JOIN user_role ON user_role.role_id = role_permission.role_id
WHERE user_role.user_id = ?
ORDER BY permission.code;

-- name: UserRoleDetails :many
SELECT role.id, role.code, role.name, role.description, role.created_at, role.updated_at
FROM role
JOIN user_role ON user_role.role_id = role.id
WHERE user_role.user_id = ?
ORDER BY role.code;

-- name: UserRolesByUserIds :many
SELECT user_role.user_id, role.id, role.code, role.name, role.description, role.created_at, role.updated_at
FROM user_role
JOIN role ON role.id = user_role.role_id
WHERE user_role.user_id IN (sqlc.slice('user_ids'))
ORDER BY user_role.user_id, role.code;

-- name: CreateUser :exec
INSERT INTO `user` (id, username, password_hash, status, oauth_provider, oauth_provider_id, email, auth_source, created_at, updated_at, last_login_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateUser :exec
UPDATE `user`
SET username = ?, password_hash = ?, status = ?, email = ?, auth_source = ?,
    oauth_provider = ?, oauth_provider_id = ?, updated_at = ?
WHERE id = ?;

-- name: SetUserStatus :exec
UPDATE `user`
SET status = ?, updated_at = ?
WHERE id = ?;

-- name: MarkUserLoggedIn :exec
UPDATE `user`
SET last_login_at = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM `user`
WHERE id = ?;

-- name: DeleteUserRoles :exec
DELETE FROM user_role
WHERE user_id = ?;

-- name: InsertUserRole :exec
INSERT INTO user_role (user_id, role_id, created_at)
VALUES (?, ?, ?);

-- name: TouchUserUpdatedAt :exec
UPDATE `user`
SET updated_at = ?
WHERE id = ?;
