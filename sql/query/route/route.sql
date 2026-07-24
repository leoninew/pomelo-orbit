-- name: CountRoutes :one
SELECT COUNT(*)
FROM route
WHERE project_id = ?
  AND (? = '' OR name LIKE ? OR domain LIKE ? OR target_url LIKE ?);

-- name: ListRoutes :many
SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at
FROM route
WHERE project_id = ?
  AND (? = '' OR name LIKE ? OR domain LIKE ? OR target_url LIKE ?)
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: ListAllRoutes :many
SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at
FROM route
WHERE project_id = ?
ORDER BY id DESC;

-- name: ListEnabledRoutes :many
SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at
FROM route
WHERE enabled = ?
ORDER BY name ASC, id ASC;

-- name: RouteByID :one
SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at
FROM route
WHERE id = ?;

-- name: RouteByDomain :one
SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at
FROM route
WHERE domain = ?;

-- name: CreateRoute :exec
INSERT INTO route (id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateRoute :exec
UPDATE route
SET name = ?, domain = ?, path_prefix = ?, target_url = ?, enabled = ?, https_enabled = ?, cert_pem = ?, cert_key = ?, cert_type = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteRoute :exec
DELETE FROM route
WHERE id = ?;
