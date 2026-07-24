-- name: GatewayConfigByApplication :one
SELECT application_id, rest_api_url, base_domain, image, default_entrypoint, tls_mode, created_at, updated_at
FROM gateway_config
WHERE application_id = ?;

-- name: ListGatewayApplications :many
SELECT id, project_id, name, code, kind, image_pull_policy, created_at, updated_at
FROM application
WHERE project_id = ? AND kind = ?
ORDER BY created_at DESC, id;

-- name: ListAllGatewayApplications :many
SELECT id, project_id, name, code, kind, image_pull_policy, created_at, updated_at
FROM application
WHERE kind = ?
ORDER BY created_at, id;

-- name: ResolveActiveGatewayConfig :one
SELECT gc.application_id, gc.rest_api_url, gc.base_domain, gc.image, gc.default_entrypoint, gc.tls_mode, gc.created_at, gc.updated_at
FROM gateway_config gc
INNER JOIN service s ON s.application_id = gc.application_id
INNER JOIN application a ON a.id = gc.application_id
WHERE a.kind = ? AND s.status IN (?, ?)
ORDER BY s.updated_at DESC, gc.application_id
LIMIT 1;

-- name: CountActiveGatewayServices :one
SELECT COUNT(*)
FROM service s
INNER JOIN application a ON a.id = s.application_id
WHERE a.kind = ? AND s.status IN (?, ?)
  AND (? = '' OR s.application_id <> ?);

-- name: InsertGatewayConfig :exec
INSERT INTO gateway_config (application_id, rest_api_url, base_domain, image, default_entrypoint, tls_mode, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateGatewayConfig :exec
UPDATE gateway_config
SET rest_api_url = ?, base_domain = ?, image = ?, default_entrypoint = ?, tls_mode = ?, updated_at = ?
WHERE application_id = ?;
