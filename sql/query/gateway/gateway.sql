-- name: GatewayConfigByApplication :one
SELECT application_id, traefik_component_name, rest_api_url, rest_ready_timeout_seconds,
       base_domain, default_entrypoint, tls_mode, acme_profile, acme_email,
       dns_api_token, created_at, updated_at
FROM gateway_config
WHERE application_id = ?;

-- name: GatewayVersionBindingsByApplication :many
SELECT application_id, profile, version_id
FROM gateway_acme_profile_version
WHERE application_id = ?
ORDER BY CASE profile
    WHEN 'base' THEN 0
    WHEN 'http' THEN 1
    WHEN 'dns' THEN 2
    WHEN 'http-dns' THEN 3
    ELSE 4
END;

-- name: GatewayRuntimeServiceCode :one
SELECT code
FROM service
WHERE application_id = ?
ORDER BY CASE WHEN status = 'running' THEN 0 ELSE 1 END,
         CASE WHEN instance_key = 'default' THEN 0 ELSE 1 END,
         updated_at DESC,
         id
LIMIT 1;

-- name: ListGatewayApplications :many
SELECT a.id, a.project_id, a.name, a.code, a.kind, a.created_at, a.updated_at
FROM application a
INNER JOIN gateway_config gc ON gc.application_id = a.id
WHERE a.project_id = ?
ORDER BY a.id DESC;

-- name: GatewayBindingByProjectID :one
SELECT gc.application_id
FROM environment e
INNER JOIN gateway_config gc ON gc.application_id = e.gateway_application_id
INNER JOIN application a ON a.id = gc.application_id AND a.project_id = e.project_id
WHERE e.project_id = ?;

-- name: InsertGatewayConfig :exec
INSERT INTO gateway_config (
  application_id, traefik_component_name, rest_api_url, rest_ready_timeout_seconds,
  base_domain, default_entrypoint, tls_mode, acme_profile, acme_email,
  dns_api_token, created_at, updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateGatewayConfig :exec
UPDATE gateway_config
SET traefik_component_name = ?, rest_api_url = ?, rest_ready_timeout_seconds = ?,
    base_domain = ?, default_entrypoint = ?, tls_mode = ?, acme_profile = ?, acme_email = ?,
    dns_api_token = ?, updated_at = ?
WHERE application_id = ?;

-- name: DeleteGatewayVersionBindings :exec
DELETE FROM gateway_acme_profile_version
WHERE application_id = ?;

-- name: InsertGatewayVersionBinding :exec
INSERT INTO gateway_acme_profile_version (application_id, profile, version_id)
VALUES (?, ?, ?);
