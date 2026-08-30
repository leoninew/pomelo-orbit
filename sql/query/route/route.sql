-- name: CountRoutes :one
SELECT COUNT(*)
FROM route
WHERE project_id = sqlc.arg(project_id)
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR name LIKE sqlc.narg(search_pattern)
    OR domain LIKE sqlc.narg(search_pattern)
    OR target_url LIKE sqlc.narg(search_pattern));

-- name: ListRoutes :many
SELECT id, project_id, name, protocol, domain, path_prefix, target_url, listen_port, service_id, component_name, endpoint_protocol, endpoint_container_port, enabled, https_enabled, cert_pem, cert_key, cert_type, acme_challenge, created_at, updated_at
FROM route
WHERE project_id = sqlc.arg(project_id)
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR name LIKE sqlc.narg(search_pattern)
    OR domain LIKE sqlc.narg(search_pattern)
    OR target_url LIKE sqlc.narg(search_pattern))
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: ListAllRoutes :many
SELECT id, project_id, name, protocol, domain, path_prefix, target_url, listen_port, service_id, component_name, endpoint_protocol, endpoint_container_port, enabled, https_enabled, cert_pem, cert_key, cert_type, acme_challenge, created_at, updated_at
FROM route
WHERE project_id = ?
ORDER BY id DESC;

-- name: ListEnabledRoutes :many
SELECT id, project_id, name, protocol, domain, path_prefix, target_url, listen_port, service_id, component_name, endpoint_protocol, endpoint_container_port, enabled, https_enabled, cert_pem, cert_key, cert_type, acme_challenge, created_at, updated_at
FROM route
WHERE enabled = ?
ORDER BY name ASC, id ASC;

-- name: RouteByID :one
SELECT id, project_id, name, protocol, domain, path_prefix, target_url, listen_port, service_id, component_name, endpoint_protocol, endpoint_container_port, enabled, https_enabled, cert_pem, cert_key, cert_type, acme_challenge, created_at, updated_at
FROM route
WHERE id = ?;

-- name: RouteByDomain :one
SELECT id, project_id, name, protocol, domain, path_prefix, target_url, listen_port, service_id, component_name, endpoint_protocol, endpoint_container_port, enabled, https_enabled, cert_pem, cert_key, cert_type, acme_challenge, created_at, updated_at
FROM route
WHERE domain = ?;

-- name: CreateRoute :exec
INSERT INTO route (id, project_id, name, protocol, domain, path_prefix, target_url, listen_port, service_id, component_name, endpoint_protocol, endpoint_container_port, enabled, https_enabled, cert_pem, cert_key, cert_type, acme_challenge, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateRoute :exec
UPDATE route
SET name = ?, protocol = ?, domain = ?, path_prefix = ?, target_url = ?, listen_port = ?, service_id = ?, component_name = ?, endpoint_protocol = ?, endpoint_container_port = ?, enabled = ?, https_enabled = ?, cert_pem = ?, cert_key = ?, cert_type = ?, acme_challenge = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteRoute :exec
DELETE FROM route
WHERE id = ?;
