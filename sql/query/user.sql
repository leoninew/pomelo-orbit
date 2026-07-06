-- name: UserByUsername :one
SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
       email, auth_source, created_at, updated_at, last_login_at
FROM user
WHERE username = ?;

-- name: UserByID :one
SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
       email, auth_source, created_at, updated_at, last_login_at
FROM user
WHERE id = ?;

-- name: UserByEmail :one
SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
       email, auth_source, created_at, updated_at, last_login_at
FROM user
WHERE email = ?;

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

-- name: CreateUser :exec
INSERT INTO user (id, username, password_hash, status, oauth_provider, oauth_provider_id, email, auth_source, created_at, updated_at, last_login_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteUser :exec
DELETE FROM user
WHERE id = ?;

-- name: SaveLoginHistory :exec
INSERT INTO login_history (id, user_id, username, ip_address, user_agent, login_at, success)
VALUES (?, ?, ?, ?, ?, ?, ?);
