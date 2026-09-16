-- name: ListServicesByApplication :many
SELECT id, project_id, application_id, instance_key, code, version_id, status, created_at, updated_at
FROM service
WHERE application_id = sqlc.arg(application_id)
  AND project_id = sqlc.arg(project_id)
ORDER BY instance_key;

-- name: ServiceById :one
SELECT id, project_id, application_id, instance_key, code, version_id, status, created_at, updated_at
FROM service
WHERE service.id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: ServiceByKey :one
SELECT id, project_id, application_id, instance_key, code, version_id, status, created_at, updated_at
FROM service
WHERE application_id = sqlc.arg(application_id)
  AND instance_key = sqlc.arg(instance_key)
  AND project_id = sqlc.arg(project_id);

-- name: ServiceByProjectAndCode :one
SELECT id, project_id, application_id, instance_key, code, version_id, status, created_at, updated_at
FROM service
WHERE project_id = sqlc.arg(project_id)
  AND code = sqlc.arg(code);

-- name: ServiceIdByKey :one
SELECT id
FROM service
WHERE application_id = sqlc.arg(application_id)
  AND instance_key = sqlc.arg(instance_key)
  AND project_id = sqlc.arg(project_id);

-- name: InsertService :exec
INSERT INTO service (id, project_id, application_id, instance_key, code, version_id, status, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(project_id), sqlc.arg(application_id), sqlc.arg(instance_key), sqlc.arg(code), sqlc.arg(version_id), sqlc.arg(status), sqlc.arg(created_at), sqlc.arg(updated_at));

-- name: UpdateService :exec
UPDATE service
SET version_id = sqlc.arg(version_id), status = sqlc.arg(status), updated_at = sqlc.arg(updated_at)
WHERE service.id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: UpdateServiceConfiguration :exec
UPDATE service
SET instance_key = sqlc.arg(instance_key), version_id = sqlc.arg(version_id), updated_at = sqlc.arg(updated_at)
WHERE service.id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: UpdateServiceStatus :exec
UPDATE service
SET status = sqlc.arg(status), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: UpdateServiceAfterDeploy :exec
UPDATE service
SET status = sqlc.arg(status), version_id = sqlc.arg(version_id), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: DeleteService :exec
DELETE FROM service
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: ServiceEnvByService :many
SELECT service_id, env_key, value
FROM service_env
WHERE service_id = sqlc.arg(service_id)
  AND EXISTS (
    SELECT 1 FROM service
    WHERE service.id = service_env.service_id
      AND service.project_id = sqlc.arg(project_id)
  )
ORDER BY env_key;

-- name: DeleteServiceEnv :exec
DELETE FROM service_env
WHERE service_id = sqlc.arg(service_id)
  AND EXISTS (
    SELECT 1 FROM service
    WHERE service.id = service_env.service_id
      AND service.project_id = sqlc.arg(project_id)
  );

-- name: InsertServiceEnv :exec
INSERT INTO service_env (service_id, env_key, value)
SELECT sqlc.arg(service_id), sqlc.arg(env_key), sqlc.arg(value)
WHERE EXISTS (
  SELECT 1 FROM service
  WHERE service.id = sqlc.arg(service_id)
    AND service.project_id = sqlc.arg(project_id)
);

-- name: TouchService :exec
UPDATE service
SET updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: ServiceComponentsByService :many
SELECT id, service_id, source_version_component_id, component_name, entrypoint_json, command_json, pull_policy, restart_policy, status, created_at, updated_at
FROM service_component
WHERE service_id = sqlc.arg(service_id)
  AND EXISTS (
    SELECT 1 FROM service
    WHERE service.id = service_component.service_id
      AND service.project_id = sqlc.arg(project_id)
  )
ORDER BY component_name;

-- name: ServiceComponentById :one
SELECT id, service_id, source_version_component_id, component_name, entrypoint_json, command_json, pull_policy, restart_policy, status, created_at, updated_at
FROM service_component
WHERE service_component.id = sqlc.arg(id)
  AND EXISTS (
    SELECT 1 FROM service
    WHERE service.id = service_component.service_id
      AND service.project_id = sqlc.arg(project_id)
  );

-- name: DeleteServiceComponents :exec
DELETE FROM service_component
WHERE service_id = sqlc.arg(service_id)
  AND EXISTS (
    SELECT 1 FROM service
    WHERE service.id = service_component.service_id
      AND service.project_id = sqlc.arg(project_id)
  );

