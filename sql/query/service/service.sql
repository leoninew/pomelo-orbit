-- name: ListServicesByApplication :many
SELECT id, application_id, instance_key, code, version_id, status, created_at, updated_at
FROM service
WHERE application_id = ?
ORDER BY instance_key;

-- name: ServiceByID :one
SELECT id, application_id, instance_key, code, version_id, status, created_at, updated_at
FROM service
WHERE id = ?;

-- name: ServiceByKey :one
SELECT id, application_id, instance_key, code, version_id, status, created_at, updated_at
FROM service
WHERE application_id = ? AND instance_key = ?;

-- name: ServiceByCode :one
SELECT id, application_id, instance_key, code, version_id, status, created_at, updated_at
FROM service
WHERE code = ?;

-- name: ServiceIDByKey :one
SELECT id
FROM service
WHERE application_id = ? AND instance_key = ?;

-- name: InsertService :exec
INSERT INTO service (id, application_id, instance_key, code, version_id, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateService :exec
UPDATE service
SET version_id = ?, status = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateServiceConfiguration :exec
UPDATE service
SET instance_key = ?, version_id = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateServiceStatus :exec
UPDATE service
SET status = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateServiceAfterDeploy :exec
UPDATE service
SET status = ?, version_id = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteService :exec
DELETE FROM service
WHERE id = ?;

-- name: ServiceEnvByService :many
SELECT service_id, env_key, value
FROM service_env
WHERE service_id = ?
ORDER BY env_key;

-- name: DeleteServiceEnv :exec
DELETE FROM service_env
WHERE service_id = ?;

-- name: InsertServiceEnv :exec
INSERT INTO service_env (service_id, env_key, value)
VALUES (?, ?, ?);

-- name: TouchService :exec
UPDATE service
SET updated_at = ?
WHERE id = ?;

-- name: ServiceComponentsByService :many
SELECT id, service_id, source_version_component_id, component_name, status, created_at, updated_at
FROM service_component
WHERE service_id = ?
ORDER BY component_name;

-- name: ServiceComponentByID :one
SELECT id, service_id, source_version_component_id, component_name, status, created_at, updated_at
FROM service_component
WHERE id = ?;

-- name: DeleteServiceComponents :exec
DELETE FROM service_component
WHERE service_id = ?;

-- name: InsertServiceComponent :exec
INSERT INTO service_component (
  id, service_id, source_version_component_id, component_name, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateServiceComponentSource :exec
UPDATE service_component
SET source_version_component_id = ?, component_name = ?, updated_at = ?
WHERE id = ? AND service_id = ?;

-- name: ServiceComponentEnvByComponent :many
SELECT service_component_id, env_key, value, state
FROM service_component_env
WHERE service_component_id = ?
ORDER BY env_key;

-- name: DeleteServiceComponentEnv :exec
DELETE FROM service_component_env
WHERE service_component_id = ?;

-- name: InsertServiceComponentEnv :exec
INSERT INTO service_component_env (service_component_id, env_key, value, state)
VALUES (?, ?, ?, ?);

-- name: ServiceComponentMountsByComponent :many
SELECT service_component_id, target, source, state
FROM service_component_mount
WHERE service_component_id = ?
ORDER BY target;

-- name: DeleteServiceComponentMounts :exec
DELETE FROM service_component_mount
WHERE service_component_id = ?;

-- name: InsertServiceComponentMount :exec
INSERT INTO service_component_mount (id, service_component_id, target, source, state)
VALUES (?, ?, ?, ?, ?);

-- name: ServiceComponentResourceByComponent :one
SELECT service_component_id, limit_cpus, limit_memory, reservation_cpus, reservation_memory, state
FROM service_component_resource
WHERE service_component_id = ?;

-- name: DeleteServiceComponentResource :exec
DELETE FROM service_component_resource
WHERE service_component_id = ?;

-- name: InsertServiceComponentResource :exec
INSERT INTO service_component_resource (
  service_component_id, limit_cpus, limit_memory, reservation_cpus, reservation_memory, state
) VALUES (?, ?, ?, ?, ?, ?);

-- name: ServiceComponentEndpointsByComponent :many
SELECT service_component_id, name, mode, bind_address, listen_port, entrypoint, path_prefix, state
FROM service_component_endpoint
WHERE service_component_id = ?
ORDER BY name;

-- name: DeleteServiceComponentEndpoints :exec
DELETE FROM service_component_endpoint
WHERE service_component_id = ?;

-- name: InsertServiceComponentEndpoint :exec
INSERT INTO service_component_endpoint (
  service_component_id, name, mode, bind_address, listen_port, entrypoint, path_prefix, state
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: CountServicesByProject :one
SELECT COUNT(*)
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
WHERE a.project_id = CAST(sqlc.arg(project_id) AS TEXT)
  AND (
    CAST(sqlc.narg(application_id) AS TEXT) IS NULL
    OR s.application_id = CAST(sqlc.narg(application_id) AS TEXT)
  )
  AND (
    CAST(sqlc.narg(status) AS TEXT) IS NULL
    OR s.status = CAST(sqlc.narg(status) AS TEXT)
  )
  AND (
    CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
    OR a.name LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR a.code LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR s.code LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR s.instance_key LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR v.label LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
  );

-- name: ListServicesByProject :many
SELECT s.id, s.application_id, s.instance_key, s.code, s.version_id, s.status, s.created_at, s.updated_at,
       a.name AS application_name, a.code AS application_code, a.kind AS application_kind,
       v.label AS version_label
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
WHERE a.project_id = CAST(sqlc.arg(project_id) AS TEXT)
  AND (
    CAST(sqlc.narg(application_id) AS TEXT) IS NULL
    OR s.application_id = CAST(sqlc.narg(application_id) AS TEXT)
  )
  AND (
    CAST(sqlc.narg(status) AS TEXT) IS NULL
    OR s.status = CAST(sqlc.narg(status) AS TEXT)
  )
  AND (
    CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
    OR a.name LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR a.code LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR s.code LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR s.instance_key LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR v.label LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
  )
ORDER BY s.created_at DESC, a.name, s.instance_key
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: ServiceListItemByID :one
SELECT s.id, s.application_id, s.instance_key, s.code, s.version_id, s.status, s.created_at, s.updated_at,
       a.name AS application_name, a.code AS application_code, a.kind AS application_kind,
       v.label AS version_label
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
WHERE s.id = ?;
