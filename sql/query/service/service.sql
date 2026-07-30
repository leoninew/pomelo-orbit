-- name: ListServicesByApplication :many
SELECT id, application_id, instance_key, version_id, runtime_config_json, status, created_at, updated_at
FROM service
WHERE application_id = ?
ORDER BY instance_key;

-- name: ServiceByID :one
SELECT id, application_id, instance_key, version_id, runtime_config_json, status, created_at, updated_at
FROM service
WHERE id = ?;

-- name: ServiceExposesByService :many
SELECT id, service_id, component_name, protocol, container_port, path_prefix, access, listen_port, created_at, updated_at
FROM service_expose
WHERE service_id = ?
ORDER BY component_name, protocol, container_port;

-- name: CountServiceExposesByVersionComponent :one
SELECT COUNT(*)
FROM service_expose se
INNER JOIN service s ON s.id = se.service_id
WHERE s.version_id = ?
  AND se.component_name = ?;

-- name: PublicTCPServiceExposesByListen :many
SELECT se.id, se.service_id, se.component_name, se.protocol, se.container_port, se.path_prefix, se.access, se.listen_port, se.created_at, se.updated_at
FROM service_expose se
WHERE se.access = 'public'
  AND se.protocol = 'tcp'
  AND COALESCE(se.listen_port, se.container_port) = ?
ORDER BY se.service_id, se.id;

-- name: LocalServiceExposesByListen :many
SELECT se.id, se.service_id, se.component_name, se.protocol, se.container_port, se.path_prefix, se.access, se.listen_port, se.created_at, se.updated_at
FROM service_expose se
WHERE se.access = 'local'
  AND COALESCE(se.listen_port, se.container_port) = ?
ORDER BY se.service_id, se.id;

-- name: ServiceByKey :one
SELECT id, application_id, instance_key, version_id, runtime_config_json, status, created_at, updated_at
FROM service
WHERE application_id = ? AND instance_key = ?;

-- name: ServiceIDByKey :one
SELECT id
FROM service
WHERE application_id = ? AND instance_key = ?;

-- name: InsertService :exec
INSERT INTO service (
  id, application_id, instance_key, version_id, runtime_config_json, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateService :exec
UPDATE service
SET version_id = ?, runtime_config_json = ?, status = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateServiceRuntimeConfig :exec
UPDATE service
SET runtime_config_json = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateServiceConfiguration :exec
UPDATE service
SET instance_key = ?, version_id = ?, runtime_config_json = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateServiceStatus :exec
UPDATE service
SET status = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateServiceAfterDeploy :exec
UPDATE service
SET status = ?, version_id = ?, updated_at = ?
WHERE id = ?;

-- name: DetachDeploymentServiceRefs :exec
UPDATE deployment
SET service_id = NULL
WHERE service_id = ?;

-- name: DeleteService :exec
DELETE FROM service
WHERE id = ?;

-- name: DeleteServiceExposes :exec
DELETE FROM service_expose
WHERE service_id = ?;

-- name: InsertServiceExpose :exec
INSERT INTO service_expose (
  id, service_id, component_name, protocol, container_port, path_prefix, access, listen_port, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CountServicesByProject :one
SELECT COUNT(*)
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
WHERE a.project_id = ?
  AND (? = '' OR s.application_id = ?)
  AND (? = '' OR s.status = ?)
  AND (? = '' OR a.name LIKE ? OR a.code LIKE ? OR s.instance_key LIKE ? OR v.label LIKE ?);

-- name: ListServicesByProject :many
SELECT s.id, s.application_id, s.instance_key, s.version_id, s.runtime_config_json, s.status, s.created_at, s.updated_at,
       a.name AS application_name, a.code AS application_code, a.kind AS application_kind,
       v.label AS version_label
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
WHERE a.project_id = ?
  AND (? = '' OR s.application_id = ?)
  AND (? = '' OR s.status = ?)
  AND (? = '' OR a.name LIKE ? OR a.code LIKE ? OR s.instance_key LIKE ? OR v.label LIKE ?)
ORDER BY s.updated_at DESC, a.name, s.instance_key
LIMIT ? OFFSET ?;

-- name: ServiceListItemByID :one
SELECT s.id, s.application_id, s.instance_key, s.version_id, s.runtime_config_json, s.status, s.created_at, s.updated_at,
       a.name AS application_name, a.code AS application_code, a.kind AS application_kind,
       v.label AS version_label
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
WHERE s.id = ?;