-- name: InsertServiceComponent :exec
INSERT INTO service_component (
  id, service_id, source_version_component_id, component_name, entrypoint_json, command_json, pull_policy, restart_policy, status, created_at, updated_at
) SELECT
  sqlc.arg(id), sqlc.arg(service_id), sqlc.arg(source_version_component_id), sqlc.arg(component_name), sqlc.arg(entrypoint_json), sqlc.arg(command_json), sqlc.arg(pull_policy), sqlc.arg(restart_policy), sqlc.arg(status), sqlc.arg(created_at), sqlc.arg(updated_at)
WHERE EXISTS (
  SELECT 1 FROM service
  WHERE service.id = sqlc.arg(service_id)
    AND service.project_id = sqlc.arg(project_id)
);

-- name: UpdateServiceComponentOverlayFields :exec
UPDATE service_component
SET entrypoint_json = sqlc.arg(entrypoint_json), command_json = sqlc.arg(command_json), pull_policy = sqlc.arg(pull_policy), restart_policy = sqlc.arg(restart_policy), updated_at = sqlc.arg(updated_at)
WHERE service_component.id = sqlc.arg(id)
  AND EXISTS (
    SELECT 1 FROM service
    WHERE service.id = service_component.service_id
      AND service.project_id = sqlc.arg(project_id)
  );

-- name: UpdateServiceComponentSource :exec
UPDATE service_component
SET source_version_component_id = sqlc.arg(source_version_component_id), component_name = sqlc.arg(component_name), updated_at = sqlc.arg(updated_at)
WHERE service_component.id = sqlc.arg(id)
  AND service_id = sqlc.arg(service_id)
  AND EXISTS (
    SELECT 1 FROM service
    WHERE service.id = service_component.service_id
      AND service.project_id = sqlc.arg(project_id)
  );

-- name: ServiceComponentEnvByComponent :many
SELECT service_component_id, env_key, value, state
FROM service_component_env
WHERE service_component_id = sqlc.arg(service_component_id)
  AND EXISTS (
    SELECT 1 FROM service_component
    JOIN service ON service.id = service_component.service_id
    WHERE service_component.id = service_component_env.service_component_id
      AND service.project_id = sqlc.arg(project_id)
  )
ORDER BY env_key;

-- name: DeleteServiceComponentEnv :exec
DELETE FROM service_component_env
WHERE service_component_id = sqlc.arg(service_component_id)
  AND EXISTS (
    SELECT 1 FROM service_component
    JOIN service ON service.id = service_component.service_id
    WHERE service_component.id = service_component_env.service_component_id
      AND service.project_id = sqlc.arg(project_id)
  );

-- name: InsertServiceComponentEnv :exec
INSERT INTO service_component_env (service_component_id, env_key, value, state)
SELECT sqlc.arg(service_component_id), sqlc.arg(env_key), sqlc.arg(value), sqlc.arg(state)
WHERE EXISTS (
  SELECT 1 FROM service_component
  JOIN service ON service.id = service_component.service_id
  WHERE service_component.id = sqlc.arg(service_component_id)
    AND service.project_id = sqlc.arg(project_id)
);

-- name: ServiceComponentMountsByComponent :many
SELECT service_component_id, target, source, source_is_host_path, state
FROM service_component_mount
WHERE service_component_id = sqlc.arg(service_component_id)
  AND EXISTS (
    SELECT 1 FROM service_component
    JOIN service ON service.id = service_component.service_id
    WHERE service_component.id = service_component_mount.service_component_id
      AND service.project_id = sqlc.arg(project_id)
  )
ORDER BY target;

-- name: DeleteServiceComponentMounts :exec
DELETE FROM service_component_mount
WHERE service_component_id = sqlc.arg(service_component_id)
  AND EXISTS (
    SELECT 1 FROM service_component
    JOIN service ON service.id = service_component.service_id
    WHERE service_component.id = service_component_mount.service_component_id
      AND service.project_id = sqlc.arg(project_id)
  );

-- name: InsertServiceComponentMount :exec
INSERT INTO service_component_mount (id, service_component_id, target, source, source_is_host_path, state)
SELECT sqlc.arg(id), sqlc.arg(service_component_id), sqlc.arg(target), sqlc.arg(source), sqlc.arg(source_is_host_path), sqlc.arg(state)
WHERE EXISTS (
  SELECT 1 FROM service_component
  JOIN service ON service.id = service_component.service_id
  WHERE service_component.id = sqlc.arg(service_component_id)
    AND service.project_id = sqlc.arg(project_id)
);

-- name: ServiceComponentResourceByComponent :one
SELECT service_component_id, limit_cpus, limit_memory, reservation_cpus, reservation_memory, state
FROM service_component_resource
WHERE service_component_id = sqlc.arg(service_component_id)
  AND EXISTS (
    SELECT 1 FROM service_component
    JOIN service ON service.id = service_component.service_id
    WHERE service_component.id = service_component_resource.service_component_id
      AND service.project_id = sqlc.arg(project_id)
  );

