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
WHERE project_id = sqlc.arg(project_id)
ORDER BY id DESC;

-- name: ListEnabledRoutesByProjectId :many
SELECT id, project_id, name, protocol, domain, path_prefix, target_url, listen_port, service_id, component_name, endpoint_protocol, endpoint_container_port, enabled, https_enabled, cert_pem, cert_key, cert_type, acme_challenge, created_at, updated_at
FROM route
WHERE project_id = sqlc.arg(project_id)
  AND enabled = sqlc.arg(enabled)
ORDER BY name ASC, id ASC;

-- name: RouteById :one
SELECT id, project_id, name, protocol, domain, path_prefix, target_url, listen_port, service_id, component_name, endpoint_protocol, endpoint_container_port, enabled, https_enabled, cert_pem, cert_key, cert_type, acme_challenge, created_at, updated_at
FROM route
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: RouteByProjectIdAndDomain :one
SELECT id, project_id, name, protocol, domain, path_prefix, target_url, listen_port, service_id, component_name, endpoint_protocol, endpoint_container_port, enabled, https_enabled, cert_pem, cert_key, cert_type, acme_challenge, created_at, updated_at
FROM route
WHERE project_id = sqlc.arg(project_id)
  AND domain = sqlc.arg(domain);

-- name: CreateRoute :exec
INSERT INTO route (id, project_id, name, protocol, domain, path_prefix, target_url, listen_port, service_id, component_name, endpoint_protocol, endpoint_container_port, enabled, https_enabled, cert_pem, cert_key, cert_type, acme_challenge, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateRoute :exec
UPDATE route
SET name = sqlc.arg(name), protocol = sqlc.arg(protocol), domain = sqlc.arg(domain), path_prefix = sqlc.arg(path_prefix), target_url = sqlc.arg(target_url), listen_port = sqlc.arg(listen_port), service_id = sqlc.arg(service_id), component_name = sqlc.arg(component_name), endpoint_protocol = sqlc.arg(endpoint_protocol), endpoint_container_port = sqlc.arg(endpoint_container_port), enabled = sqlc.arg(enabled), https_enabled = sqlc.arg(https_enabled), cert_pem = sqlc.arg(cert_pem), cert_key = sqlc.arg(cert_key), cert_type = sqlc.arg(cert_type), acme_challenge = sqlc.arg(acme_challenge), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: DeleteRoute :exec
DELETE FROM route
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);