-- name: DeleteServiceComponentResource :exec
DELETE FROM service_component_resource
WHERE service_component_id = sqlc.arg(service_component_id)
  AND EXISTS (
    SELECT 1 FROM service_component
    JOIN service ON service.id = service_component.service_id
    WHERE service_component.id = service_component_resource.service_component_id
      AND service.project_id = sqlc.arg(project_id)
  );

-- name: InsertServiceComponentResource :exec
INSERT INTO service_component_resource (
  service_component_id, limit_cpus, limit_memory, reservation_cpus, reservation_memory, state
) SELECT
  sqlc.arg(service_component_id), sqlc.arg(limit_cpus), sqlc.arg(limit_memory), sqlc.arg(reservation_cpus), sqlc.arg(reservation_memory), sqlc.arg(state)
WHERE EXISTS (
  SELECT 1 FROM service_component
  JOIN service ON service.id = service_component.service_id
  WHERE service_component.id = sqlc.arg(service_component_id)
    AND service.project_id = sqlc.arg(project_id)
);

-- name: ServiceComponentEndpointsByComponent :many
SELECT id, service_component_id, protocol, container_port, mode, bind_address, listen_port, entrypoint, path_prefix, state
FROM service_component_endpoint
WHERE service_component_id = sqlc.arg(service_component_id)
  AND EXISTS (
    SELECT 1 FROM service_component
    JOIN service ON service.id = service_component.service_id
    WHERE service_component.id = service_component_endpoint.service_component_id
      AND service.project_id = sqlc.arg(project_id)
  )
ORDER BY protocol, container_port;

-- name: DeleteServiceComponentEndpoints :exec
DELETE FROM service_component_endpoint
WHERE service_component_id = sqlc.arg(service_component_id)
  AND EXISTS (
    SELECT 1 FROM service_component
    JOIN service ON service.id = service_component.service_id
    WHERE service_component.id = service_component_endpoint.service_component_id
      AND service.project_id = sqlc.arg(project_id)
  );

-- name: InsertServiceComponentEndpoint :exec
INSERT INTO service_component_endpoint (
  id, service_component_id, protocol, container_port, mode, bind_address, listen_port, entrypoint, path_prefix, state
) SELECT
  sqlc.arg(id), sqlc.arg(service_component_id), sqlc.arg(protocol), sqlc.arg(container_port), sqlc.arg(mode), sqlc.arg(bind_address), sqlc.arg(listen_port), sqlc.arg(entrypoint), sqlc.arg(path_prefix), sqlc.arg(state)
WHERE EXISTS (
  SELECT 1 FROM service_component
  JOIN service ON service.id = service_component.service_id
  WHERE service_component.id = sqlc.arg(service_component_id)
    AND service.project_id = sqlc.arg(project_id)
);

-- name: CountServicesByProject :one
SELECT COUNT(*)
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
WHERE s.project_id = sqlc.arg(project_id)
  AND (
    CAST(sqlc.narg(application_id) AS CHAR) IS NULL
    OR s.application_id = sqlc.narg(application_id)
  )
  AND (
    CAST(sqlc.narg(status) AS CHAR) IS NULL
    OR s.status = sqlc.narg(status)
  )
  AND (
    CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR a.name LIKE sqlc.narg(search_pattern)
    OR s.code LIKE sqlc.narg(search_pattern)
  );

-- name: ListServicesByProject :many
SELECT s.id, s.project_id, s.application_id, s.instance_key, s.code, s.version_id, s.status, s.created_at, s.updated_at,
       a.name AS application_name, a.code AS application_code, a.kind AS application_kind,
       v.label AS version_label
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
WHERE s.project_id = sqlc.arg(project_id)
  AND (
    CAST(sqlc.narg(application_id) AS CHAR) IS NULL
    OR s.application_id = sqlc.narg(application_id)
  )
  AND (
    CAST(sqlc.narg(status) AS CHAR) IS NULL
    OR s.status = sqlc.narg(status)
  )
  AND (
    CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR a.name LIKE sqlc.narg(search_pattern)
    OR s.code LIKE sqlc.narg(search_pattern)
  )
ORDER BY s.id DESC
LIMIT ? OFFSET ?;

-- name: ServiceListItemById :one
SELECT s.id, s.project_id, s.application_id, s.instance_key, s.code, s.version_id, s.status, s.created_at, s.updated_at,
       a.name AS application_name, a.code AS application_code, a.kind AS application_kind,
       v.label AS version_label
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN version v ON v.id = s.version_id
WHERE s.id = sqlc.arg(id)
  AND s.project_id = sqlc.arg(project_id);
